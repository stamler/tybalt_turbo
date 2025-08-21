package routes

import (
	_ "embed" // Needed for //go:embed
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed job_staff_summary.sql
var jobStaffSummaryQuery string

// JobStaffSummaryRow models a single row from job_staff_summary.sql
type JobStaffSummaryRow struct {
	Number     string  `db:"number" json:"number"`
	GivenName  string  `db:"given_name" json:"given_name"`
	Surname    string  `db:"surname" json:"surname"`
	Hours      float64 `db:"hours" json:"hours"`
	Value      float64 `db:"value" json:"value"`
	Total      float64 `db:"total" json:"total"`
	Percent    float64 `db:"percent" json:"percent"`
	MealsHours float64 `db:"meals_hours" json:"meals_hours"`
	UID        string  `db:"uid" json:"uid"`
}

// createGetJobStaffSummaryHandler executes job_staff_summary.sql for a job and date range
func createGetJobStaffSummaryHandler(app core.App) func(e *core.RequestEvent) error {
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

		var rows []JobStaffSummaryRow
		if err := app.DB().NewQuery(jobStaffSummaryQuery).Bind(params).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		return e.JSON(http.StatusOK, rows)
	}
}
