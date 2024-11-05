// this file implements cleaning and validation rules for the expenses collection

package hooks

import (
	"net/http"
	"strings"
	"tybalt/utilities"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

// This feature flag is used to limit the total of expenses that don't have a
// corresponding purchase order.
const limitNonPoAmounts = true
const NO_PO_EXPENSE_LIMIT = 100.0

// The cleanExpense function is used to remove properties from the expense
// record that are not allowed to be set based on the value of the record's
// expense_type property. This is intended to reduce round trips to the database
// and to ensure that the record is in a valid state before it is created or
// updated. It is called by ProcessExpense to reduce the number of fields
// that need to be validated.
func cleanExpense(app core.App, expenseRecord *models.Record) error {

	// get the user's manager and set the approver field
	profile, err := app.Dao().FindFirstRecordByFilter("profiles", "uid = {:userId}", dbx.Params{
		"userId": expenseRecord.GetString("uid"),
	})
	if err != nil {
		return &HookError{
			Code:    http.StatusInternalServerError,
			Message: "hook error when cleaning expense",
			Data: map[string]CodeError{
				"uid": {
					Code:    "error_finding_profile",
					Message: "error finding profile for user",
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
			return &HookError{
				Code:    http.StatusInternalServerError,
				Message: "hook error when cleaning expense",
				Data: map[string]CodeError{
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
			return &HookError{
				Code:    http.StatusBadRequest,
				Message: "hook error when cleaning expense",
				Data: map[string]CodeError{
					"distance": {
						Code:    "not_an_integer",
						Message: "distance must be an integer for mileage expenses",
					},
				},
			}
		}

		totalMileageExpense, mileageErr := utilities.CalculateMileageTotal(app, expenseRecord, expenseRateRecord)
		if mileageErr != nil {
			return &HookError{
				Code:    http.StatusInternalServerError,
				Message: "hook error when cleaning expense",
				Data: map[string]CodeError{
					"total": {
						Code:    "error_calculating_mileage_total",
						Message: "error calculating mileage total",
					},
				},
			}
		}

		// update the properties appropriate for a mileage expense
		expenseRecord.Set("total", totalMileageExpense)
		expenseRecord.Set("vendor_name", "")

		// NOTE: during commit, we re-run the mileage calculation factoring in the
		// entire year's mileage total that is committed. This solves the issue of
		// out-of-order mileage expenses and acknowledges only committed expenses as
		// the source of truth.

	case "Allowance":
		expenseRateRecord, err := utilities.GetExpenseRateRecord(app, expenseRecord)
		if err != nil {
			return &HookError{
				Code:    http.StatusInternalServerError,
				Message: "hook error when cleaning expense",
				Data: map[string]CodeError{
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
		expenseRecord.Set("vendor_name", "")
	}
	return nil
}

// The processExpense function is used to process the expense record. It is
// called by the hooks for the expenses collection to ensure that the record
// is in a valid state before it is created or updated.
func ProcessExpense(app core.App, expenseRecord *models.Record, context echo.Context) error {

	// if the expense record is submitted, return an error
	if expenseRecord.Get("submitted") == true {
		return &HookError{
			Code:    http.StatusBadRequest,
			Message: "hook error when processing expense",
			Data: map[string]CodeError{
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
		return &HookError{
			Code:    http.StatusInternalServerError,
			Message: "hook error when processing expense",
			Data: map[string]CodeError{
				"pay_period_ending": {
					Code:    "error_generating",
					Message: "error generating pay period ending",
				},
			},
		}
	}
	expenseRecord.Set("pay_period_ending", payPeriodEnding)

	// if the expense record has a purchase_order, load it
	var poRecord *models.Record = nil
	var err error = nil
	purchaseOrder := expenseRecord.GetString("purchase_order")
	if purchaseOrder != "" {
		poRecord, err = app.Dao().FindRecordById("purchase_orders", purchaseOrder)
		if err != nil {
			return &HookError{
				Code:    http.StatusInternalServerError,
				Message: "hook error when processing expense",
				Data: map[string]CodeError{
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
		return &HookError{
			Code:    http.StatusBadRequest,
			Message: "hook error when processing expense",
			Data: map[string]CodeError{
				"purchase_order": {
					Code:    "not_active",
					Message: "purchase order must be active",
				},
			},
		}
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
			return &HookError{
				Code:    http.StatusInternalServerError,
				Message: "hook error when processing expense",
				Data: map[string]CodeError{
					"purchase_order": {
						Code:    "error_calculating",
						Message: "error calculating cumulative total",
					},
				},
			}
		}
	}
	// validate the expense record
	if err := validateExpense(expenseRecord, poRecord, existingExpensesTotal); err != nil {
		return err
	}
	return nil
}
