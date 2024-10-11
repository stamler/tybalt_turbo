package hooks

import (
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/models"
)

// These constants are used to determine whether an expense is within the
// allowed percentage or value of the total of a purchase order. The lesser of
// the two limits is used to determine if the expense is valid.
const MAX_PURCHASE_ORDER_EXCESS_PERCENT = 0.05
const MAX_PURCHASE_ORDER_EXCESS_VALUE = 100.0

// The validateExpense function is used to validate the expense record. It is
// called by ProcessExpense to ensure that the record is in a valid state before
// it is created or updated.
func validateExpense(expenseRecord *models.Record, poRecord *models.Record) error {

	poTotal := 0.0
	poType := "Normal"
	poRecordProvided := false
	totalLimit := 0.0
	excessErrorText := fmt.Sprintf("%0.2f%%", MAX_PURCHASE_ORDER_EXCESS_PERCENT*100)
	if poRecord != nil {
		poRecordProvided = true
		poTotal = poRecord.GetFloat("total")
		poType = poRecord.GetString("type")

		// The maximum allowed amount is the lesser of the value and percent limits
		totalLimit = poTotal * (1.0 + MAX_PURCHASE_ORDER_EXCESS_PERCENT)
		if MAX_PURCHASE_ORDER_EXCESS_VALUE < poTotal*MAX_PURCHASE_ORDER_EXCESS_PERCENT {
			totalLimit = poTotal + MAX_PURCHASE_ORDER_EXCESS_VALUE
			excessErrorText = fmt.Sprintf("$%0.2f", MAX_PURCHASE_ORDER_EXCESS_VALUE)
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
		return apis.NewBadRequestError("an expense against a purchase_order cannot be validated without a corresponding purchase order record", nil)
	}

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
			validation.When(limitNonPoAmounts && !hasPurchaseOrder && !isMileage && !isFuelCard && !isPersonalReimbursement && !isAllowance,
				validation.Max(NO_PO_EXPENSE_LIMIT).Exclusive().Error(fmt.Sprintf("a purchase order is required for expenses of $%0.2f or more", NO_PO_EXPENSE_LIMIT)),
			),
			validation.When(hasPurchaseOrder && poType == "Normal",
				validation.Max(totalLimit).Exclusive().Error(fmt.Sprintf("expense exceeds purchase order total of $%0.2f by more than %s", poTotal, excessErrorText)),
			),
			// TODO: Add validation for Cumulative POs
			// TODO: Add validation for Reccuring POs
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
