// this file implements cleaning and validation rules for the expenses collection

package hooks

import (
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

		// NOTE: during commit, we re-run the mileage calculation factoring in the
		// entire year's mileage total that is committed. This solves the issue of
		// out-of-order mileage expenses and acknowledges only committed expenses as
		// the source of truth.

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
	}
	return nil
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

	// TODO: throw an error if the caller is not allowed to create an expense
	// for this purchase order (define the rules for this)

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
