package routes

import (
	"errors"
	"net/http"
	"strings"
	"tybalt/errs"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type timeSheetMissingExceptionResponse struct {
	UID        string `json:"uid"`
	WeekEnding string `json:"week_ending"`
	Ignored    bool   `json:"ignored"`
}

func requireTimeTrackingViewer(app core.App, auth *core.Record) error {
	isReportHolder, err := utilities.HasClaim(app, auth, "report")
	if err != nil {
		return err
	}
	if isReportHolder {
		return nil
	}

	isCommitter, err := utilities.HasClaim(app, auth, "commit")
	if err != nil {
		return err
	}
	if isCommitter {
		return nil
	}

	return &errs.HookError{
		Status:  http.StatusForbidden,
		Message: "you are not authorized to view time tracking",
		Data: map[string]errs.CodeError{
			"global": {
				Code:    "unauthorized",
				Message: "you are not authorized to view time tracking",
			},
		},
	}
}

func requireCommitClaim(app core.App, auth *core.Record) error {
	hasCommitClaim, err := utilities.HasClaim(app, auth, "commit")
	if err != nil {
		return err
	}
	if !hasCommitClaim {
		return &errs.HookError{
			Status:  http.StatusForbidden,
			Message: "you are not authorized to update missing timesheet exceptions",
			Data: map[string]errs.CodeError{
				"global": {
					Code:    "unauthorized",
					Message: "you are not authorized to update missing timesheet exceptions",
				},
			},
		}
	}
	return nil
}

func validateWeekEndingParam(weekEnding string) error {
	if weekEnding == "" {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "invalid week ending",
			Data: map[string]errs.CodeError{
				"week_ending": {
					Code:    "required",
					Message: "weekEnding is required",
				},
			},
		}
	}
	if !isValidWeekEnding(weekEnding) {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "invalid week ending",
			Data: map[string]errs.CodeError{
				"week_ending": {
					Code:    "invalid_format",
					Message: "weekEnding must be in YYYY-MM-DD format",
				},
			},
		}
	}
	return nil
}

func validateUserExists(app core.App, uid string) error {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "invalid uid",
			Data: map[string]errs.CodeError{
				"uid": {
					Code:    "required",
					Message: "uid is required",
				},
			},
		}
	}

	if _, err := app.FindRecordById("users", uid); err != nil {
		return &errs.HookError{
			Status:  http.StatusNotFound,
			Message: "user not found",
			Data: map[string]errs.CodeError{
				"uid": {
					Code:    "not_found",
					Message: "user not found",
				},
			},
		}
	}
	return nil
}

func createMissingTimesheetExceptionHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireCommitClaim(app, e.Auth); err != nil {
			return writeHookError(e, err)
		}

		weekEnding := e.Request.PathValue("weekEnding")
		if err := validateWeekEndingParam(weekEnding); err != nil {
			return writeHookError(e, err)
		}

		uid := e.Request.PathValue("uid")
		if err := validateUserExists(app, uid); err != nil {
			return writeHookError(e, err)
		}

		response := timeSheetMissingExceptionResponse{
			UID:        uid,
			WeekEnding: weekEnding,
			Ignored:    true,
		}

		err := app.RunInTransaction(func(txApp core.App) error {
			_, err := txApp.NonconcurrentDB().NewQuery(`
				INSERT OR IGNORE INTO time_sheet_missing_exceptions (
					uid,
					week_ending,
					created_by,
					created,
					updated
				) VALUES (
					{:uid},
					{:week_ending},
					{:created_by},
					strftime('%Y-%m-%d %H:%M:%fZ', 'now'),
					strftime('%Y-%m-%d %H:%M:%fZ', 'now')
				)
			`).Bind(dbx.Params{
				"uid":         uid,
				"week_ending": weekEnding,
				"created_by":  e.Auth.Id,
			}).Execute()
			return err
		})
		if err != nil {
			return writeHookError(e, err)
		}

		return e.JSON(http.StatusOK, response)
	}
}

func createDeleteMissingTimesheetExceptionHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireCommitClaim(app, e.Auth); err != nil {
			return writeHookError(e, err)
		}

		weekEnding := e.Request.PathValue("weekEnding")
		if err := validateWeekEndingParam(weekEnding); err != nil {
			return writeHookError(e, err)
		}

		uid := e.Request.PathValue("uid")
		if err := validateUserExists(app, uid); err != nil {
			return writeHookError(e, err)
		}

		err := app.RunInTransaction(func(txApp core.App) error {
			_, err := txApp.NonconcurrentDB().NewQuery(`
				DELETE FROM time_sheet_missing_exceptions
				WHERE uid = {:uid} AND week_ending = {:week_ending}
			`).Bind(dbx.Params{
				"uid":         uid,
				"week_ending": weekEnding,
			}).Execute()
			return err
		})
		if err != nil {
			return writeHookError(e, err)
		}

		return e.JSON(http.StatusOK, timeSheetMissingExceptionResponse{
			UID:        uid,
			WeekEnding: weekEnding,
			Ignored:    false,
		})
	}
}

func createTimesheetIgnoredHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireTimeTrackingViewer(app, e.Auth); err != nil {
			return writeHookError(e, err)
		}

		weekEnding := e.Request.PathValue("weekEnding")
		if err := validateWeekEndingParam(weekEnding); err != nil {
			return writeHookError(e, err)
		}

		query := `
            SELECT
                u.id AS id,
                COALESCE(p.given_name, '') AS given_name,
                COALESCE(p.surname, '') AS surname,
                COALESCE(u.email, '') AS email
            FROM time_sheet_missing_exceptions ex
            JOIN users u ON u.id = ex.uid
            LEFT JOIN time_sheets ts ON ts.uid = u.id AND ts.week_ending = ex.week_ending
            LEFT JOIN profiles p ON p.uid = u.id
            LEFT JOIN admin_profiles ap ON ap.uid = u.id
            WHERE ex.week_ending = {:week_ending}
              AND ts.id IS NULL
              AND COALESCE(ap.time_sheet_expected, 0) = 1
            ORDER BY p.surname, p.given_name
        `

		var rows []trackingBasicUserRow
		if err := app.DB().NewQuery(query).Bind(dbx.Params{
			"week_ending": weekEnding,
		}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query", err)
		}

		return e.JSON(http.StatusOK, rows)
	}
}

func writeHookError(e *core.RequestEvent, err error) error {
	var hookErr *errs.HookError
	if errors.As(err, &hookErr) {
		return e.JSON(hookErr.Status, hookErr)
	}
	return e.Error(http.StatusInternalServerError, "unexpected error", err)
}
