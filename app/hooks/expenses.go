// this file implements cleaning and validation rules for the expenses collection

package hooks

import (
	"errors"
	"log"
	"strings"
	"time"
	"tybalt/utilities"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
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
func cleanExpense(app *pocketbase.PocketBase, expenseRecord *models.Record) error {

	// get the user's manager and set the approver field
	profile, err := app.Dao().FindFirstRecordByFilter("profiles", "uid = {:userId}", dbx.Params{
		"userId": expenseRecord.GetString("uid"),
	})
	if err != nil {
		return err
	}
	approver := profile.Get("manager")
	expenseRecord.Set("approver", approver)

	// if the expense record has a payment_type of Mileage or Allowance, we
	// need to fetch the appropriate expense rate from the expense_rates
	// collection and set the rate and total fields on the expense record
	paymentType := expenseRecord.GetString("payment_type")

	switch paymentType {
	case "Mileage":
		expenseDate := expenseRecord.GetString("date")
		expenseRateRecord, err := getExpenseRateRecord(app, expenseRecord)
		if err != nil {
			return err
		}

		// if the paymentType is "Mileage", distance must be a positive integer
		// and we calculate the total by multiplying distance by the rate
		distance := expenseRecord.GetFloat("distance")
		// check if the distance is an integer
		if distance != float64(int(distance)) {
			return errors.New("distance must be an integer for mileage expenses")
		}

		startDate, err := utilities.GetAnnualPayrollPeriodStartDate(app, expenseDate)
		if err != nil {
			return err
		}

		totalMileageExpense, mileageErr := utilities.CalculateMileageTotal(app, int(distance), startDate, expenseDate, expenseRateRecord)
		if mileageErr != nil {
			return mileageErr
		}

		// update the properties appropriate for a mileage expense
		expenseRecord.Set("total", totalMileageExpense)
		expenseRecord.Set("vendor_name", "")

		// TODO: during commit, we need to re-run the mileage calculation
		// factoring in the entire year's mileage total that is committed. This
		// solves the issue of out-of-order mileage expenses and acknowledges only
		// committed expenses as the source of truth.

	case "Allowance":
		expenseRateRecord, err := getExpenseRateRecord(app, expenseRecord)
		if err != nil {
			return err
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

func getExpenseRateRecord(app *pocketbase.PocketBase, expenseRecord *models.Record) (*models.Record, error) {
	// Expense rates are stored in the expense_rates collection in PocketBase.
	// The records have an effective_date property that designates the date the
	// rate is effective. We must fetch the appropriate record from the
	// expense_rates collection based on the expense record's date property.
	expenseDate := expenseRecord.GetString("date")
	expenseDateAsTime, parseErr := time.Parse(time.DateOnly, expenseDate)
	if parseErr != nil {
		return nil, parseErr
	}

	// fetch the expense rate record from the expense_rates collection
	expenseRateRecords, findErr := app.Dao().FindRecordsByFilter("expense_rates", "effective_date <= {:expenseDate}", "-effective_date", 1, 0, dbx.Params{
		"expenseDate": expenseDateAsTime.Format("2006-01-02"),
	})
	if findErr != nil {
		return nil, findErr
	}

	// if there are no expense rate records, return an error
	if len(expenseRateRecords) == 0 {
		return nil, errors.New("no expense rate record found for the given date")
	}
	return expenseRateRecords[0], nil
}

// The processExpense function is used to process the expense record. It is
// called by the hooks for the expenses collection to ensure that the record
// is in a valid state before it is created or updated.
func ProcessExpense(app *pocketbase.PocketBase, expenseRecord *models.Record, context echo.Context) error {

	// if the expense record is submitted, return an error
	if expenseRecord.Get("submitted") == true {
		return apis.NewBadRequestError("cannot edit submitted expense", nil)
	}

	// clean the expense record
	if err := cleanExpense(app, expenseRecord); err != nil {
		return err
	}

	// write the pay_period_ending property to the record. This is derived
	// exclusively from the date property.
	payPeriodEnding, ppEndErr := utilities.GeneratePayPeriodEnding(expenseRecord.GetString("date"))
	log.Printf("payPeriodEnding: %s", payPeriodEnding)
	if ppEndErr != nil {
		return apis.NewBadRequestError("Error generating pay_period_ending", ppEndErr)
	}
	expenseRecord.Set("pay_period_ending", payPeriodEnding)

	// validate the expense record
	if err := validateExpense(expenseRecord); err != nil {
		return err
	}
	return nil
}
