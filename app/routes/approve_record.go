package routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func createApproveRecordHandler(app core.App, collectionName string) func(e *core.RequestEvent) error {
	// This route handles the approval of a record.
	// It performs the following actions:
	// 1. Retrieves the authenticated user's ID.
	// 2. Runs a database transaction to:
	//    a. Fetch the record by ID.
	//    b. Verify that the authenticated user is the assigned approver.
	//    c. Check if the record is submitted and not committed or already approved.
	//    d. Set the approval timestamp.
	//    e. Save the updated record.
	// 3. Returns a success message if approved, or an error message if any checks fail.
	// This ensures that only valid, submitted records can be approved by the correct user.
	return func(e *core.RequestEvent) error {
		if err := requireExpensesEditing(app, collectionName); err != nil {
			return err
		}

		authRecord := e.Auth
		userId := authRecord.Id

		var httpResponseStatusCode int

		err := app.RunInTransaction(func(txApp core.App) error {
			record, err := txApp.FindRecordById(collectionName, e.Request.PathValue("id"))
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{
					Code:    "record_not_found",
					Message: fmt.Sprintf("error fetching record: %v", err),
				}
			}

			// Check if the user is the approver
			if record.GetString("approver") != userId {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{
					Code:    "unauthorized",
					Message: "you are not authorized to approve this record",
				}
			}

			// Check if the record is submitted
			if !record.GetBool("submitted") {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "record_not_submitted",
					Message: "only submitted records can be approved",
				}
			}

			// Check if the record is committed
			if !record.GetDateTime("committed").IsZero() {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "record_committed",
					Message: "committed records cannot be approved",
				}
			}

			// Check if the record is already approved
			if !record.GetDateTime("approved").IsZero() {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "record_already_approved",
					Message: "this record is already approved",
				}
			}

			if poErr := validateExpensePurchaseOrderIsActive(txApp, record); poErr != nil {
				httpResponseStatusCode = http.StatusBadRequest
				if poErr.Code == "purchase_order_lookup_error" {
					httpResponseStatusCode = http.StatusInternalServerError
				}
				return poErr
			}

			// Set the approved timestamp
			record.Set("approved", time.Now())

			// Save the updated record
			if err := txApp.Save(record); err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_saving_record",
					Message: fmt.Sprintf("error saving record: %v", err),
				}
			}

			return nil
		})

		if err != nil {
			return e.JSON(httpResponseStatusCode, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]string{"message": "Record approved successfully"})
	}
}
