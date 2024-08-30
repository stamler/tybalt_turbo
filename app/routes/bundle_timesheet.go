package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

func createBundleTimesheetHandler(app *pocketbase.PocketBase) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req WeekEndingRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}

		// Validate the date
		weekEndingTime, err := time.Parse("2006-01-02", req.WeekEnding)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid date format. Use YYYY-MM-DD"})
		}

		// validate weekEndingTime is a Saturday
		if weekEndingTime.Weekday() != time.Saturday {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Week ending date must be a Saturday"})
		}

		// Store the week ending date string
		weekEnding := weekEndingTime.Format("2006-01-02")

		// get the auth record from the context
		authRecord := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		userId := authRecord.Id

		/*
			This function will throw an error if a time_sheets record already
			exists for the specified uid and week_ending. If a timesheet does not
			exist, it will validate information across all of the user's time
			entries records for the week ending date. If the information is valid,
			it will then create a new time_sheets record for the user with the
			given week ending date then write the timesheet id to every time
			entries record that has the same user id and week ending date.
		*/

		// Start a transaction
		err = app.Dao().RunInTransaction(func(txDao *daos.Dao) error {

			// Check if a time sheet already exists. This is also prevented by the
			// unique index on the time_sheets table so may not be necessary, but
			// it does send a more user-friendly error message.
			existingTimeSheet, err := txDao.FindFirstRecordByFilter("time_sheets", "uid={:userId} && week_ending={:weekEnding}", dbx.Params{
				"userId":     userId,
				"weekEnding": weekEnding,
			})
			if err == nil && existingTimeSheet != nil {
				return fmt.Errorf("a time sheet already exists for this user and week ending date")
			}

			// Get the candidate time entries
			timeEntries, err := txDao.FindRecordsByFilter("time_entries", "uid={:userId} && week_ending={:weekEnding}", "", 0, 0, dbx.Params{
				"userId":     userId,
				"weekEnding": weekEnding,
			})
			if err != nil {
				return fmt.Errorf("error fetching time entries: %v", err)
			}

			// Load the user's admin_profile record to get values for the new
			// time_sheets record
			admin_profile, err := txDao.FindFirstRecordByFilter("admin_profiles", "uid={:userId}", dbx.Params{
				"userId": userId,
			})
			if err != nil {
				return fmt.Errorf("error fetching user's admin profile: %v", err)
			}
			// Validate the time entries as a group
			if err := validateTimeEntries(txDao, admin_profile, timeEntries); err != nil {
				return err
			}

			// Get the collection for time_sheets
			time_sheets_collection, err := app.Dao().FindCollectionByNameOrId("time_sheets")
			if err != nil {
				return fmt.Errorf("error fetching time_sheets collection: %v", err)
			}

			// Get the manager (approver) from the profiles collection
			profile, err := txDao.FindFirstRecordByFilter("profiles", "uid = {:userId}", dbx.Params{
				"userId": userId,
			})
			if err != nil {
				return fmt.Errorf("error fetching user profile: %v", err)
			}
			approver := profile.Get("manager")

			// Create new time sheet
			newTimeSheet := models.NewRecord(time_sheets_collection)
			newTimeSheet.Set("uid", userId)
			newTimeSheet.Set("week_ending", weekEnding)
			newTimeSheet.Set("approver", approver)
			newTimeSheet.Set("submitted", true)

			// set values in the new time sheet
			newTimeSheet.Set("work_week_hours", admin_profile.Get("work_week_hours"))
			newTimeSheet.Set("salary", admin_profile.Get("salary"))

			if err := txDao.SaveRecord(newTimeSheet); err != nil {
				return fmt.Errorf("error creating new time sheet: %v", err)
			}

			// if the admin_profile.skip_min_time_check is set to "on_next_bundle",
			// then we need to change it to "no" and save the record
			if admin_profile.Get("skip_min_time_check") == "on_next_bundle" {
				admin_profile.Set("skip_min_time_check", "no")
				if err := txDao.SaveRecord(admin_profile); err != nil {
					return fmt.Errorf("error updating admin profile: %v", err)
				}
			}

			// Update time entries with new time sheet ID
			for _, entry := range timeEntries {
				entry.Set("tsid", newTimeSheet.Id)
				if err := txDao.SaveRecord(entry); err != nil {
					return fmt.Errorf("error updating time entry: %v", err)
				}
			}

			return nil // Return nil if transaction is successful
		})

		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Time sheet processed successfully"})
	}
}
