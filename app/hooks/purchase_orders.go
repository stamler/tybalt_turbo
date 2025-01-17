// This file implements cleaning and validation rules for the purchase_orders
// collection.

package hooks

import (
	"fmt"
	"time"
	"tybalt/utilities"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const (
	// The amount of money below which a purchase order does not require a second
	// approval.
	MANAGER_PO_LIMIT = 500
	// The amount of money below which a purchase order does not require SMG
	// approval but can be second approved by a VP if necessary.
	VP_PO_LIMIT = 2500
	// The maximum number of days between the start and end dates for a recurring
	// purchase order.
	RECURRING_MAX_DAYS = 400
)

// The cleanPurchaseOrder function is used to remove properties from the
// purchase_order record that are not allowed to be set based on the value of
// the record's type property. It is also used to set the approver and, if
// applicable, the second_approver_claim fields based on the value of the total
// and type fields. This is intended to reduce round trips to the database and
// to ensure that the record is in a valid state before it is created or
// updated. It is called by ProcessPurchaseOrder to reduce the number of fields
// that need to be validated.
func cleanPurchaseOrder(app core.App, purchaseOrderRecord *core.Record) error {
	typeString := purchaseOrderRecord.GetString("type")

	// Normal and Cumulative Purchase both have empty values for
	// end_date and frequency
	if typeString == "Normal" || typeString == "Cumulative" {
		purchaseOrderRecord.Set("end_date", "")
		purchaseOrderRecord.Set("frequency", "")
	}

	// set the second_approver_claim field
	secondApproverClaim, err := getSecondApproverClaim(app, purchaseOrderRecord)
	if err != nil {
		return err
	}
	purchaseOrderRecord.Set("second_approver_claim", secondApproverClaim)

	return nil
}

// cross-field validation is performed in this function. It is expected that the
// purchase_order record has already been cleaned by the cleanPurchaseOrder
// function. This ensures that only the fields that are allowed to be set are
// present in the record prior to validation. The function returns an error if
// the record is invalid, otherwise it returns nil.
func validatePurchaseOrder(app core.App, purchaseOrderRecord *core.Record) error {
	isRecurring := purchaseOrderRecord.GetString("type") == "Recurring"

	dateAsTime, parseErr := time.Parse("2006-01-02", purchaseOrderRecord.Get("date").(string))
	if parseErr != nil {
		return parseErr
	}

	validationsErrors := validation.Errors{
		"date": validation.Validate(
			purchaseOrderRecord.Get("date"),
			validation.Required.Error("date is required"),
			validation.Date("2006-01-02").Error("must be a valid date"),
		),
		"end_date": validation.Validate(
			purchaseOrderRecord.Get("end_date"),
			validation.When(isRecurring,
				validation.Required.Error("end_date is required for recurring purchase orders"),
				validation.Date("2006-01-02").Error("must be a valid date").Min(dateAsTime).RangeError("end date must be after start date").Max(dateAsTime.AddDate(0, 0, RECURRING_MAX_DAYS)).RangeError(fmt.Sprintf("end date must be within %v days of the start date", RECURRING_MAX_DAYS)),
			).Else(
				validation.In("").Error("end_date is not permitted for non-recurring purchase orders"),
			),
		),
		"frequency": validation.Validate(
			purchaseOrderRecord.Get("frequency"),
			validation.When(isRecurring,
				validation.Required.Error("frequency is required for recurring purchase orders"),
			).Else(
				validation.In("").Error("frequency is not permitted for non-recurring purchase orders"))),
		"description": validation.Validate(purchaseOrderRecord.Get("description"), validation.Length(5, 0).Error("must be at least 5 characters")),
		"approver":    validation.Validate(purchaseOrderRecord.GetString("approver"), validation.By(utilities.ApproverHasDivisionPermission(app, purchaseOrderRecord.GetString("approver"), purchaseOrderRecord.GetString("division")))),
		// "global":                validation.Validate(totalHours, validation.Max(18.0).Error("Total hours must not exceed 18")),
	}.Filter()

	return validationsErrors
}

// The ProcessPurchaseOrder function is used to validate the purchase_order
// record before it is created or updated. A lot of the work is done by
// PocketBase itself so this is for cross-field validation. If the
// purchase_order record is invalid this function throws an error explaining
// which field(s) are invalid and why.
func ProcessPurchaseOrder(app core.App, e *core.RecordRequestEvent) error {
	record := e.Record
	// get the auth record from the context
	authRecord := e.Auth

	// If the uid property is not equal to the authenticated user's uid, return an
	// error.
	if record.GetString("uid") != authRecord.Id {
		return apis.NewApiError(400, "uid property must be equal to the authenticated user's id", map[string]validation.Error{})
	}

	// set properties to nil if they are not allowed to be set based on the type
	cleanErr := cleanPurchaseOrder(app, record)
	if cleanErr != nil {
		return apis.NewBadRequestError("Error cleaning purchase_order record", cleanErr)
	}

	// validate the purchase_order record
	if validationErr := validatePurchaseOrder(app, record); validationErr != nil {
		return apis.NewBadRequestError("Validation error", validationErr)
	}

	return nil
}

func getSecondApproverClaim(app core.App, purchaseOrderRecord *core.Record) (string, error) {
	var secondApproverClaim string

	poType := purchaseOrderRecord.GetString("type")
	total := purchaseOrderRecord.GetFloat("total")

	// Calculate the total value for recurring purchase orders
	totalValue := total
	var err error
	if poType == "Recurring" {
		// ignore the number of occurrences, we just want the total value
		_, totalValue, err = utilities.CalculateRecurringPurchaseOrderTotalValue(app, purchaseOrderRecord)
		if err != nil {
			return "", err
		}
	}

	if totalValue >= VP_PO_LIMIT {
		// Set second approver claim to 'smg'
		claim, err := app.FindFirstRecordByFilter("claims", "name = {:claimName}", dbx.Params{
			"claimName": "smg",
		})
		if err != nil {
			return "", fmt.Errorf("error fetching SMG claim: %v", err)
		}
		secondApproverClaim = claim.Id
	} else if totalValue >= MANAGER_PO_LIMIT {
		// Set second approver claim to 'vp'
		claim, err := app.FindFirstRecordByFilter("claims", "name = {:claimName}", dbx.Params{
			"claimName": "vp",
		})
		if err != nil {
			return "", fmt.Errorf("error fetching VP claim: %v", err)
		}
		secondApproverClaim = claim.Id
	}

	// If neither condition is met, secondApproverClaim remains an empty string
	return secondApproverClaim, nil
}
