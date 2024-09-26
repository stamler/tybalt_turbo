// this file implements cleaning and validation rules for the expenses collection

package hooks

import (
	"errors"
	"log"
	"strings"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/models"
)

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
	if paymentType == "Mileage" || paymentType == "Allowance" {

		// Expense rates are stored in the expense_rates collection in PocketBase.
		// The records have an effective_date property that designates the date the
		// rate is effective. We must fetch the appropriate record from the
		// expense_rates collection based on the expense record's date property.
		expenseDate := expenseRecord.GetString("date")
		expenseDateAsTime, parseErr := time.Parse(time.DateOnly, expenseDate)
		if parseErr != nil {
			return parseErr
		}

		// fetch the expense rate record from the expense_rates collection
		expenseRateRecord, findErr := app.Dao().FindRecordsByFilter("expense_rates", "effective_date <= {:expenseDate}", "-effective_date", 1, 0, dbx.Params{
			"expenseDate": expenseDateAsTime,
		})
		if findErr != nil {
			return findErr
		}

		// if there are no expense rate records, return an error
		if len(expenseRateRecord) == 0 {
			return errors.New("no expense rate record found for the given date")
		}

		if paymentType == "Mileage" {
			// if the paymentType is "Mileage", distance must be an integer greater than
			// 0 and we calculate the total by multiplying distance by the rate
			distance := expenseRecord.GetFloat("distance")
			if distance <= 0 {
				return errors.New("distance must be greater than 0 for mileage expenses")
			}
			// check if the distance is an integer
			if distance != float64(int(distance)) {
				return errors.New("distance must be an integer for mileage expenses")
			}
			// the mileage property on the expense_rate record is a JSON object with
			// keys that represent the lower bound of the distance band and a value
			// that represents the rate for that distance band. We extract the mileage
			// property JSON string into a map[string]interface{} and then set the
			// total field on the expense record.
			expenseRate := expenseRateRecord[0].Get("mileage").(map[string]interface{})
			expenseRecord.Set("rate", expenseRate["rate"])
			expenseRecord.Set("total", distance*expenseRate["rate"].(float64))
		} else if paymentType == "Allowance" {
			// breakfast, lunch, dinner, and lodging are all properties on the
			// expense_rate record. if the paymentType is "Allowance", the
			// allowance_type property of the expenseRecord will have one or more of
			// the following values: Breakfast, Lunch, Dinner, Lodging. It is a JSON
			// array of strings. We use this to determine which of the rates to sum
			// up to get the total allowance for the expense.

			// get the allowance_types property from the expense record
			allowanceTypes := expenseRecord.Get("allowance_types").([]string)

			// sum up the rates for the allowance types that are present
			total := 0.0
			for _, allowanceType := range allowanceTypes {
				total += expenseRateRecord[0].GetFloat(strings.ToLower(allowanceType))
			}

			// build a description of the expense by joining the allowance types
			// with commas
			allowanceDescription := "Allowance for "
			allowanceDescription += strings.Join(allowanceTypes, ", ")

			// set the total and description on the expense record
			expenseRecord.Set("total", total)
			expenseRecord.Set("description", allowanceDescription)
		}
	}
	return nil
}

// The validateExpense function is used to validate the expense record. It is
// called by ProcessExpense to ensure that the record is in a valid state before
// it is created or updated.
func validateExpense(app *pocketbase.PocketBase, expenseRecord *models.Record) error {

	paymentType := expenseRecord.GetString("payment_type")
	isAllowance := paymentType == "Allowance"

	validationsErrors := validation.Errors{
		"date": validation.Validate(
			expenseRecord.Get("date"),
			validation.Required.Error("date is required"),
			validation.Date("2006-01-02").Error("must be a valid date"),
		),
		"description": validation.Validate(
			expenseRecord.Get("description"),
			validation.When(!isAllowance,
				validation.Required.Error("required for non-allowance expenses"),
				validation.Length(5, 0).Error("must be at least 5 characters"),
			),
		),
		"vendor_name": validation.Validate(
			expenseRecord.Get("vendor_name"),
			validation.When(!isAllowance,
				validation.Required.Error("vendor_name is required for non-allowance expenses"),
				validation.Length(2, 0).Error("must be at least 2 characters"),
			),
		),
	}.Filter()

	return validationsErrors

}

// The processExpense function is used to process the expense record. It is
// called by the hooks for the expenses collection to ensure that the record
// is in a valid state before it is created or updated.
func ProcessExpense(app *pocketbase.PocketBase, expenseRecord *models.Record, context echo.Context) error {

	// clean the expense record
	if err := cleanExpense(app, expenseRecord); err != nil {
		return err
	}

	// write the pay_period_ending property to the record. This is derived
	// exclusively from the date property.
	payPeriodEnding, ppEndErr := generatePayPeriodEnding(expenseRecord.GetString("date"))
	log.Printf("payPeriodEnding: %s", payPeriodEnding)
	if ppEndErr != nil {
		return apis.NewBadRequestError("Error generating pay_period_ending", ppEndErr)
	}
	expenseRecord.Set("pay_period_ending", payPeriodEnding)

	// validate the expense record
	if err := validateExpense(app, expenseRecord); err != nil {
		return err
	}
	return nil
}
