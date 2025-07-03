package routes

import (
	_ "embed" // Needed for //go:embed
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed job_time_summary.sql
var jobTimeSummaryQuery string

// createGetJobTimeSummaryHandler returns an HTTP handler that executes the
// job_time_summary.sql query for the requested job id and optional filters.
// The optional filters are provided as query parameters:
//   - division   (division id)
//   - time_type  (time type id)
//   - uid        (user id)
//   - category   (category id)
func createGetJobTimeSummaryHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Path parameter is always present in the route pattern
		id := e.Request.PathValue("id")
		if id == "" {
			return e.Error(http.StatusBadRequest, "id is required", nil)
		}

		// Optional query parameters for additional filtering
		q := e.Request.URL.Query()
		division := q.Get("division")
		timeType := q.Get("time_type")
		uid := q.Get("uid")
		category := q.Get("category")

		// Execute the SQL query with bound parameters
		var rows []dbx.NullStringMap
		err := app.DB().NewQuery(jobTimeSummaryQuery).Bind(dbx.Params{
			"id":        id,
			"division":  division,
			"time_type": timeType,
			"uid":       uid,
			"category":  category,
		}).All(&rows)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		// The query returns a single aggregated row.
		if len(rows) == 0 {
			return e.JSON(http.StatusOK, map[string]any{})
		}
		return e.JSON(http.StatusOK, rows[0])
	}
}
