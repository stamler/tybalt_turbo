package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

func createSubmitRecordHandler(app *pocketbase.PocketBase, collectionName string) echo.HandlerFunc {
	// This route handles the submission of a record.
	// It performs the following actions:
	// 1. Retrieves the authenticated user's ID
	// 2. Runs a database transaction to:
	//    a. Fetch the record by ID.
	//    b. Verify that the authenticated user has the same ID as the record's uid.
	//    c. Verify the record is not yet submitted.
	//		d. Verify the record isn't rejected.
	//    e. Set submitted to true.
	//    f. Save the updated record.
	// 3. Returns a success message if submitted, or an error message if any checks fail.
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

			// Verify the caller is the record's owner
			if record.Get("uid") != userId {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{
					Code:    "unauthorized",
					Message: "you are not authorized to submit this record",
				}
			}

			// Check if the record is submitted
			if record.GetBool("submitted") {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "record_submitted",
					Message: "this record is already submitted",
				}
			}

			// Check if the record is rejected
			if !record.GetDateTime("rejected").IsZero() {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "record_rejected",
					Message: "rejected records cannot be submitted",
				}
			}

			// Set submitted to true
			record.Set("submitted", true)

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
