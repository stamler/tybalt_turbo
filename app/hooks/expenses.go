// this file implements cleaning and validation rules for the expenses collection

package hooks

import (
	"errors"
	"fmt"
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

// This feature flag is used to limit the amount of expenses that don't have a
// corresponding purchase order. The operation of this will need to be
// revisited if we ever allow expenses to be created without a PO number.
const limitNonPoAmounts = true
const NO_PO_EXPENSE_LIMIT = 500.0

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
		expenseRateRecords, findErr := app.Dao().FindRecordsByFilter("expense_rates", "effective_date <= {:expenseDate}", "-effective_date", 1, 0, dbx.Params{
			"expenseDate": expenseDateAsTime.Format("2006-01-02"),
		})
		if findErr != nil {
			return findErr
		}

		// if there are no expense rate records, return an error
		if len(expenseRateRecords) == 0 {
			return errors.New("no expense rate record found for the given date")
		}
		expenseRateRecord := expenseRateRecords[0]

		if paymentType == "Mileage" {
			// if the paymentType is "Mileage", distance must be a positive integer
			// and we calculate the total by multiplying distance by the rate
			distance := expenseRecord.GetFloat("distance")
			// check if the distance is an integer
			if distance != float64(int(distance)) {
				return errors.New("distance must be an integer for mileage expenses")
			}

			totalMileageExpense, mileageErr := calculateMileageTotal(distance, expenseRateRecord)
			if mileageErr != nil {
				return mileageErr
			}

			expenseRecord.Set("total", totalMileageExpense)
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
				total += expenseRateRecord.GetFloat(strings.ToLower(allowanceType))
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
	isPersonalReimbursement := paymentType == "PersonalReimbursement"
	isMileage := paymentType == "Mileage"
	isCorporateCreditCard := paymentType == "CorporateCreditCard"

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
			validation.When(!isAllowance && !isPersonalReimbursement && !isMileage,
				validation.Required.Error("required for this expense type"),
				validation.Length(2, 0).Error("must be at least 2 characters"),
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
			validation.When(limitNonPoAmounts && expenseRecord.Get("purchase_order") == "",
				validation.Max(NO_PO_EXPENSE_LIMIT).Exclusive().Error(fmt.Sprintf("a purchase order is required for expenses of $%0.2f or more", NO_PO_EXPENSE_LIMIT)),
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
