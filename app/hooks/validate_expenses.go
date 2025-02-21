package hooks

import (
	"fmt"
	"net/http"
	"time"
	"tybalt/constants"
	"tybalt/utilities"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
)

// The validateExpense function is used to validate the expense record. It is
// called by ProcessExpense to ensure that the record is in a valid state before
// it is created or updated.
func validateExpense(expenseRecord *core.Record, poRecord *core.Record, existingExpensesTotal float64, byPassTotalLimit bool) error {

	var (
		poType           string = "Normal"
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
			return &HookError{
				Status:  http.StatusInternalServerError,
				Message: "error parsing purchase order date",
				Data: map[string]CodeError{
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
				return &HookError{
					Status:  http.StatusBadRequest,
					Message: "error parsing purchase order end date",
					Data: map[string]CodeError{
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
				return &HookError{
					Status:  http.StatusBadRequest,
					Message: "cumulative expenses exceed purchase order total",
					Data: map[string]CodeError{
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

	hasJob := expenseRecord.Get("job") != ""
	hasPurchaseOrder := expenseRecord.Get("purchase_order") != ""
	paymentType := expenseRecord.GetString("payment_type")
	isAllowance := paymentType == "Allowance"
	isPersonalReimbursement := paymentType == "PersonalReimbursement"
	isMileage := paymentType == "Mileage"
	isCorporateCreditCard := paymentType == "CorporateCreditCard"
	isFuelCard := paymentType == "FuelCard"

	// Throw an error if hasPurchaseOrder is true but poRecordProvided is false
	if hasPurchaseOrder && !poRecordProvided {
		return &HookError{
			Status:  http.StatusInternalServerError,
			Message: "an expense against a purchase_orders record cannot be validated without a corresponding purchase order record",
			Data: map[string]CodeError{
				"purchase_order": {
					Code:    "missing_purchase_order",
					Message: "an expense against a purchase_orders record cannot be validated without a corresponding purchase order record",
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
			validation.When(hasPurchaseOrder && (poType == "Normal" || poType == "Recurring"),
				validation.Max(totalLimit).Error(fmt.Sprintf("expense exceeds purchase order total of $%0.2f by more than %s", poTotal, excessErrorText)),
			),
			// TODO: Prevent a second expense from being created for a Normal PO i.e.
			// Two Expenses cannot exist for the same purchase_order if the type of
			// that purchase order is Normal. We could potentially do this by checking
			// if existingExpensesTotal is greater than 0 and if poType is
			// Normal then return an error in the "global" field.
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
	}.Filter()

	return validationsErrors

}
