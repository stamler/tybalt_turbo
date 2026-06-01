package routes

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"tybalt/errs"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type copyTimesheetEntriesResponse struct {
	Message      string   `json:"message"`
	CopiedCount  int      `json:"copied_count"`
	NewRecordIDs []string `json:"new_record_ids"`
	WeekEnding   string   `json:"week_ending"`
}

func createCopyTimesheetEntriesNextWeekHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireTimeEditing(app); err != nil {
			return err
		}
		if err := requireTimeClaim(app, e.Auth); err != nil {
			return err
		}

		authRecord := e.Auth
		userID := authRecord.Id
		var httpResponseStatusCode int
		var response copyTimesheetEntriesResponse

		err := app.RunInTransaction(func(txApp core.App) error {
			sourceID := e.Request.PathValue("id")
			sourceTimeSheet, err := txApp.FindRecordById("time_sheets", sourceID)
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{
					Code:    "record_not_found",
					Message: fmt.Sprintf("time sheet %s not found", sourceID),
				}
			}

			if sourceTimeSheet.GetString("uid") != userID {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{
					Code:    "unauthorized",
					Message: "you are not the owner of this time sheet",
				}
			}

			sourceWeekEnding, err := time.Parse(time.DateOnly, sourceTimeSheet.GetString("week_ending"))
			if err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "invalid_week_ending",
					Message: fmt.Sprintf("invalid week_ending on source time sheet: %v", err),
				}
			}
			targetWeekEnding := sourceWeekEnding.AddDate(0, 0, 7).Format(time.DateOnly)

			existingTimeSheet, err := txApp.FindFirstRecordByFilter("time_sheets", "uid={:userID} && week_ending={:weekEnding}", dbx.Params{
				"userID":     userID,
				"weekEnding": targetWeekEnding,
			})
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_checking_target_time_sheet",
					Message: fmt.Sprintf("error checking target time sheet: %v", err),
				}
			}
			if err == nil && existingTimeSheet != nil {
				httpResponseStatusCode = http.StatusConflict
				return &CodeError{
					Code:    "target_time_sheet_exists",
					Message: "a time sheet already exists for the next week",
				}
			}

			existingEntry, err := txApp.FindFirstRecordByFilter("time_entries", "uid={:userID} && week_ending={:weekEnding}", dbx.Params{
				"userID":     userID,
				"weekEnding": targetWeekEnding,
			})
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_checking_target_time_entries",
					Message: fmt.Sprintf("error checking target time entries: %v", err),
				}
			}
			if err == nil && existingEntry != nil {
				httpResponseStatusCode = http.StatusConflict
				return &CodeError{
					Code:    "target_time_entries_exist",
					Message: "time entries already exist for the next week",
				}
			}

			sourceEntries, err := txApp.FindRecordsByFilter("time_entries", "uid={:userID} && tsid={:timeSheetID}", "date", 0, 0, dbx.Params{
				"userID":      userID,
				"timeSheetID": sourceID,
			})
			if err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_loading_time_entries",
					Message: fmt.Sprintf("error loading source time entries: %v", err),
				}
			}
			if len(sourceEntries) == 0 {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "source_time_sheet_empty",
					Message: "there are no time entries to copy from this time sheet",
				}
			}

			newRecordIDs := make([]string, 0, len(sourceEntries))
			for _, sourceEntry := range sourceEntries {
				copied, err := copyTimeEntryWithDayOffset(txApp, authRecord, sourceEntry, 7)
				if err != nil {
					httpResponseStatusCode = copyTimeEntryErrorStatus(err)
					return err
				}
				if copied.WeekEnding != targetWeekEnding {
					httpResponseStatusCode = http.StatusInternalServerError
					return &CodeError{
						Code:    "source_entry_week_mismatch",
						Message: "a source time entry date does not match the source time sheet week",
					}
				}
				newRecordIDs = append(newRecordIDs, copied.ID)
			}

			response = copyTimesheetEntriesResponse{
				Message:      "Time sheet entries copied to next week",
				CopiedCount:  len(newRecordIDs),
				NewRecordIDs: newRecordIDs,
				WeekEnding:   targetWeekEnding,
			}
			return nil
		})

		if err != nil {
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

		return e.JSON(http.StatusCreated, response)
	}
}
