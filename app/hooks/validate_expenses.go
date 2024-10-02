package hooks

import (
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/models"
)

// The validateExpense function is used to validate the expense record. It is
// called by ProcessExpense to ensure that the record is in a valid state before
// it is created or updated.
func validateExpense(expenseRecord *models.Record) error {

	hasJob := expenseRecord.Get("job") != ""
	hasPurchaseOrder := expenseRecord.Get("purchase_order") != ""
	paymentType := expenseRecord.GetString("payment_type")
	isAllowance := paymentType == "Allowance"
	isPersonalReimbursement := paymentType == "PersonalReimbursement"
	isMileage := paymentType == "Mileage"
	isCorporateCreditCard := paymentType == "CorporateCreditCard"
	isFuelCard := paymentType == "FuelCard"

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
