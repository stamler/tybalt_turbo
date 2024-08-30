package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

func createApproveTimesheetHandler(app *pocketbase.PocketBase) echo.HandlerFunc {
	// This route handles the approval of a timesheet.
	// It performs the following actions:
	// 1. Validates the request body for a valid timesheet ID.
	// 2. Retrieves the authenticated user's ID.
	// 3. Runs a database transaction to:
	//    a. Fetch the timesheet by ID.
	//    b. Verify that the authenticated user is the assigned approver.
	//    c. Check if the timesheet is submitted and not locked or already approved.
	//    d. Set the approval timestamp.
	//    e. Save the updated timesheet.
	// 4. Returns a success message if approved, or an error message if any checks fail.
	// This ensures that only valid, submitted timesheets can be approved by the correct user.
	return func(c echo.Context) error {
		var req TimeSheetIdRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}

		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		userId := authRecord.Id

		err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
			timeSheet, err := txDao.FindRecordById("time_sheets", req.TimeSheetId)
			if err != nil {
				return fmt.Errorf("error fetching time sheet: %v", err)
			}

			// Check if the user is the approver
			if timeSheet.GetString("approver") != userId {
				return fmt.Errorf("you are not authorized to approve this time sheet")
			}

			// Check if the timesheet is submitted
			if !timeSheet.GetBool("submitted") {
				return fmt.Errorf("only submitted time sheets can be approved")
			}

			// Check if the timesheet is locked
			if timeSheet.GetBool("locked") {
				return fmt.Errorf("locked time sheets cannot be approved")
			}

			// Check if the timesheet is already approved
			if !timeSheet.GetDateTime("approved").IsZero() {
				return fmt.Errorf("this time sheet is already approved")
			}

			// Set the approved timestamp
			timeSheet.Set("approved", time.Now())

			// Save the updated timesheet
			if err := txDao.SaveRecord(timeSheet); err != nil {
				return fmt.Errorf("error saving time sheet: %v", err)
			}

			return nil
		})

		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Timesheet approved successfully"})
	}
}
