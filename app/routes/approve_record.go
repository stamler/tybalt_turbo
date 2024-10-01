package routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

func createApproveRecordHandler(app *pocketbase.PocketBase, collectionName string) echo.HandlerFunc {
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
	return func(c echo.Context) error {
		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		userId := authRecord.Id

		err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
			record, err := txDao.FindRecordById(collectionName, c.PathParam("id"))
			if err != nil {
				return fmt.Errorf("error fetching record: %v", err)
			}

			// Check if the user is the approver
			if record.GetString("approver") != userId {
				return fmt.Errorf("you are not authorized to approve this record")
			}

			// Check if the record is submitted
			if !record.GetBool("submitted") {
				return fmt.Errorf("only submitted records can be approved")
			}

			// Check if the record is committed
			if !record.GetDateTime("committed").IsZero() {
				return fmt.Errorf("committed records cannot be approved")
			}

			// Check if the record is already approved
			if !record.GetDateTime("approved").IsZero() {
				return fmt.Errorf("this record is already approved")
			}

			// Set the approved timestamp
			record.Set("approved", time.Now())

			// Save the updated record
			if err := txDao.SaveRecord(record); err != nil {
				return fmt.Errorf("error saving record: %v", err)
			}

			return nil
		})

		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "Record approved successfully"})
	}
}
