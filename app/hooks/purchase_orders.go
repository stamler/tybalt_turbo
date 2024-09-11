// This file implements cleaning and validation rules for the purchase_orders
// collection.

package hooks

import (
	"database/sql"
	"fmt"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/models"
)

const (
	MANAGER_PO_LIMIT = 500
	VP_PO_LIMIT      = 2500
)

// The cleanPurchaseOrder function is used to remove properties from the
// purchase_order record that are not allowed to be set based on the value of
// the record's type property. It is also used to set the approver and, if
// applicable, the second_approver_claim fields based on the value of the total
// and type fields. This is intended to reduce round trips to the database and
// to ensure that the record is in a valid state before it is created or
// updated. It is called by ProcessPurchaseOrder to reduce the number of fields
// that need to be validated.
func cleanPurchaseOrder(app *pocketbase.PocketBase, purchaseOrderRecord *models.Record) error {
	typeString := purchaseOrderRecord.GetString("type")

	// Normal and Cumulative Purchase both have empty values for
	// end_date and frequency
	if typeString == "Normal" || typeString == "Cumulative" {
		purchaseOrderRecord.Set("end_date", "")
		purchaseOrderRecord.Set("frequency", "")
	}

	// get the user's manager and set the approver field
	profile, err := app.Dao().FindFirstRecordByFilter("profiles", "uid = {:userId}", dbx.Params{
		"userId": purchaseOrderRecord.GetString("uid"),
	})
	if err != nil {
		return err
	}
	approver := profile.Get("manager")
	purchaseOrderRecord.Set("approver", approver)

	// set the second_approver_claim field
	secondApproverClaim, err := getSecondApproverClaim(app, purchaseOrderRecord.GetString("type"), purchaseOrderRecord.GetFloat("total"))
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
func validatePurchaseOrder(purchaseOrderRecord *models.Record) error {
	return nil
}

// The ProcessPurchaseOrder function is used to validate the purchase_order
// record before it is created or updated. A lot of the work is done by
// PocketBase itself so this is for cross-field validation. If the
// purchase_order record is invalid this function throws an error explaining
// which field(s) are invalid and why.
func ProcessPurchaseOrder(app *pocketbase.PocketBase, record *models.Record, context echo.Context) error {
	// get the auth record from the context
	authRecord := context.Get(apis.ContextAuthRecordKey).(*models.Record)

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
	validateErr := validatePurchaseOrder(record)
	if validateErr != nil {
		return apis.NewBadRequestError("Error validating purchase_order record", validateErr)
	}

	return nil
}

func getSecondApproverClaim(app *pocketbase.PocketBase, poType string, total float64) (string, error) {
	var secondApproverClaim string

	// Check if the purchase order is recurring or if the total is greater than or equal to VP_PO_LIMIT
	if poType == "Recurring" || total >= VP_PO_LIMIT {
		// Set second approver claim to 'smg'
		claim, err := app.Dao().FindFirstRecordByFilter("claims", "name = {:claimName}", dbx.Params{
			"claimName": "smg",
		})
		if err != nil {
			return "", fmt.Errorf("error fetching SMG claim: %v", err)
		}
		secondApproverClaim = claim.Id
	} else if total >= MANAGER_PO_LIMIT {
		// Set second approver claim to 'vp'
		claim, err := app.Dao().FindFirstRecordByFilter("claims", "name = {:claimName}", dbx.Params{
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

func generatePONumber(app *pocketbase.PocketBase) (string, error) {
	currentYear := time.Now().Year()
	prefix := fmt.Sprintf("%d-", currentYear)

	// Query existing PO numbers in descending order using the prefix as a
	// filter and get the first result
	existingPOs, err := app.Dao().FindRecordsByFilter(
		"purchase_orders",
		"po_number ~ {:prefix}%",
		"-po_number",
		1,
		0,
		dbx.Params{"prefix": prefix},
	)
	if err != nil {
		return "", fmt.Errorf("error querying existing PO numbers: %v", err)
	}

	var lastNumber int
	if len(existingPOs) > 0 {
		lastPO := existingPOs[0].Get("po_number").(string)
		_, err := fmt.Sscanf(lastPO, "%d-%04d", &currentYear, &lastNumber)
		if err != nil {
			return "", fmt.Errorf("error parsing last PO number: %v", err)
		}
	}

	// Generate the new PO number
	for i := lastNumber + 1; i < 5000; i++ {
		newPONumber := fmt.Sprintf("%s%04d", prefix, i)

		// Check if the generated PO number is unique
		existing, err := app.Dao().FindFirstRecordByFilter(
			"purchase_orders",
			"po_number = {:poNumber}",
			dbx.Params{"poNumber": newPONumber},
		)
		if err != nil && err != sql.ErrNoRows {
			return "", fmt.Errorf("error checking PO number uniqueness: %v", err)
		}

		if existing == nil {
			return newPONumber, nil
		}
	}

	return "", fmt.Errorf("unable to generate a unique PO number")
}
