package routes

import (
	"fmt"
	"net/http"
	"time"
	"tybalt/notifications"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
)

func createRejectRecordHandler(app core.App, collectionName string) func(e *core.RequestEvent) error {
	// This route handles the rejection of a record.
	// It performs the following actions:
	// 1. Gets the record ID from the URL.
	// 2. Validates the request body for a valid rejection reason.
	// 3. Retrieves the authenticated user's ID.
	// 4. Runs a database transaction to:
	//    a. Fetch the record by ID.
	//    b. Verify that the authenticated user is the assigned approver.
	//    c. Check if the record is submitted and not committed or already rejected.
	//    d. Set the rejection timestamp, reason, and rejector.
	//    e. Save the updated record.
	// 5. Returns a success message if rejected, or an error message if any checks fail.
	// This ensures that only valid, submitted records can be rejected by the correct user.
	return func(e *core.RequestEvent) error {
		if err := requireExpensesEditing(app, collectionName); err != nil {
			return err
		}

		id := e.Request.PathValue("id")

		var req RejectionRequest
		if err := e.BindBody(&req); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": "you must provide a rejection reason",
				"code":    "invalid_request_body",
			})
		}

		authRecord := e.Auth
		userId := authRecord.Id

		var httpResponseStatusCode int

		err := app.RunInTransaction(func(txApp core.App) error {
			record, err := txApp.FindRecordById(collectionName, id)
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{
					Code:    "record_not_found",
					Message: fmt.Sprintf("error fetching record: %v", err),
				}
			}

			// Check if the user is authorized to reject: approver OR user with commit claim
			isApprover := record.GetString("approver") == userId
			hasCommitClaim, err := utilities.HasClaim(txApp, authRecord, "commit")
			if err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_fetching_user_claims",
					Message: fmt.Sprintf("error fetching user claims: %v", err),
				}
			}
			if !isApprover && !hasCommitClaim {
				httpResponseStatusCode = http.StatusUnauthorized
				return &CodeError{
					Code:    "rejection_unauthorized",
					Message: "you are not authorized to reject this record",
				}
			}

			// Check if the record is submitted
			if !record.GetBool("submitted") {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "record_not_submitted",
					Message: "only submitted records can be rejected",
				}
			}

			// Check if the record is committed
			if !record.GetDateTime("committed").IsZero() {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "record_committed",
					Message: "committed records cannot be rejected",
				}
			}

			// Check if the record is already rejected
			if !record.GetDateTime("rejected").IsZero() {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "record_already_rejected",
					Message: "this record is already rejected",
				}
			}

			// Check if the rejection reason is at least 4 characters long
			if len(req.RejectionReason) < 4 {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "rejection_reason_too_short",
					Message: "rejection reason must be at least 4 characters long",
				}
			}

			// Set the rejection timestamp, reason, and rejector
			record.Set("rejected", time.Now())
			record.Set("rejection_reason", req.RejectionReason)
			record.Set("rejector", userId)

			// Save the updated record
			if err := txApp.Save(record); err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "record_save_error",
					Message: fmt.Sprintf("error saving record: %v", err),
				}
			}

			return nil
		})

		if err != nil {
			if codeError, ok := err.(*CodeError); ok {
				return e.JSON(httpResponseStatusCode, map[string]interface{}{
					"message": codeError.Message,
					"code":    codeError.Code,
				})
			}
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// After successful rejection, send notifications (outside transaction to avoid blocking)
		// Reload the record to get the updated values
		rejectedRecord, err := app.FindRecordById(collectionName, id)
		if err == nil {
			// Queue notifications based on collection type
			switch collectionName {
			case "time_sheets":
				// Log error but don't fail the request if notification fails
				if notifErr := notifications.QueueTimesheetRejectedNotifications(app, rejectedRecord, userId, req.RejectionReason); notifErr != nil {
					app.Logger().Error(
						"error queueing timesheet rejection notifications",
						"timesheet_id", id,
						"error", notifErr,
					)
				}
			case "expenses":
				// Log error but don't fail the request if notification fails
				if notifErr := notifications.QueueExpenseRejectedNotifications(app, rejectedRecord, userId, req.RejectionReason); notifErr != nil {
					app.Logger().Error(
						"error queueing expense rejection notifications",
						"expense_id", id,
						"error", notifErr,
					)
				}
			}
		}

		return e.JSON(http.StatusOK, map[string]string{"message": "record rejected successfully"})
	}
}
