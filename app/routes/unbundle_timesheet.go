package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/types"
)

func createUnbundleTimesheetHandler(app *pocketbase.PocketBase) echo.HandlerFunc {
	// This function undoes the bundle timesheet operation. It will delete the
	// time_sheets record with the id specified in the request url and clear the
	// tsid field in all time entries records that have the same time sheet id.
	// This function will return an error if the time sheet does not exist or if
	// there is an error deleting the time sheet or updating the time entries. It
	// will also error if the submitted, approved, or committed fields are true
	// on the time sheet record.
	return func(c echo.Context) error {
		// get the auth record from the context
		authRecord := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		userId := authRecord.Id

		// Start a transaction
		err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {

			// Get the time sheet record
			timeSheet, err := txDao.FindRecordById("time_sheets", c.PathParam("id"))
			if err != nil {
				return fmt.Errorf("error fetching time sheet: %v", err)
			}

			if timeSheet == nil {
				return fmt.Errorf("time sheet not found")
			}

			// Check if the uid field in the time sheet record matches the user id
			if timeSheet.Get("uid") != userId {
				return fmt.Errorf("time sheet does not belong to user")
			}

			// approved time sheets must be rejected before being unbundled
			if !timeSheet.Get("approved").(types.DateTime).IsZero() {
				if timeSheet.Get("rejected") == false {
					return fmt.Errorf("approved time sheets must be rejected before being unbundled")
				}
			}

			// committed time sheets cannot be unbundled
			if !timeSheet.GetDateTime("committed").IsZero() {
				return fmt.Errorf("committed time sheets cannot be unbundled")
			}

			// Get the time entries
			timeEntries, err := txDao.FindRecordsByFilter("time_entries", "uid={:userId} && tsid={:timeSheetId}", "", 0, 0, dbx.Params{
				"userId":      userId,
				"timeSheetId": c.PathParam("id"),
			})
			if err != nil {
				return fmt.Errorf("error fetching time entries: %v", err)
			}

			// Clear the tsid field in all time entries
			for _, entry := range timeEntries {
				entry.Set("tsid", "")
				if err := txDao.SaveRecord(entry); err != nil {
					return fmt.Errorf("error updating time entry: %v", err)
				}
			}

			// Delete the time sheet
			if err := txDao.DeleteRecord(timeSheet); err != nil {
				return fmt.Errorf("error deleting time sheet: %v", err)
			}

			return nil // Return nil if transaction is successful
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Time sheet unbundled successfully"})
	}
}
