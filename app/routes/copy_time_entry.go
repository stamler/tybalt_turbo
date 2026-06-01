package routes

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"tybalt/errs"
	"tybalt/hooks"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
)

type copiedTimeEntry struct {
	ID         string
	Date       string
	WeekEnding string
}

func copyTimeEntryErrorStatus(err error) int {
	codeErr, ok := err.(*CodeError)
	if !ok {
		return http.StatusBadRequest
	}

	switch codeErr.Code {
	case "invalid_date", "invalid_week_ending", "error_saving_record":
		return http.StatusInternalServerError
	default:
		return http.StatusBadRequest
	}
}

func copyTimeEntryWithDayOffset(txApp core.App, authRecord *core.Record, original *core.Record, dayOffset int) (copiedTimeEntry, error) {
	dateStr := original.GetString("date")
	originalDate, parseErr := time.Parse("2006-01-02", dateStr)
	if parseErr != nil {
		return copiedTimeEntry{}, &CodeError{
			Code:    "invalid_date",
			Message: fmt.Sprintf("invalid date format on original record: %v", parseErr),
		}
	}
	newDate := originalDate.AddDate(0, 0, dayOffset).Format("2006-01-02")

	collection := original.Collection()
	newRecord := core.NewRecord(collection)

	for _, field := range collection.Fields {
		fieldName := field.GetName()
		switch fieldName {
		case "id", "created", "updated", "date", "week_ending", "tsid":
			continue
		default:
			newRecord.Set(fieldName, original.Get(fieldName))
		}
	}

	newRecord.Set("uid", authRecord.Id)
	newRecord.Set("date", newDate)

	weekEnding, wkErr := utilities.GenerateWeekEnding(newDate)
	if wkErr != nil {
		return copiedTimeEntry{}, &CodeError{
			Code:    "invalid_week_ending",
			Message: fmt.Sprintf("error generating week ending: %v", wkErr),
		}
	}
	newRecord.Set("week_ending", weekEnding)

	if err := hooks.ProcessTimeEntry(txApp, &core.RecordRequestEvent{
		RequestEvent: &core.RequestEvent{App: txApp, Auth: authRecord},
		Record:       newRecord,
	}); err != nil {
		return copiedTimeEntry{}, err
	}

	if err := txApp.Save(newRecord); err != nil {
		return copiedTimeEntry{}, &CodeError{
			Code:    "error_saving_record",
			Message: fmt.Sprintf("error saving copied record: %v", err),
		}
	}

	return copiedTimeEntry{
		ID:         newRecord.Id,
		Date:       newDate,
		WeekEnding: weekEnding,
	}, nil
}

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
		if err := requireTimeEditing(app); err != nil {
			return err
		}
		if err := requireTimeClaim(app, e.Auth); err != nil {
			return err
		}

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

			copied, err := copyTimeEntryWithDayOffset(txApp, authRecord, original, 1)
			if err != nil {
				httpResponseStatusCode = copyTimeEntryErrorStatus(err)
				return err
			}

			newRecordId = copied.ID
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
			var hookErr *errs.HookError
			if errors.As(err, &hookErr) {
				return e.JSON(hookErr.Status, hookErr)
			}
			return err
		}

		return e.JSON(http.StatusCreated, map[string]string{
			"message":       "Time entry copied to tomorrow",
			"new_record_id": newRecordId,
		})
	}
}
