package routes

import (
	_ "embed"
	"net/http"
	"strings"
	"tybalt/errs"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed work_records_list.sql
var workRecordsListQuery string

//go:embed work_records_details.sql
var workRecordsDetailsQuery string

type WorkRecordSearchRow struct {
	WorkRecord string `db:"work_record" json:"work_record"`
	Prefix     string `db:"prefix" json:"prefix"`
	EntryCount int    `db:"entry_count" json:"entry_count"`
	SearchText string `json:"search_text"`
}

type WorkRecordEntryRow struct {
	ID          string  `db:"id" json:"id"`
	WorkRecord  string  `db:"work_record" json:"work_record"`
	WeekEnding  string  `db:"week_ending" json:"week_ending"`
	UID         string  `db:"uid" json:"uid"`
	Hours       float64 `db:"hours" json:"hours"`
	JobNumber   string  `db:"job_number" json:"job_number"`
	JobID       string  `db:"job_id" json:"job_id"`
	Description string  `db:"description" json:"description"`
	Surname     string  `db:"surname" json:"surname"`
	GivenName   string  `db:"given_name" json:"given_name"`
	TimesheetID string  `db:"timesheet_id" json:"timesheet_id"`
}

func requireWorkRecordViewer(app core.App, auth *core.Record) error {
	isReportHolder, err := utilities.HasClaim(app, auth, "report")
	if err != nil {
		return err
	}
	if isReportHolder {
		return nil
	}

	isWorkRecordHolder, err := utilities.HasClaim(app, auth, "work_record")
	if err != nil {
		return err
	}
	if isWorkRecordHolder {
		return nil
	}

	return &errs.HookError{
		Status:  http.StatusForbidden,
		Message: "you are not authorized to view work records",
		Data: map[string]errs.CodeError{
			"global": {
				Code:    "unauthorized",
				Message: "you are not authorized to view work records",
			},
		},
	}
}

func createGetWorkRecordsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireWorkRecordViewer(app, e.Auth); err != nil {
			return writeHookError(e, err)
		}

		var rows []WorkRecordSearchRow
		if err := app.DB().NewQuery(workRecordsListQuery).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute work records query", err)
		}

		for i := range rows {
			rows[i].SearchText = buildWorkRecordSearchText(rows[i].WorkRecord)
		}

		return e.JSON(http.StatusOK, rows)
	}
}

func createGetWorkRecordDetailsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireWorkRecordViewer(app, e.Auth); err != nil {
			return writeHookError(e, err)
		}

		workRecord := strings.TrimSpace(e.Request.PathValue("workRecord"))
		if workRecord == "" {
			return e.Error(http.StatusBadRequest, "workRecord is required", nil)
		}

		var rows []WorkRecordEntryRow
		if err := app.DB().NewQuery(workRecordsDetailsQuery).Bind(dbx.Params{
			"work_record": workRecord,
		}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute work record details query", err)
		}

		return e.JSON(http.StatusOK, rows)
	}
}

func buildWorkRecordSearchText(workRecord string) string {
	upper := strings.ToUpper(strings.TrimSpace(workRecord))
	if upper == "" {
		return ""
	}

	parts := make([]string, 0, 6)
	add := func(token string) {
		token = strings.TrimSpace(token)
		if token == "" {
			return
		}
		for _, existing := range parts {
			if existing == token {
				return
			}
		}
		parts = append(parts, token)
	}

	add(upper)
	add(strings.ReplaceAll(upper, "-", ""))

	if len(upper) > 1 {
		withoutPrefix := upper[1:]
		add(withoutPrefix)
		add(strings.ReplaceAll(withoutPrefix, "-", ""))

		pieces := strings.SplitN(withoutPrefix, "-", 2)
		if len(pieces) == 2 {
			add(pieces[0])
			add(pieces[1])
			add(pieces[0] + pieces[1])
		}
	}

	return strings.Join(parts, " ")
}
