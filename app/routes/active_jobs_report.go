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

type activeJobReportRow struct {
	Status           string `db:"status" json:"status"`
	JobNumber        string `db:"job_number" json:"job_number"`
	ManagerName      string `db:"manager_name" json:"manager_name"`
	BranchName       string `db:"branch_name" json:"branch_name"`
	TimeAndMaterials bool   `db:"time_and_materials" json:"time_and_materials"`
}

func requireReportViewer(app core.App, auth *core.Record) error {
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

func createActiveJobsReportHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireReportViewer(app, e.Auth); err != nil {
			return writeHookError(e, err)
		}

		query := `
			SELECT
				j.status AS status,
				COALESCE(j.number, '') AS job_number,
				TRIM(COALESCE(m.given_name, '') || ' ' || COALESCE(m.surname, '')) AS manager_name,
				COALESCE(b.name, '') AS branch_name,
				COALESCE(j.time_and_materials, 0) AS time_and_materials
			FROM jobs j
			LEFT JOIN profiles m ON m.uid = j.manager
			LEFT JOIN branches b ON b.id = j.branch
			WHERE j.status = 'Active'
			  AND j.number NOT LIKE 'P%'
			ORDER BY j.number DESC
		`

		var rows []activeJobReportRow
		if err := app.DB().NewQuery(query).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query", err)
		}

		var csvBuilder strings.Builder
		writer := csv.NewWriter(&csvBuilder)
		headers := []string{
			"status",
			"job_number",
			"manager_name",
			"branch_name",
			"time_and_materials",
		}
		if err := writer.Write(headers); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to write csv header", err)
		}
		for _, row := range rows {
			if err := writer.Write([]string{
				row.Status,
				row.JobNumber,
				row.ManagerName,
				row.BranchName,
				strconv.FormatBool(row.TimeAndMaterials),
			}); err != nil {
				return e.Error(http.StatusInternalServerError, "failed to write csv row", err)
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to finalize csv", err)
		}

		e.Response.Header().Set("Content-Type", "text/csv; charset=utf-8")
		e.Response.Header().Set("Content-Disposition", `attachment; filename="active_jobs.csv"`)
		return e.String(http.StatusOK, csvBuilder.String())
	}
}
