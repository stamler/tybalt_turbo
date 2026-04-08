// time_entry_branch_mismatches.go exposes a CSV diagnostics report for
// job-linked time entries whose stored branch differs from the current branch
// on the referenced job.
//
// Why this exists:
// New and updated time entries now force branch = job.branch whenever a job is
// present, so under normal application writes this mismatch should stop growing.
// This report is meant to help find legacy rows, imported rows, or other data
// that bypassed the normal time-entry hook path.
//
// What it reports:
// - rows where time_entries.job is present
// - rows where time_entries.branch is non-blank
// - rows where jobs.branch is non-blank
// - rows where time_entries.branch != jobs.branch
//
// What it does NOT report:
// - job-linked rows where time_entries.branch is blank
// - job-linked rows where jobs.branch is blank
//
// That omission is intentional for now because current hook behavior rejects
// job-linked writes when the job has no branch, and the main cleanup target is
// explicit branch drift between two populated branch fields. If we later want
// this report to act as a full "branch integrity" audit, we should expand the
// query to include blank-vs-nonblank mismatches too.
package routes

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"tybalt/errs"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
)

type timeEntryBranchMismatchRow struct {
	TimeEntryID         string  `db:"time_entry_id" json:"time_entry_id"`
	Date                string  `db:"date" json:"date"`
	WeekEnding          string  `db:"week_ending" json:"week_ending"`
	Hours               float64 `db:"hours" json:"hours"`
	UID                 string  `db:"uid" json:"uid"`
	EmployeeName        string  `db:"employee_name" json:"employee_name"`
	JobID               string  `db:"job_id" json:"job_id"`
	JobNumber           string  `db:"job_number" json:"job_number"`
	JobDescription      string  `db:"job_description" json:"job_description"`
	TimeEntryBranchID   string  `db:"time_entry_branch_id" json:"time_entry_branch_id"`
	TimeEntryBranchCode string  `db:"time_entry_branch_code" json:"time_entry_branch_code"`
	TimeEntryBranchName string  `db:"time_entry_branch_name" json:"time_entry_branch_name"`
	JobBranchID         string  `db:"job_branch_id" json:"job_branch_id"`
	JobBranchCode       string  `db:"job_branch_code" json:"job_branch_code"`
	JobBranchName       string  `db:"job_branch_name" json:"job_branch_name"`
	Description         string  `db:"description" json:"description"`
}

func requireTimeEntryBranchMismatchReportViewer(app core.App, auth *core.Record) error {
	hasReportClaim, err := utilities.HasClaim(app, auth, "report")
	if err != nil {
		return err
	}
	if hasReportClaim {
		return nil
	}

	return &errs.HookError{
		Status:  http.StatusForbidden,
		Message: "you are not authorized to view this report",
		Data: map[string]errs.CodeError{
			"global": {
				Code:    "unauthorized",
				Message: "you are not authorized to view this report",
			},
		},
	}
}

func createTimeEntryBranchMismatchesReportHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireTimeEntryBranchMismatchReportViewer(app, e.Auth); err != nil {
			return writeHookError(e, err)
		}

		query := `
				SELECT
					te.id AS time_entry_id,
					te.date,
					te.week_ending,
					COALESCE(te.hours, 0) AS hours,
					te.uid,
					TRIM(COALESCE(p.given_name, '') || ' ' || COALESCE(p.surname, '')) AS employee_name,
					j.id AS job_id,
				COALESCE(j.number, '') AS job_number,
				COALESCE(j.description, '') AS job_description,
				COALESCE(teb.id, '') AS time_entry_branch_id,
				COALESCE(teb.code, '') AS time_entry_branch_code,
				COALESCE(teb.name, '') AS time_entry_branch_name,
				COALESCE(jb.id, '') AS job_branch_id,
				COALESCE(jb.code, '') AS job_branch_code,
				COALESCE(jb.name, '') AS job_branch_name,
				COALESCE(te.description, '') AS description
			FROM time_entries te
			INNER JOIN jobs j ON j.id = te.job
			LEFT JOIN profiles p ON p.uid = te.uid
			LEFT JOIN branches teb ON teb.id = te.branch
			LEFT JOIN branches jb ON jb.id = j.branch
			WHERE te.job != ''
			  AND te.branch != ''
			  AND j.branch != ''
			  AND te.branch != j.branch
			ORDER BY te.date DESC, employee_name ASC, job_number ASC, te.id ASC
		`

		var rows []timeEntryBranchMismatchRow
		if err := app.DB().NewQuery(query).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query", err)
		}

		var csvBuilder strings.Builder
		writer := csv.NewWriter(&csvBuilder)
		headers := []string{
			"time_entry_id",
			"date",
			"week_ending",
			"hours",
			"uid",
			"employee_name",
			"job_id",
			"job_number",
			"job_description",
			"time_entry_branch_id",
			"time_entry_branch_code",
			"time_entry_branch_name",
			"job_branch_id",
			"job_branch_code",
			"job_branch_name",
			"description",
		}
		if err := writer.Write(headers); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to write csv header", err)
		}
		for _, row := range rows {
			if err := writer.Write([]string{
				row.TimeEntryID,
				row.Date,
				row.WeekEnding,
				strconv.FormatFloat(row.Hours, 'f', -1, 64),
				row.UID,
				row.EmployeeName,
				row.JobID,
				row.JobNumber,
				row.JobDescription,
				row.TimeEntryBranchID,
				row.TimeEntryBranchCode,
				row.TimeEntryBranchName,
				row.JobBranchID,
				row.JobBranchCode,
				row.JobBranchName,
				row.Description,
			}); err != nil {
				return e.Error(http.StatusInternalServerError, "failed to write csv row", err)
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to finalize csv", err)
		}

		e.Response.Header().Set("Content-Type", "text/csv; charset=utf-8")
		e.Response.Header().Set("Content-Disposition", `attachment; filename="time_entry_branch_mismatches.csv"`)
		return e.String(http.StatusOK, csvBuilder.String())
	}
}
