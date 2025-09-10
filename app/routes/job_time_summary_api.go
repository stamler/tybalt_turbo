package routes

import (
	"database/sql"
	_ "embed" // Needed for //go:embed
	"net/http"
	"strconv"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed job_time_summary.sql
var jobTimeSummaryQuery string

// summaryRow maps the single-row result of job_time_summary.sql.
type summaryRow struct {
	TotalHours    sql.NullString `db:"total_hours"`
	EarliestEntry sql.NullString `db:"earliest_entry"`
	LatestEntry   sql.NullString `db:"latest_entry"`
	Branches      sql.NullString `db:"branches"`
	Divisions     sql.NullString `db:"divisions"`
	TimeTypes     sql.NullString `db:"time_types"`
	Names         sql.NullString `db:"names"`
	Categories    sql.NullString `db:"categories"`
}

// createGetJobTimeSummaryHandler returns an HTTP handler that executes the
// job_time_summary.sql query for the requested job id and optional filters.
// The optional filters are provided as query parameters:
//   - division   (division id)
//   - time_type  (time type id)
//   - uid        (user id)
//   - category   (category id)
func createGetJobTimeSummaryHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.Error(http.StatusBadRequest, "id is required", nil)
		}

		q := e.Request.URL.Query()
		division := q.Get("division")
		branch := q.Get("branch")
		timeType := q.Get("time_type")
		uid := q.Get("uid")
		category := q.Get("category")

		var row summaryRow
		if err := app.DB().NewQuery(jobTimeSummaryQuery).Bind(dbx.Params{
			"id":        id,
			"branch":    branch,
			"division":  division,
			"time_type": timeType,
			"uid":       uid,
			"category":  category,
		}).One(&row); err != nil {
			if err == sql.ErrNoRows {
				return e.JSON(http.StatusOK, map[string]any{})
			}
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		// Helper to unwrap sql.NullString to string
		ns := func(n sql.NullString) string {
			if n.Valid {
				return n.String
			}
			return ""
		}

		// Convert total_hours to float64 if possible; otherwise 0
		var total float64
		if row.TotalHours.Valid {
			if f, err := strconv.ParseFloat(row.TotalHours.String, 64); err == nil {
				total = f
			}
		}

		resp := map[string]any{
			"total_hours":    total,
			"earliest_entry": ns(row.EarliestEntry),
			"latest_entry":   ns(row.LatestEntry),
			"branches":       ns(row.Branches),
			"divisions":      ns(row.Divisions),
			"time_types":     ns(row.TimeTypes),
			"names":          ns(row.Names),
			"categories":     ns(row.Categories),
		}

		return e.JSON(http.StatusOK, resp)
	}
}
