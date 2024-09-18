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
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}

		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		userId := authRecord.Id

		err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
			timeSheet, err := txDao.FindRecordById("time_sheets", id)
			if err != nil {
				return fmt.Errorf("error fetching time sheet: %v", err)
			}

			// Check if the user is the approver
			if timeSheet.GetString("approver") != userId {
				return fmt.Errorf("you are not authorized to reject this time sheet")
			}

			// Check if the timesheet is submitted
			if !timeSheet.GetBool("submitted") {
				return fmt.Errorf("only submitted time sheets can be rejected")
			}

			// Check if the timesheet is locked
			if timeSheet.GetBool("locked") {
				return fmt.Errorf("locked time sheets cannot be rejected")
			}

			// Check if the timesheet is already rejected
			if timeSheet.GetBool("rejected") {
				return fmt.Errorf("this time sheet is already rejected")
			}

			// Check if the rejection reason is at least 4 characters long
			if len(req.RejectionReason) < 4 {
				return fmt.Errorf("rejection reason must be at least 4 characters long")
			}

			// Set the rejection timestamp, reason, and rejector
			timeSheet.Set("rejected", true)
			timeSheet.Set("rejection_reason", req.RejectionReason)
			timeSheet.Set("rejector", userId)

			// Save the updated timesheet
			if err := txDao.SaveRecord(timeSheet); err != nil {
				return fmt.Errorf("error saving time sheet: %v", err)
			}

			return nil
		})

		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Timesheet rejected successfully"})
	}
}
