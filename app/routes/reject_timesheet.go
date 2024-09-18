package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

func createRejectTimesheetHandler(app *pocketbase.PocketBase) echo.HandlerFunc {
	// This route handles the rejection of a timesheet.
	// It performs the following actions:
	// 1. Gets the timesheet ID from the URL.
	// 2. Validates the request body for a valid rejection reason.
	// 3. Retrieves the authenticated user's ID.
	// 4. Runs a database transaction to:
	//    a. Fetch the timesheet by ID.
	//    b. Verify that the authenticated user is the assigned approver.
	//    c. Check if the timesheet is submitted and not locked or already rejected.
	//    d. Set the rejection timestamp, reason, and rejector.
	//    e. Save the updated timesheet.
	// 5. Returns a success message if rejected, or an error message if any checks fail.
	// This ensures that only valid, submitted timesheets can be rejected by the correct user.
	return func(c echo.Context) error {

		id := c.PathParam("id")

		var req struct {
			RejectionReason string `json:"rejectionReason"`
		}
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
			timeSheet, err := txDao.FindRecordById("time_sheets", id)
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{
					Code:    "record_not_found",
					Message: fmt.Sprintf("error fetching time sheet: %v", err),
				}
			}

			// Check if the user is the approver
			if timeSheet.GetString("approver") != userId {
				httpResponseStatusCode = http.StatusUnauthorized
				return &CodeError{
					Code:    "rejection_unauthorized",
					Message: "you are not authorized to reject this time sheet",
				}
			}

			// Check if the timesheet is submitted
			if !timeSheet.GetBool("submitted") {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "timesheet_not_submitted",
					Message: "only submitted time sheets can be rejected",
				}
			}

			// Check if the timesheet is locked
			if timeSheet.GetBool("locked") {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "timesheet_locked",
					Message: "locked time sheets cannot be rejected",
				}
			}

			// Check if the timesheet is already rejected
			if timeSheet.GetBool("rejected") {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "timesheet_already_rejected",
					Message: "this time sheet is already rejected",
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
			timeSheet.Set("rejected", true)
			timeSheet.Set("rejection_reason", req.RejectionReason)
			timeSheet.Set("rejector", userId)

			// Save the updated timesheet
			if err := txDao.SaveRecord(timeSheet); err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "timesheet_save_error",
					Message: fmt.Sprintf("error saving time sheet: %v", err),
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

		return c.JSON(http.StatusOK, map[string]string{"message": "Timesheet rejected successfully"})
	}
}
