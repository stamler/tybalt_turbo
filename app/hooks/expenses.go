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
func cleanExpense(app core.App, expenseRecord *core.Record) error {
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

	purchaseOrderID := strings.TrimSpace(expenseRecord.GetString("purchase_order"))

	// No-PO expenses default to capital (no job) or project (with job).
	if purchaseOrderID == "" {
		hasJob := strings.TrimSpace(expenseRecord.GetString("job")) != ""
		expenseRecord.Set("kind", utilities.DefaultExpenditureKindIDForJob(hasJob))
	}

	// get the user's manager and set the approver field
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

	// Check that the approver (manager) is an active user
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

	// Ensure the approver (manager) has the `tapr` claim
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

	// If a purchase order is linked, force branch from the PO branch.
	if purchaseOrderID != "" {
		purchaseOrderRecord, err := app.FindRecordById("purchase_orders", purchaseOrderID)
		if err != nil || purchaseOrderRecord == nil {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "hook error when cleaning expense",
				Data: map[string]errs.CodeError{
					"purchase_order": {Code: "not_found", Message: "referenced purchase order not found"},
				},
			}
		}
		poBranchID := strings.TrimSpace(purchaseOrderRecord.GetString("branch"))
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

		// Mileage expenses do not have attachments, so we set the attachment
		// property to an empty string
		expenseRecord.Set("attachment", "")

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

		// Allowance expenses do not have attachments, so we set the attachment
		// property to an empty string
		expenseRecord.Set("attachment", "")
	}
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
	// if the expense record is submitted, return an error
	if expenseRecord.Get("submitted") == true {
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
	if err := cleanExpense(app, expenseRecord); err != nil {
		return err
	}

	// ensure the referenced division is active
	if err := ensureActiveDivision(app, expenseRecord.GetString("division"), "division"); err != nil {
		return err
	}

	// write the pay_period_ending property to the record. This is derived
	// exclusively from the date property.
	payPeriodEnding, ppEndErr := utilities.GeneratePayPeriodEnding(expenseRecord.GetString("date"))
	if ppEndErr != nil {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "hook error when processing expense",
			Data: map[string]errs.CodeError{
				"pay_period_ending": {
					Code:    "error_generating",
					Message: "error generating pay period ending",
				},
			},
		}
	}
	expenseRecord.Set("pay_period_ending", payPeriodEnding)

	// if the expense record has an attachment, calculate the sha256 hash of the
	// file and set the attachment_hash property on the record
	attachmentHash, hashErr := CalculateFileFieldHash(e, "attachment")
	if hashErr != nil {
		return hashErr
	}
	if attachmentHash != "" {
		// New file uploaded - check for duplicate before setting hash
		// Look for existing expense with the same attachment hash (excluding this record)
		existingExpense, _ := app.FindFirstRecordByFilter("expenses", "attachment_hash = {:hash} && id != {:id}", dbx.Params{
			"hash": attachmentHash,
			"id":   expenseRecord.Id,
		})
		if existingExpense != nil {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "duplicate attachment detected",
				Data: map[string]errs.CodeError{
					"attachment": {
						Code:    "duplicate_file",
						Message: "This file has already been uploaded to another expense",
					},
				},
			}
		}

		expenseRecord.Set("attachment_hash", attachmentHash)
	} else if expenseRecord.GetString("attachment") == "" {
		// Attachment was explicitly removed - clear the hash
		expenseRecord.Set("attachment_hash", "")
	}
	// Otherwise: no new file and attachment still exists - leave hash unchanged

	// if the expense record has a purchase_order, load it
	var poRecord *core.Record
	var err error
	purchaseOrder := expenseRecord.GetString("purchase_order")
	if purchaseOrder != "" {
		poRecord, err = app.FindRecordById("purchase_orders", purchaseOrder)
		if err != nil {
			return &errs.HookError{
				Status:  http.StatusInternalServerError,
				Message: "hook error when processing expense",
				Data: map[string]errs.CodeError{
					"purchase_order": {
						Code:    "not_found",
						Message: "purchase order not found",
					},
				},
			}
		}

		// Expense kind is inherited from its purchase order and is not user-editable.
		poHasJob := strings.TrimSpace(poRecord.GetString("job")) != ""
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

	// validate the expense record
	if err := validateExpense(app, expenseRecord, poRecord, existingExpensesTotal, hasPayablesAdminClaim); err != nil {
		return err
	}
	return nil
}
