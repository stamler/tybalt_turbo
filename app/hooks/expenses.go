// this file implements cleaning and validation rules for the expenses collection

package hooks

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"tybalt/errs"
	"tybalt/utilities"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// The cleanExpense function is used to remove properties from the expense
// record that are not allowed to be set based on the value of the record's
// expense_type property. This is intended to reduce round trips to the database
// and to ensure that the record is in a valid state before it is created or
// updated. It is called by ProcessExpense to reduce the number of fields
// that need to be validated.
func cleanExpense(app core.App, expenseRecord *core.Record, poRecord *core.Record, creatorApprover bool) error {
	if expenseRecord.GetString("uid") == "" {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "hook error when cleaning expense",
			Data: map[string]errs.CodeError{
				"uid": {
					Code:    "not_provided",
					Message: "no uid provided in for expense record",
				},
			},
		}
	}

	purchaseOrderID := expenseRecord.GetString("purchase_order")
	homeCurrencyID := ""
	if homeCurrency, err := utilities.FindHomeCurrency(app); err == nil && homeCurrency != nil {
		homeCurrencyID = homeCurrency.Id
	}
	setHomeCurrency := func() {
		if homeCurrencyID != "" {
			expenseRecord.Set("currency", homeCurrencyID)
			return
		}
		expenseRecord.Set("currency", "")
	}

	// No-PO expenses default to capital (no job) or project (with job).
	if purchaseOrderID == "" {
		hasJob := expenseRecord.GetString("job") != ""
		expenseRecord.Set("kind", utilities.DefaultExpenditureKindIDForJob(hasJob))
	}

	if creatorApprover {
		creatorUID := expenseRecord.GetString("creator")
		if err := validateBookkeeperApprover(app, creatorUID); err != nil {
			return err
		}
		expenseRecord.Set("approver", creatorUID)
	} else if err := setManagerApprover(app, expenseRecord); err != nil {
		return err
	}

	// If a purchase order is linked, force branch from the PO branch.
	if purchaseOrderID != "" {
		if poRecord == nil {
			var err error
			poRecord, err = app.FindRecordById("purchase_orders", purchaseOrderID)
			if err != nil {
				return &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "hook error when cleaning expense",
					Data: map[string]errs.CodeError{
						"purchase_order": {Code: "not_found", Message: "referenced purchase order not found"},
					},
				}
			}
		}
		if poRecord == nil {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "hook error when cleaning expense",
				Data: map[string]errs.CodeError{
					"purchase_order": {Code: "not_found", Message: "referenced purchase order not found"},
				},
			}
		}
		poBranchID := poRecord.GetString("branch")
		if poBranchID == "" {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "hook error when cleaning expense",
				Data: map[string]errs.CodeError{
					"purchase_order": {Code: "missing_branch", Message: "referenced purchase order is missing a branch"},
				},
			}
		}
		expenseRecord.Set("branch", poBranchID)
		expenseRecord.Set("currency", poRecord.GetString("currency"))
	} else {
		// Set branch from job if provided; otherwise from user's default branch.
		jobId := expenseRecord.GetString("job")
		if jobId != "" {
			jobRecord, err := app.FindRecordById("jobs", jobId)
			if err != nil || jobRecord == nil {
				return &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "hook error when cleaning expense",
					Data: map[string]errs.CodeError{
						"job": {Code: "not_found", Message: "referenced job not found"},
					},
				}
			}
			branchId := jobRecord.GetString("branch")
			if branchId == "" {
				return &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "hook error when cleaning expense",
					Data: map[string]errs.CodeError{
						"job": {Code: "missing_branch", Message: "referenced job is missing a branch"},
					},
				}
			}
			expenseRecord.Set("branch", branchId)
		} else {
			uid := expenseRecord.GetString("uid")
			adminProfile, err := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": uid})
			if err != nil || adminProfile == nil {
				return &errs.HookError{
					Status:  http.StatusInternalServerError,
					Message: "hook error when cleaning expense",
					Data: map[string]errs.CodeError{
						"uid": {Code: "profile_lookup_error", Message: "error looking up admin profile"},
					},
				}
			}
			defaultBranchId := adminProfile.GetString("default_branch")
			if defaultBranchId == "" {
				return &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "hook error when cleaning expense",
					Data: map[string]errs.CodeError{
						"uid": {Code: "missing_default_branch", Message: "your admin_profiles record is missing a default_branch"},
					},
				}
			}
			expenseRecord.Set("branch", defaultBranchId)
		}
	}

	// if the expense record has a payment_type of Mileage or Allowance, we
	// need to fetch the appropriate expense rate from the expense_rates
	// collection and set the rate and total fields on the expense record
	paymentType := expenseRecord.GetString("payment_type")

	switch paymentType {
	case "Mileage":
		// Check personal vehicle insurance expiry for mileage expenses
		uid := expenseRecord.GetString("uid")
		adminProfile, apErr := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": uid})
		if apErr != nil || adminProfile == nil {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "hook error when cleaning expense",
				Data: map[string]errs.CodeError{
					"uid": {Code: "profile_lookup_error", Message: "error looking up admin profile for insurance check"},
				},
			}
		}

		insuranceExpiry := adminProfile.GetString("personal_vehicle_insurance_expiry")
		if insuranceExpiry == "" {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "personal vehicle insurance expiry not set",
				Data: map[string]errs.CodeError{
					"date": {
						Code:    "insurance_expiry_missing",
						Message: "cannot submit mileage expense: personal vehicle insurance expiry must be updated with a valid date",
					},
				},
			}
		}

		expiryDate, parseErr := time.Parse(time.DateOnly, insuranceExpiry)
		if parseErr != nil {
			return &errs.HookError{
				Status:  http.StatusInternalServerError,
				Message: "error parsing insurance expiry date",
			}
		}

		expenseDate, parseErr := time.Parse(time.DateOnly, expenseRecord.GetString("date"))
		if parseErr != nil {
			return &errs.HookError{
				Status:  http.StatusInternalServerError,
				Message: "error parsing expense date",
			}
		}

		if expenseDate.After(expiryDate) {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "personal vehicle insurance expired",
				Data: map[string]errs.CodeError{
					"date": {
						Code:    "insurance_expired",
						Message: fmt.Sprintf("cannot submit mileage expense: personal vehicle insurance expired on %s", insuranceExpiry),
					},
				},
			}
		}

		expenseRateRecord, err := utilities.GetExpenseRateRecord(app, expenseRecord)
		if err != nil {
			return &errs.HookError{
				Status:  http.StatusInternalServerError,
				Message: "hook error when cleaning expense",
				Data: map[string]errs.CodeError{
					"global": {
						Code:    "error_loading_expense_rate_record",
						Message: "error loading expense rate record",
					},
				},
			}
		}

		// if the paymentType is "Mileage", distance must be a positive integer
		// and we calculate the total by multiplying distance by the rate
		distance := expenseRecord.GetFloat("distance")
		// check if the distance is an integer
		if distance != float64(int(distance)) {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "hook error when cleaning expense",
				Data: map[string]errs.CodeError{
					"distance": {
						Code:    "not_an_integer",
						Message: "distance must be an integer for mileage expenses",
					},
				},
			}
		}

		totalMileageExpense, mileageErr := utilities.CalculateMileageTotal(app, expenseRecord, expenseRateRecord)
		if mileageErr != nil {
			return &errs.HookError{
				Status:  http.StatusInternalServerError,
				Message: "hook error when cleaning expense",
				Data: map[string]errs.CodeError{
					"total": {
						Code:    "error_calculating_mileage_total",
						Message: "error calculating mileage total",
					},
				},
			}
		}

		// update the properties appropriate for a mileage expense
		expenseRecord.Set("total", totalMileageExpense)
		expenseRecord.Set("vendor", "")

		// NOTE: during commit, we re-run the mileage calculation factoring in the
		// entire year's mileage total that is committed. This solves the issue of
		// out-of-order mileage expenses and acknowledges only committed expenses as
		// the source of truth.

		// NOTE: As of 2025-05-23, we are not using the mileage calculation for
		// payroll. See payroll_expenses.sql for the query that is used to calculate
		// the mileage total for payroll.
		// TODO: remove this note when we start using the mileage calculation for
		// payroll and figure out how to calculate the total for imported expenses
		// that don't have the total property set.

	case "PersonalReimbursement":
		uid := expenseRecord.GetString("uid")
		adminProfile, apErr := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": uid})
		if apErr != nil || adminProfile == nil {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "hook error when cleaning expense",
				Data: map[string]errs.CodeError{
					"uid": {Code: "profile_lookup_error", Message: "error looking up admin profile for personal reimbursement check"},
				},
			}
		}

		if !adminProfile.GetBool("allow_personal_reimbursement") {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "personal reimbursement not allowed",
				Data: map[string]errs.CodeError{
					"payment_type": {
						Code:    "personal_reimbursement_not_allowed",
						Message: "cannot submit personal reimbursement expense: personal reimbursement is not enabled for your profile",
					},
				},
			}
		}

	case "Allowance":
		expenseRateRecord, err := utilities.GetExpenseRateRecord(app, expenseRecord)
		if err != nil {
			return &errs.HookError{
				Status:  http.StatusInternalServerError,
				Message: "hook error when cleaning expense",
				Data: map[string]errs.CodeError{
					"global": {
						Code:    "error_loading_expense_rate_record",
						Message: "error loading expense rate record",
					},
				},
			}
		}

		// If the paymentType is "Allowance", the allowance_type property of the
		// expenseRecord will have one or more of the following values: Breakfast,
		// Lunch, Dinner, Lodging. It is a JSON array of strings. We use this to
		// determine which of the rates to sum up to get the total allowance for the
		// expense.
		allowanceTypes := expenseRecord.Get("allowance_types").([]string)

		// sum up the rates for the allowance types that are present
		total := 0.0
		for _, allowanceType := range allowanceTypes {
			total += expenseRateRecord.GetFloat(strings.ToLower(allowanceType))
		}

		// build a description of the expense by joining the allowance types
		// with commas
		allowanceDescription := "Allowance for "
		allowanceDescription += strings.Join(allowanceTypes, ", ")

		// update the properties appropriate for an allowance expense
		expenseRecord.Set("total", total)
		expenseRecord.Set("description", allowanceDescription)
		expenseRecord.Set("vendor", "")

		// NOTE: As of 2025-05-23, we are not using the allowance calculation for
		// payroll. See payroll_expenses.sql for the query that is used to calculate
		// the allowance total for payroll.
		// TODO: remove this note when we start using the allowance calculation for
		// payroll and figure out how to calculate the total for imported expenses
		// that don't have the total property set.

	}

	if purchaseOrderID == "" {
		switch paymentType {
		case "Allowance", "FuelCard", "Mileage", "PersonalReimbursement":
			setHomeCurrency()
		}
	}

	if expenseRecord.GetString("currency") == "" && homeCurrencyID != "" {
		expenseRecord.Set("currency", homeCurrencyID)
	}

	currencyInfo, err := utilities.ResolveCurrencyInfo(app, expenseRecord.GetString("currency"))
	if err != nil {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "hook error when cleaning expense",
			Data: map[string]errs.CodeError{
				"currency": {Code: "not_found", Message: "referenced currency not found"},
			},
		}
	}
	if err := utilities.RequirePositiveForeignCurrencyRate(currencyInfo); err != nil {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "hook error when cleaning expense",
			Data: map[string]errs.CodeError{
				"currency": {
					Code:    "missing_rate",
					Message: "selected foreign currency is missing an exchange rate",
				},
			},
		}
	}

	if utilities.IsHomeCurrencyInfo(currencyInfo) {
		expenseRecord.Set("settled_total", expenseRecord.GetFloat("total"))
		expenseRecord.Set("settler", "")
		expenseRecord.Set("settled", "")
	} else if paymentType == "Expense" {
		expenseRecord.Set("settler", "")
		expenseRecord.Set("settled", "")
	} else {
		expenseRecord.Set("settled_total", 0)
		expenseRecord.Set("settler", "")
		expenseRecord.Set("settled", "")
	}
	return nil
}

func setManagerApprover(app core.App, expenseRecord *core.Record) error {
	profile, err := app.FindFirstRecordByFilter("profiles", "uid = {:userId}", dbx.Params{
		"userId": expenseRecord.GetString("uid"),
	})
	if profile == nil {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "hook error when cleaning expense",
			Data: map[string]errs.CodeError{
				"uid": {
					Code:    "no_profile_found_for_uid",
					Message: "no profile found for uid",
				},
			},
		}
	}
	if err != nil {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "hook error when cleaning expense",
			Data: map[string]errs.CodeError{
				"uid": {
					Code:    "error_during_profile_lookup",
					Message: "error during profile lookup",
				},
			},
		}
	}
	approverUID := profile.GetString("manager")

	active, activeErr := utilities.IsUserActive(app, approverUID)
	if activeErr != nil {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "failed to check approver active status",
		}
	}
	if !active {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "your manager must be an active user to create expenses",
			Data: map[string]errs.CodeError{
				"approver": {Code: "approver_not_active", Message: "the approver (your manager) is not an active user"},
			},
		}
	}

	hasTapr, taprErr := utilities.HasClaimByUserID(app, approverUID, "tapr")
	if taprErr != nil {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "hook error when cleaning expense",
			Data: map[string]errs.CodeError{
				"approver": {Code: "error_checking_claim", Message: "error checking approver claim"},
			},
		}
	}
	if !hasTapr {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "hook error when cleaning expense",
			Data: map[string]errs.CodeError{
				"approver": {Code: "unqualified_approver", Message: "the approver must have the tapr claim"},
			},
		}
	}
	expenseRecord.Set("approver", approverUID)
	return nil
}

func validateBookkeeperApprover(app core.App, creatorUID string) error {
	if creatorUID == "" {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "hook error when cleaning expense",
			Data: map[string]errs.CodeError{
				"creator": {Code: "not_provided", Message: "creator is required"},
			},
		}
	}

	active, activeErr := utilities.IsUserActive(app, creatorUID)
	if activeErr != nil {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "failed to check creator active status",
		}
	}
	if !active {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "bookkeeper approver must be an active user",
			Data: map[string]errs.CodeError{
				"approver": {Code: "approver_not_active", Message: "the bookkeeper approver is not an active user"},
			},
		}
	}

	hasClaim, claimErr := utilities.HasClaimByUserID(app, creatorUID, "book_keeper")
	if claimErr != nil {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "hook error when cleaning expense",
			Data: map[string]errs.CodeError{
				"approver": {Code: "error_checking_claim", Message: "error checking bookkeeper claim"},
			},
		}
	}
	if !hasClaim {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "hook error when cleaning expense",
			Data: map[string]errs.CodeError{
				"approver": {Code: "unqualified_approver", Message: "the approver must have the book_keeper claim"},
			},
		}
	}
	return nil
}

func validateBookkeeperOnBehalfCreate(app core.App, expenseRecord *core.Record, authID string, poRecord *core.Record) error {
	if poRecord == nil {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "hook error when validating bookkeeper expense",
			Data: map[string]errs.CodeError{
				"purchase_order": {Code: "required", Message: "purchase order is required for bookkeeper-created expenses"},
			},
		}
	}

	hasClaim, claimErr := utilities.HasClaimByUserID(app, authID, "book_keeper")
	if claimErr != nil {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "error checking bookkeeper claim",
			Data: map[string]errs.CodeError{
				"creator": {Code: "error_checking_claim", Message: "error checking bookkeeper claim"},
			},
		}
	}
	if !hasClaim {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "hook error when validating uid",
			Data: map[string]errs.CodeError{
				"uid": {Code: "value_mismatch", Message: "uid must be equal to the authenticated user's id"},
			},
		}
	}

	if poRecord.GetString("status") != "Active" {
		return validation.Errors{
			"purchase_order": validation.NewError("not_active", "purchase order must be active"),
		}.Filter()
	}
	if strings.TrimSpace(poRecord.GetString("po_number")) == "" {
		return validation.Errors{
			"purchase_order": validation.NewError("missing_po_number", "purchase order must have a po number"),
		}.Filter()
	}

	poUID := poRecord.GetString("uid")
	if poUID == "" {
		return validation.Errors{
			"purchase_order": validation.NewError("missing_uid", "purchase order is missing an owner"),
		}.Filter()
	}
	if poUID == authID {
		return validation.Errors{
			"uid": validation.NewError("same_user_bookkeeper_not_applicable", "same-user purchase orders must use the regular expense flow"),
		}.Filter()
	}
	if expenseRecord.GetString("uid") != poUID {
		return validation.Errors{
			"uid": validation.NewError("must_match_purchase_order_owner", "uid must match the purchase order owner"),
		}.Filter()
	}

	paymentType := expenseRecord.GetString("payment_type")
	if paymentType != "OnAccount" && paymentType != "CorporateCreditCard" {
		return validation.Errors{
			"payment_type": validation.NewError("not_allowed_for_bookkeeper", "bookkeeper-created expenses must be OnAccount or CorporateCreditCard"),
		}.Filter()
	}

	expenseRecord.Set("uid", poUID)
	return nil
}

// The processExpense function is used to process the expense record. It is
// called by the hooks for the expenses collection to ensure that the record
// is in a valid state before it is created or updated.
func ProcessExpense(app core.App, e *core.RecordRequestEvent) error {
	if err := checkExpensesEditing(app); err != nil {
		return err
	}

	expenseRecord := e.Record
	authRecord := e.Auth
	if authRecord == nil || authRecord.Id == "" {
		return &errs.HookError{
			Status:  http.StatusUnauthorized,
			Message: "hook error when validating uid",
			Data: map[string]errs.CodeError{
				"uid": {
					Code:    "authentication_required",
					Message: "authentication is required",
				},
			},
		}
	}

	// if the expense record has a purchase_order, load it before ownership and
	// approver decisions because the bookkeeper path is defined by the linked PO.
	var poRecord *core.Record
	var err error
	purchaseOrder := expenseRecord.GetString("purchase_order")
	if purchaseOrder != "" {
		poRecord, err = app.FindRecordById("purchase_orders", purchaseOrder)
		if err != nil {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "hook error when processing expense",
				Data: map[string]errs.CodeError{
					"purchase_order": {
						Code:    "not_found",
						Message: "purchase order not found",
					},
				},
			}
		}
	}

	authID := authRecord.Id
	requestUID := expenseRecord.GetString("uid")
	creatorApprover := false
	if expenseRecord.IsNew() {
		expenseRecord.Set("creator", authID)
		if requestUID == authID {
			// Regular creates, including bookkeepers entering against their own PO,
			// continue through manager-derived approval.
		} else {
			if err := validateBookkeeperOnBehalfCreate(app, expenseRecord, authID, poRecord); err != nil {
				return err
			}
			creatorApprover = true
		}
	} else {
		original := expenseRecord.Original()
		if original == nil {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "hook error when validating uid",
				Data: map[string]errs.CodeError{
					"uid": {
						Code:    "missing_original_record",
						Message: "cannot validate ownership for this expense update",
					},
				},
			}
		}

		originalUID := original.GetString("uid")
		originalCreator := original.GetString("creator")
		if original.GetString("creator") != authID {
			return &errs.HookError{
				Status:  http.StatusForbidden,
				Message: "hook error when validating uid",
				Data: map[string]errs.CodeError{
					"uid": {
						Code:    "not_owner",
						Message: "only the creator can edit this expense",
					},
				},
			}
		}
		if requestUID != originalUID {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "hook error when validating uid",
				Data: map[string]errs.CodeError{
					"uid": {
						Code:    "immutable_field",
						Message: "uid cannot be changed",
					},
				},
			}
		}
		if expenseRecord.GetString("creator") != originalCreator {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "hook error when validating creator",
				Data: map[string]errs.CodeError{
					"creator": {
						Code:    "immutable_field",
						Message: "creator cannot be changed",
					},
				},
			}
		}
		creatorApprover = originalCreator != "" && originalCreator != originalUID
		if creatorApprover {
			if err := validateBookkeeperOnBehalfCreate(app, expenseRecord, authID, poRecord); err != nil {
				return err
			}
		}

		if original.GetBool("submitted") {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "hook error when processing expense",
				Data: map[string]errs.CodeError{
					"submitted": {
						Code:    "is_submitted",
						Message: "cannot edit submitted expense",
					},
				},
			}
		}
	}

	// if the expense record is submitted, return an error
	if expenseRecord.GetBool("submitted") {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "hook error when processing expense",
			Data: map[string]errs.CodeError{
				"submitted": {
					Code:    "is_submitted",
					Message: "cannot edit submitted expense",
				},
			},
		}
	}

	// clean the expense record
	if err := cleanExpense(app, expenseRecord, poRecord, creatorApprover); err != nil {
		return err
	}

	// ensure the referenced division is active
	if err := EnsureActiveDivision(app, expenseRecord.GetString("division"), "division"); err != nil {
		return err
	}

	if err := utilities.EnsureUserCanUseBranch(app, expenseRecord.GetString("branch"), authRecord.Id, "branch"); err != nil {
		return err
	}

	// Keep "attachment" as the expense-write request field even though expense
	// files now live on expense_documents. It is intentionally an input name, not
	// a storage location:
	//
	//   - Users are editing an expense, so the form should continue to have one
	//     obvious "Attachment" control.
	//   - Letting the client upload directly to expense_documents would expose a
	//     document picker/write path that could bypass expense ownership,
	//     PO-backed duplicate reuse, source_expense visibility, missing-receipt
	//     handling, and rollback rules.
	//   - The server needs one transaction-shaped workflow: validate the expense,
	//     create or reuse the document row, link attachment_document, and clean up
	//     the document upload if the final expense save fails.
	//
	// The custom /api/expenses route also stashes uploaded files under the raw
	// "attachment:unsaved" key so this hook keeps working after the physical
	// expenses.attachment file column is removed.
	NormalizePendingFileNames(expenseRecord, "attachment")
	requestInfo, requestInfoErr := e.RequestInfo()
	if requestInfoErr != nil {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "error reading request info",
			Data: map[string]errs.CodeError{
				"global": {Code: "request_info_error", Message: "error reading request info"},
			},
		}
	}
	if rawSourceExpenseID, ok := requestInfo.Body["source_expense"]; ok {
		if sourceExpenseID := strings.TrimSpace(fmt.Sprint(rawSourceExpenseID)); sourceExpenseID != "" {
			expenseRecord.Set("source_expense", sourceExpenseID)
		}
	}
	explicitAttachmentRemoval := false
	if rawAttachment, ok := requestInfo.Body["attachment"]; ok && strings.TrimSpace(fmt.Sprint(rawAttachment)) == "" {
		explicitAttachmentRemoval = len(expenseRecord.GetUnsavedFiles("attachment")) == 0 && expenseRecord.GetString("source_expense") == ""
		if explicitAttachmentRemoval {
			expenseRecord.Set("attachment_document", "")
		}
	}
	originalAttachmentDocument := ""
	originalAttachmentMissingReason := ""
	if !expenseRecord.IsNew() {
		if original := expenseRecord.Original(); original != nil {
			originalAttachmentDocument = original.GetString("attachment_document")
			originalAttachmentMissingReason = original.GetString("attachment_missing_reason")
		}
	}

	// If the request carried a file in the virtual "attachment" input, hash it
	// before validation finishes. The hash is used only to create/reuse the
	// expense_documents row; it must not be treated as a value for the old
	// expenses.attachment_hash column.
	attachmentHash, hashErr := CalculateFileFieldHash(e, "attachment")
	if hashErr != nil {
		return hashErr
	}

	if poRecord != nil {
		// Expense kind is inherited from its purchase order and is not user-editable.
		poHasJob := poRecord.GetString("job") != ""
		inheritedKind := utilities.NormalizeExpenditureKindID(poRecord.GetString("kind"), poHasJob)
		expenseRecord.Set("kind", inheritedKind)
	}

	// if the purchaseOrder has a status that is not "Active", return an error
	if poRecord != nil && poRecord.GetString("status") != "Active" {
		return validation.Errors{
			"purchase_order": validation.NewError("not_active", "purchase order must be active"),
		}.Filter()
	}

	// A decision was made via Teams meeting on August 26, 2025 to allow anybody to
	// submit an expense against any active purchase order provided the other rules
	// are met. If this changes in the future, we will need to add checks here to
	// ensure that the caller is allowed to create an expense for the specified
	// purchase order.

	// if the purchase_order record's type property is "Cumulative", get the sum
	// of the total property of all of the expenses associated with this purchase
	// order and pass the sum to the validateExpense function. Do this using SQL
	// via DB().NewQuery() since we just want the sum.
	existingExpensesTotal := 0.0
	if poRecord != nil && poRecord.GetString("type") == "Cumulative" {
		existingExpensesTotal, err = utilities.CumulativeTotalExpensesForPurchaseOrder(app, poRecord, false)
		if err != nil {
			return &errs.HookError{
				Status:  http.StatusInternalServerError,
				Message: "hook error when processing expense",
				Data: map[string]errs.CodeError{
					"purchase_order": {
						Code:    "error_calculating",
						Message: "error calculating cumulative total",
					},
				},
			}
		}
	}

	// Check if user has payables_admin claim
	hasPayablesAdminClaim, err := utilities.HasClaim(app, e.Auth, "payables_admin")
	if err != nil {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "error checking claim",
			Data: map[string]errs.CodeError{
				"global": {
					Code:    "error_checking_claim",
					Message: "error checking payables_admin claim",
				},
			},
		}
	}

	// attachment_document is server-managed. Restore it before validation so a
	// client-supplied relation cannot satisfy attachment-required checks.
	if expenseRecord.IsNew() || explicitAttachmentRemoval {
		expenseRecord.Set("attachment_document", "")
		expenseRecord.Set("attachment_missing_reason", "")
	} else {
		expenseRecord.Set("attachment_document", originalAttachmentDocument)
		expenseRecord.Set("attachment_missing_reason", originalAttachmentMissingReason)
	}

	// validate the expense record
	if err := validateExpense(app, expenseRecord, poRecord, existingExpensesTotal, hasPayablesAdminClaim); err != nil {
		return err
	}

	hasBookKeeperClaim, err := utilities.HasClaim(app, e.Auth, "book_keeper")
	if err != nil {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "error checking claim",
			Data: map[string]errs.CodeError{
				"global": {
					Code:    "error_checking_claim",
					Message: "error checking book_keeper claim",
				},
			},
		}
	}

	hasAttachmentIntent := attachmentHash != "" || expenseRecord.GetString("source_expense") != ""
	attachmentlessType := expensePaymentTypeSkipsAttachment(expenseRecord.GetString("payment_type"))

	clearExpenseDocumentForAttachmentlessType(expenseRecord)
	if attachmentlessType {
		// Attachmentless expense types should never keep or accept a document relation.
		expenseRecord.Set("attachment_missing_reason", "")
	} else if explicitAttachmentRemoval {
		expenseRecord.Set("attachment_document", "")
		expenseRecord.Set("attachment", "")
		expenseRecord.Set("attachment_hash", "")
		expenseRecord.Set("attachment_missing_reason", "")
	} else if hasAttachmentIntent {
		documentID, err := resolveExpenseDocumentForSave(app, e, attachmentHash, hasBookKeeperClaim, poRecord != nil)
		if err != nil {
			return err
		}
		applyResolvedExpenseDocument(expenseRecord, documentID)
		expenseRecord.Set("attachment_missing_reason", "")
	} else if !expenseRecord.IsNew() {
		expenseRecord.Set("attachment_document", originalAttachmentDocument)
	} else {
		expenseRecord.Set("attachment_document", "")
	}

	return nil
}
