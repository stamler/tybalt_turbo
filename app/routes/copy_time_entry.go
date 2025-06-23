package routes

import (
	"fmt"
	"net/http"
	"time"

	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
)

// createCopyTimeEntryHandler returns a route handler that duplicates the given
// time_entries record but with the date moved forward by one day. The handler
// enforces two constraints:
//  1. The authenticated user must own the original record (uid matches).
//  2. The original record must not already be bundled into a time sheet
//     (tsid field must be an empty string).
//
// On success the handler creates a brand-new time_entries record (letting
// PocketBase generate the id) and returns a 201 JSON payload containing the id
// of the newly-created record.
func createCopyTimeEntryHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		userId := authRecord.Id

		var httpResponseStatusCode int
		var newRecordId string

		err := app.RunInTransaction(func(txApp core.App) error {
			originalId := e.Request.PathValue("id")

			// Fetch the original record
			original, err := txApp.FindRecordById("time_entries", originalId)
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{
					Code:    "record_not_found",
					Message: fmt.Sprintf("time entry %s not found", originalId),
				}
			}

			// Ensure the authenticated user owns the record
			if original.GetString("uid") != userId {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{
					Code:    "unauthorized",
					Message: "you are not the owner of this time entry",
				}
			}

			// Reject if the entry is already bundled into a time sheet
			if original.GetString("tsid") != "" {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "entry_already_bundled",
					Message: "cannot copy a time entry that is already bundled into a time sheet",
				}
			}

			// Parse and increment the date
			dateStr := original.GetString("date")
			originalDate, parseErr := time.Parse("2006-01-02", dateStr)
			if parseErr != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "invalid_date",
					Message: fmt.Sprintf("invalid date format on original record: %v", parseErr),
				}
			}
			newDate := originalDate.AddDate(0, 0, 1).Format("2006-01-02")

			// Prepare the new record using the same collection schema
			collection := original.Collection()
			newRecord := core.NewRecord(collection)

			// Copy all schema fields except those that must change/are managed
			for _, field := range collection.Fields {
				fieldName := field.GetName()
				switch fieldName {
				case "id", "created", "updated", "date", "week_ending", "tsid":
					// skip; handled separately / left blank
					continue
				default:
					newRecord.Set(fieldName, original.Get(fieldName))
				}
			}

			// Explicitly set required fields
			newRecord.Set("uid", userId)
			newRecord.Set("date", newDate)

			// Calculate correct week_ending based on the new date (Saturday)
			weekEnding, wkErr := utilities.GenerateWeekEnding(newDate)
			if wkErr != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "invalid_week_ending",
					Message: fmt.Sprintf("error generating week ending: %v", wkErr),
				}
			}
			newRecord.Set("week_ending", weekEnding)
			// tsid intentionally left empty so the record is unbundled

			// Persist the new record (triggers all hooks/validation)
			if err := txApp.Save(newRecord); err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_saving_record",
					Message: fmt.Sprintf("error saving copied record: %v", err),
				}
			}
			newRecordId = newRecord.Id
			return nil
		})

		if err != nil {
			// If the error is our custom CodeError, use the prepared status code
			if codeErr, ok := err.(*CodeError); ok {
				if httpResponseStatusCode == 0 {
					httpResponseStatusCode = http.StatusBadRequest
				}
				return e.JSON(httpResponseStatusCode, map[string]any{
					"error": codeErr.Message,
					"code":  codeErr.Code,
				})
			}
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusCreated, map[string]string{
			"message":       "Time entry copied to tomorrow",
			"new_record_id": newRecordId,
		})
	}
}
