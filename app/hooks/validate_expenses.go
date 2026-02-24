package hooks

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"tybalt/constants"
	"tybalt/errs"
	"tybalt/utilities"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
)

// The validateExpense function is used to validate the expense record. It is
// called by ProcessExpense to ensure that the record is in a valid state before
// it is created or updated.
func validateExpense(app core.App, expenseRecord *core.Record, poRecord *core.Record, existingExpensesTotal float64, byPassTotalLimit bool) error {

	var (
		poType           string = "One-Time"
		poRecordProvided bool   = false
		poTotal          float64
		poDate           time.Time
		poEndDate        time.Time
		totalLimit       float64
		excessErrorText  string = fmt.Sprintf("%0.2f%%", constants.MAX_PURCHASE_ORDER_EXCESS_PERCENT*100)
		parseErr         error
	)
	if poRecord != nil {
		poRecordProvided = true
		poTotal = poRecord.GetFloat("total")
		poType = poRecord.GetString("type")
		poDate, parseErr = time.Parse(time.DateOnly, poRecord.GetString("date"))
		if parseErr != nil {
			return &errs.HookError{
				Status:  http.StatusInternalServerError,
				Message: "error parsing purchase order date",
				Data: map[string]errs.CodeError{
					"purchase_order": {
						Code:    "error_parsing_date",
						Message: "error parsing purchase order date",
					},
				},
			}
		}
		if poType == "Recurring" {
			poEndDate, parseErr = time.Parse(time.DateOnly, poRecord.GetString("end_date"))
			if parseErr != nil {
				return &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "error parsing purchase order end date",
					Data: map[string]errs.CodeError{
						"purchase_order": {
							Code:    "error_parsing_end_date",
							Message: "error parsing purchase order end date",
						},
					},
				}
			}
		}

		// The maximum allowed total for all purchase_orders records is the lesser
		// of the value and percent limits.
		totalLimit = poTotal * (1.0 + constants.MAX_PURCHASE_ORDER_EXCESS_PERCENT) // initialize with percent limit
		if constants.MAX_PURCHASE_ORDER_EXCESS_VALUE < poTotal*constants.MAX_PURCHASE_ORDER_EXCESS_PERCENT {
			totalLimit = poTotal + constants.MAX_PURCHASE_ORDER_EXCESS_VALUE // use value limit instead
			excessErrorText = fmt.Sprintf("$%0.2f", constants.MAX_PURCHASE_ORDER_EXCESS_VALUE)
		}

		// For Cumulative POs, we check for overflow before other validations.
		// This is done here (rather than in the "total" validation) because:
		// 1. It allows early detection and a specific, actionable error for the child PO workflow
		// 2. We can provide rich error data (overflow amount, parent PO) that the validation framework doesn't support
		// 3. This represents a special case that can lead to child PO creation, not just rejection
		// 4. It makes testing clearer by distinctly separating this special case
		if poType == "Cumulative" {
			newTotal := existingExpensesTotal + expenseRecord.GetFloat("total")
			if newTotal > poTotal {
				overflowAmount := newTotal - poTotal
				// TODO: This returns a validation.Error per https://pocketbase.io/docs/go-routing/#error-response
				// However, the po number is not returned in the error data and neither is the poTotal and overflowAmount
				// properly delimited (it's just combined into a string). How can we return structured error data given
				// the constraints from the documentation? I've tried using SafeErrorItem but I'm having trouble importing
				// the router package, possibly related to versioning issues.
				return &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "cumulative expenses exceed purchase order total",
					Data: map[string]errs.CodeError{
						"total": {
							Code:    "cumulative_po_overflow",
							Message: "cumulative expenses exceed purchase order total",
							Data: map[string]any{
								"purchase_order":  poRecord.Id,
								"po_number":       poRecord.GetString("po_number"),
								"po_total":        poTotal,
								"overflow_amount": overflowAmount,
							},
						},
					},
				}
			}
			totalLimit -= existingExpensesTotal
		}
	}

	hasJob := expenseRecord.GetString("job") != ""
	hasPurchaseOrder := expenseRecord.GetString("purchase_order") != ""
	paymentType := expenseRecord.GetString("payment_type")
	isAllowance := paymentType == "Allowance"
	isPersonalReimbursement := paymentType == "PersonalReimbursement"
	isMileage := paymentType == "Mileage"
	isCorporateCreditCard := paymentType == "CorporateCreditCard"
	isFuelCard := paymentType == "FuelCard"

	// Require an attachment for all types except Allowance, Mileage, and PersonalReimbursement.
	// Accept either an already stored filename or a new uploaded file in the current multipart request.
	requiresAttachment := !(isAllowance || isMileage || isPersonalReimbursement)
	if requiresAttachment {
		hasStoredAttachment := expenseRecord.GetString("attachment") != ""
		hasUploadedAttachment := len(expenseRecord.GetUnsavedFiles("attachment")) > 0
		if !hasStoredAttachment && !hasUploadedAttachment {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "hook error when validating expense",
				Data: map[string]errs.CodeError{
					"attachment": {
						Code:    "required",
						Message: "attachment is required",
					},
				},
			}
		}
	}

	// Throw an error if hasPurchaseOrder is true but poRecordProvided is false
	if hasPurchaseOrder && !poRecordProvided {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "an expense against a purchase_orders record cannot be validated without a corresponding purchase order record",
			Data: map[string]errs.CodeError{
				"purchase_order": {
					Code:    "missing_purchase_order",
					Message: "an expense against a purchase_orders record cannot be validated without a corresponding purchase order record",
				},
			},
		}
	}

	// Expenses linked to a PO must retain the same job as the PO (including both empty).
	if hasPurchaseOrder {
		poJobID := strings.TrimSpace(poRecord.GetString("job"))
		expenseJobID := strings.TrimSpace(expenseRecord.GetString("job"))
		if expenseJobID != poJobID {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "hook error when validating expense",
				Data: map[string]errs.CodeError{
					"job": {
						Code:    "must_match_purchase_order",
						Message: "job must match purchase order job",
					},
				},
			}
		}
	}

	kindID := strings.TrimSpace(expenseRecord.GetString("kind"))
	if kindID == "" {
		// Backward compatibility for legacy records created before kind existed.
		// Keep create-time enforcement strict while allowing updates to old rows.
		if !expenseRecord.IsNew() {
			original := expenseRecord.Original()
			if original != nil && strings.TrimSpace(original.GetString("kind")) == "" {
				legacyHasJob := strings.TrimSpace(expenseRecord.GetString("job")) != ""
				kindID = utilities.DefaultExpenditureKindIDForJob(legacyHasJob)
				expenseRecord.Set("kind", kindID)
			}
		}
	}
	if kindID == "" {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "hook error when validating expense",
			Data: map[string]errs.CodeError{
				"kind": {
					Code:    "required",
					Message: "kind is required",
				},
			},
		}
	}
	if _, err := app.FindRecordById("expenditure_kinds", kindID); err != nil {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "hook error when validating expense",
			Data: map[string]errs.CodeError{
				"kind": {
					Code:    "not_found",
					Message: "invalid expenditure kind",
				},
			},
		}
	}

	validationsErrors := validation.Errors{
		"date": validation.Validate(
			expenseRecord.Get("date"),
			validation.Required.Error("date is required"),
			validation.Date("2006-01-02").Error("must be a valid date"),
			// the date should be on or after the "date" of the PO
			validation.When(hasPurchaseOrder,
				validation.By(utilities.DateStringLimit(poDate, false)),
			),
			// if the PO is Recurring, the date should be on or before the "end_date" of the PO
			validation.When(poType == "Recurring",
				validation.By(utilities.DateStringLimit(poEndDate, true)),
			),
		),
		"description": validation.Validate(
			expenseRecord.Get("description"),
			validation.When(!isAllowance,
				validation.Required.Error("required for non-allowance expenses"),
				validation.Length(5, 0).Error("must be at least 5 characters"),
			),
		),
		"vendor": validation.Validate(
			expenseRecord.Get("vendor"),
			validation.When(!isAllowance && !isPersonalReimbursement && !isMileage,
				validation.Required.Error("required for this expense type"),
				// validation.Length(2, 0).Error("must be at least 2 characters"),
			),
		),
		"cc_last_4_digits": validation.Validate(
			expenseRecord.Get("cc_last_4_digits"),
			validation.When(isCorporateCreditCard,
				validation.Required.Error("required for corporate credit card expenses"),
				validation.Length(4, 4).Error("must be 4 digits"),
			).Else(
				validation.Length(0, 0).Error("cc_last_4_digits is not applicable for non-corporate credit card expenses"),
			),
		),
		"total": validation.Validate(
			expenseRecord.GetFloat("total"),
			validation.Required.Error("must be greater than 0"),
			validation.Min(0.01).Error("must be greater than 0"),
			validation.When(!(byPassTotalLimit && paymentType == "OnAccount") && constants.LIMIT_NON_PO_AMOUNTS && !hasPurchaseOrder && !isMileage && !isFuelCard && !isPersonalReimbursement && !isAllowance,
				validation.Max(constants.NO_PO_EXPENSE_LIMIT).Exclusive().Error(fmt.Sprintf("a purchase order is required for expenses of $%0.2f or more", constants.NO_PO_EXPENSE_LIMIT)),
			),
			validation.When(hasPurchaseOrder && (poType == "One-Time" || poType == "Recurring"),
				validation.Max(totalLimit).Error(fmt.Sprintf("expense exceeds purchase order total of $%0.2f by more than %s", poTotal, excessErrorText)),
			),
			// TODO: Prevent a second expense from being created for a One-Time PO i.e.
			// Two Expenses cannot exist for the same purchase_order if the type of
			// that purchase order is One-Time. We could potentially do this by checking
			// if existingExpensesTotal is greater than 0 and if poType is
			// One-Time then return an error in the "global" field.
		),
		"distance": validation.Validate(
			expenseRecord.GetFloat("distance"),
			validation.When(isMileage,
				validation.Required.Error("required for mileage expenses"),
			),
		),
		"purchase_order": validation.Validate(
			expenseRecord.Get("purchase_order"),
			validation.When(hasJob && !isMileage && !isFuelCard && !isPersonalReimbursement && !isAllowance,
				validation.Required.Error("required for all expenses with a job"),
			),
		),
		"allowance_types": validation.Validate(
			expenseRecord.Get("allowance_types").([]string),
			validation.When(isAllowance,
				validation.Required.Error("required for allowance expenses"),
			),
		),
	}

	// Job-aware validation for expenses:
	// 1) job reference must exist and be Active
	// 2) when creating, or when job/division changes on update, ensure the
	//    selected division is allocated to the selected job.
	//
	// The second check is intentionally gated to avoid breaking updates to legacy
	// expenses where job/division was historically stored without strict
	// allocation enforcement.
	if hasJob {
		jobID := expenseRecord.GetString("job")
		jobRecord, err := app.FindRecordById("jobs", jobID)
		if err != nil || jobRecord == nil {
			validationsErrors["job"] = validation.NewError("invalid_reference", "invalid job reference")
		} else if jobRecord.GetString("status") != "Active" {
			validationsErrors["job"] = validation.NewError("not_active", "Job status must be Active")
		} else if shouldValidateJobDivisionAllocationOnRecord(app, expenseRecord) {
			if field, allocErr := validateDivisionAllocatedToJob(app, jobID, expenseRecord.GetString("division")); allocErr != nil {
				validationsErrors[field] = allocErr
			}
		}
	}

	return validationsErrors.Filter()

}
