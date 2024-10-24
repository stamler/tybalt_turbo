package routes

import (
	"fmt"
	"net/http"
	"time"
	"tybalt/utilities"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

func createCommitRecordHandler(app *pocketbase.PocketBase, collectionName string) echo.HandlerFunc {
	// This route handles the committing of a record.
	// It performs the following actions:
	// 1. Retrieves the authenticated user's ID
	// 2. Runs a database transaction to:
	//    a. Fetch the record by ID.
	//    b. Verify that the authenticated user has the required permissions to commit the record.
	//    c. Verify the record is both submitted and approved.
	//		d. Verify the record isn't rejected.
	//    e. Set committed to the current timestamp.
	//    f. Set committer to the authenticated user's ID.
	// 		g. If the record has a committed_week_ending property, update it with the
	//       appropriate week_ending date.
	//		h. If the record is in the expenses collection and has a payment_type of
	//       "Mileage", update the total based on the committed mileage during the
	//       annual fiscal period that corresponds to the date of the record.
	//    i. Save the updated record.
	// 3. Returns a success message if committed, or an error message if any checks fail.
	// This ensures that only the record's owner can submit it.
	return func(c echo.Context) error {
		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		userId := authRecord.Id

		var httpResponseStatusCode int

		err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
			record, err := txDao.FindRecordById(collectionName, c.PathParam("id"))
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{
					Code:    "record_not_found",
					Message: fmt.Sprintf("error fetching record: %v", err),
				}
			}

			// Verify the caller has the commit claim by querying the user_claims
			// collection for a record with uid that matches the caller's ID and cid
			// who's name in the claims collection is "commit". If the record exists,
			// the caller has the commit claim.
			hasCommitClaim, err := utilities.HasClaim(txDao, userId, "commit")
			if err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_fetching_user_claims",
					Message: fmt.Sprintf("error fetching user claims: %v", err),
				}
			}
			if !hasCommitClaim {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{
					Code:    "unauthorized",
					Message: "you are not authorized to commit this record",
				}
			}

			// Check if the record is submitted and approved
			if !record.GetBool("submitted") || record.GetDateTime("approved").IsZero() {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "record_not_submitted_or_approved",
					Message: "this record is not submitted or approved",
				}
			}

			// Check if the record is rejected
			if !record.GetDateTime("rejected").IsZero() {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "record_rejected",
					Message: "rejected records cannot be committed",
				}
			}

			// Set commit properties
			now := time.Now()
			record.Set("committer", userId)
			record.Set("committed", now)

			// if the record is an expense, set the committed_week_ending property to
			// the week_ending date that corresponds to the committed timestamp. If
			// the payment_type is "Mileage", also update the total based on the
			// committed mileage during the annual fiscal period that corresponds to
			// the date of the record.
			if record.Collection().Name == "expenses" {
				weekEnding, err := utilities.GenerateWeekEnding(now.Format(time.DateOnly))
				if err != nil {
					return &CodeError{
						Code:    "error_generating_week_ending",
						Message: fmt.Sprintf("error generating week ending: %v", err),
					}
				}
				record.Set("committed_week_ending", weekEnding)

				expenseRateRecord, err := utilities.GetExpenseRateRecord(app, record)
				if err != nil {
					return err
				}

				if record.GetString("payment_type") == "Mileage" {
					totalMileageExpense, mileageErr := utilities.CalculateMileageTotal(app, record, expenseRateRecord)
					if mileageErr != nil {
						return mileageErr
					}
					record.Set("total", totalMileageExpense)
				}
			}

			// Save the updated record
			if err := txDao.SaveRecord(record); err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_saving_record",
					Message: fmt.Sprintf("error saving record: %v", err),
				}
			}

			return nil
		})

		if err != nil {
			return c.JSON(httpResponseStatusCode, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Record submitted successfully"})
	}
}
