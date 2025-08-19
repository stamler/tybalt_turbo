package routes

import (
	_ "embed" // Needed for //go:embed
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed job_divisions_summary.sql
var jobDivisionsSummaryQuery string

// JobDivisionSummaryRow models a single row from job_divisions_summary.sql
type JobDivisionSummaryRow struct {
	Number               string  `db:"number" json:"number"`
	DivisionCode         string  `db:"division_code" json:"division_code"`
	DivisionName         string  `db:"division_name" json:"division_name"`
	JobHours             float64 `db:"hours" json:"hours"`
	DivisionValueDollars float64 `db:"value" json:"value"`
	JobValueDollars      float64 `db:"job_value_dollars" json:"job_value_dollars"`
	DivisionValuePercent float64 `db:"percent" json:"percent"`
}

// createGetJobDivisionsSummaryHandler executes job_divisions_summary.sql for a job and date range
func createGetJobDivisionsSummaryHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.Error(http.StatusBadRequest, "id is required", nil)
		}

		q := e.Request.URL.Query()
		startDate := q.Get("start_date")
		endDate := q.Get("end_date")
		if startDate == "" || endDate == "" {
			return e.Error(http.StatusBadRequest, "start_date and end_date are required", nil)
		}

		params := dbx.Params{
			"job_id":     id,
			"start_date": startDate,
			"end_date":   endDate,
		}

		var rows []JobDivisionSummaryRow
		if err := app.DB().NewQuery(jobDivisionsSummaryQuery).Bind(params).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		return e.JSON(http.StatusOK, rows)
	}
}
