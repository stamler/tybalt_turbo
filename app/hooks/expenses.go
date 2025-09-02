// this file implements cleaning and validation rules for the expenses collection

package hooks

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"strings"
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
	approver := profile.Get("manager")
	expenseRecord.Set("approver", approver)

	// Set branch from job if provided; otherwise from user's default branch
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

	// if the expense record has a payment_type of Mileage or Allowance, we
	// need to fetch the appropriate expense rate from the expense_rates
	// collection and set the rate and total fields on the expense record
	paymentType := expenseRecord.GetString("payment_type")

	switch paymentType {
	case "Mileage":
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

func calculateFileFieldHash(e *core.RecordRequestEvent, field string) (string, error) {
	// Get any files that have been uploaded for the field.
	files := e.Record.GetUnsavedFiles(field)

	// If the field is not present in the multipart form, or if it is present
	// but no actual files were uploaded for it (e.g., an empty file list).
	if len(files) == 0 {
		// No new file for this field in the current request.
		// Return empty string and no error, as there's nothing to hash.
		return "", nil
	}

	// If more than one file was uploaded for the field, this is an error,
	// as we expect only one file per field.
	if len(files) > 1 {
		return "", &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "hook error processing file for field: " + field,
			Data: map[string]errs.CodeError{
				field: {
					Code:    "too_many_files",
					Message: "too many files uploaded for field " + field,
				},
			},
		}
	}

	// At this point, len(fileHeaders) == 1. Get the first (and only) file header.
	fileReader := files[0].Reader

	// open the file
	file, err := fileReader.Open()
	if err != nil {
		return "", &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "hook error opening file for field: " + field,
			Data: map[string]errs.CodeError{
				field: {
					Code:    "error_opening_file",
					Message: "error opening file for field " + field,
				},
			},
		}
	}
	defer file.Close()

	// calculate the hash of the file
	log.Println("calculating hash for", field)

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "hook error when calculating expense attachment hash",
			Data: map[string]errs.CodeError{
				field: {
					Code:    "error_calculating_hash",
					Message: "error calculating hash",
				},
			},
		}
	}

	// return the hash as a hex string
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// The processExpense function is used to process the expense record. It is
// called by the hooks for the expenses collection to ensure that the record
// is in a valid state before it is created or updated.
func ProcessExpense(app core.App, e *core.RecordRequestEvent) error {
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
	attachmentHash, hashErr := calculateFileFieldHash(e, "attachment")
	if hashErr != nil {
		return hashErr
	}
	log.Println("attachmentHash", attachmentHash)
	expenseRecord.Set("attachment_hash", attachmentHash)

	// if the expense record has a purchase_order, load it
	var poRecord *core.Record = nil
	var err error = nil
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
	if err := validateExpense(expenseRecord, poRecord, existingExpensesTotal, hasPayablesAdminClaim); err != nil {
		return err
	}
	return nil
}
