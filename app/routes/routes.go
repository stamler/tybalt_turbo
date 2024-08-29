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
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

// Define request body for the bundle-timesheet and unbundle-timesheet routes
type WeekEndingRequest struct {
	WeekEnding string `json:"weekEnding"`
}
type TimeSheetIdRequest struct {
	TimeSheetId string `json:"timeSheetId"`
}

// Add custom routes to the app
func AddRoutes(app *pocketbase.PocketBase) {

	// Add the bundle-timesheet route
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method: http.MethodPost,
			Path:   "/api/bundle-timesheet",
			Handler: func(c echo.Context) error {
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

					// Validate the time entries as a group
					if err := validateTimeEntries(timeEntries); err != nil {
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

					// Get work_week_hours and salary status from the user's
					// admin_profiles record and set the value in the new time sheet.
					admin_profile, err := txDao.FindFirstRecordByFilter("admin_profiles", "uid={:userId}", dbx.Params{
						"userId": userId,
					})
					if err != nil {
						return fmt.Errorf("error fetching user's admin profile: %v", err)
					}
					newTimeSheet.Set("work_week_hours", admin_profile.Get("work_week_hours"))
					newTimeSheet.Set("salary", admin_profile.Get("salary"))

					if err := txDao.SaveRecord(newTimeSheet); err != nil {
						return fmt.Errorf("error creating new time sheet: %v", err)
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
			},
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})

	// add the unbundle-timesheet route

	// This function undoes the bundle-timesheet operation. It will delete the
	// time_sheets record with the id specified in the request body and clear the
	// tsid field in all time entries records that have the same time sheet id.
	// This function will return an error if the time sheet does not exist or if
	// there is an error deleting the time sheet or updating the time entries. It
	// will also error if the submitted, approved, or locked fields are true on
	// the time sheet record.
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method: http.MethodPost,
			Path:   "/api/unbundle-timesheet",
			Handler: func(c echo.Context) error {
				var req TimeSheetIdRequest
				if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
					return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
				}

				// get the auth record from the context
				authRecord := c.Get(apis.ContextAuthRecordKey).(*models.Record)
				userId := authRecord.Id

				// Start a transaction
				err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {

					// Get the time sheet record
					timeSheet, err := txDao.FindRecordById("time_sheets", req.TimeSheetId)
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
					if timeSheet.Get("approved") != "" {
						if timeSheet.Get("rejected") == false {
							return fmt.Errorf("approved time sheets must be rejected before being unbundled")
						}
					}

					// locked time sheets cannot be unbundled
					if timeSheet.Get("locked") == true {
						return fmt.Errorf("locked time sheets cannot be unbundled")
					}

					// Get the time entries
					timeEntries, err := txDao.FindRecordsByFilter("time_entries", "uid={:userId} && tsid={:timeSheetId}", "", 0, 0, dbx.Params{
						"userId":      userId,
						"timeSheetId": req.TimeSheetId,
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
			},
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})

	// Add the approve-timesheet route

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
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method: http.MethodPost,
			Path:   "/api/approve-timesheet",
			Handler: func(c echo.Context) error {
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
			},
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})

	// Add the reject-timesheet route

	// This route handles the rejection of a timesheet.
	// It performs the following actions:
	// 1. Validates the request body for a valid timesheet ID and rejection reason.
	// 2. Retrieves the authenticated user's ID.
	// 3. Runs a database transaction to:
	//    a. Fetch the timesheet by ID.
	//    b. Verify that the authenticated user is the assigned approver.
	//    c. Check if the timesheet is submitted and not locked or already rejected.
	//    d. Set the rejection timestamp, reason, and rejector.
	//    e. Save the updated timesheet.
	// 4. Returns a success message if rejected, or an error message if any checks fail.
	// This ensures that only valid, submitted timesheets can be rejected by the correct user.
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method: http.MethodPost,
			Path:   "/api/reject-timesheet",
			Handler: func(c echo.Context) error {
				var req struct {
					TimeSheetId     string `json:"timeSheetId"`
					RejectionReason string `json:"rejectionReason"`
				}
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
			},
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})
}

// This function will validate the time entries as a group. If the validation
// fails, it will return an error. If the validation passes, it will return nil.
func validateTimeEntries(entries []*models.Record) error {
	// print the number of entries
	fmt.Println("Number of entries:", len(entries))

	// Implement your validation logic here
	// For example:
	// - Check if all required fields are filled
	// - Validate that hours are within acceptable ranges
	// - Ensure total hours match expected values
	// Return an error if validation fails
	return nil
}
