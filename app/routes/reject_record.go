package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

func createRejectRecordHandler(app core.App, collectionName string) echo.HandlerFunc {
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
	return func(c echo.Context) error {

		id := c.PathParam("id")

		var req RejectionRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": "you must provide a rejection reason",
				"code":    "invalid_request_body",
			})
		}

		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		userId := authRecord.Id

		var httpResponseStatusCode int

		err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
			record, err := txDao.FindRecordById(collectionName, id)
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{
					Code:    "record_not_found",
					Message: fmt.Sprintf("error fetching record: %v", err),
				}
			}

			// Check if the user is the approver
			if record.GetString("approver") != userId {
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
			if err := txDao.SaveRecord(record); err != nil {
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
				return c.JSON(httpResponseStatusCode, map[string]interface{}{
					"message": codeError.Message,
					"code":    codeError.Code,
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "record rejected successfully"})
	}
}
