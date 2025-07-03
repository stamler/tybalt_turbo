package routes

import (
	_ "embed" // Needed for //go:embed
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed job_time_entries.sql
var jobTimeEntriesQuery string

// JobTimeEntry models a single time entry row returned by job_time_entries.sql.
// Field names follow snake_case to match SQL aliases and frontend expectations.
// Numeric and date fields are represented as strings to simplify scanning and
// JSON marshaling without custom null handling.
// Adjust types later if stricter typing is required.
type JobTimeEntry struct {
	Description  string  `db:"description" json:"description"`
	Hours        float64 `db:"hours" json:"hours"`
	ID           string  `db:"id" json:"id"`
	WorkRecord   string  `db:"work_record" json:"work_record"`
	Date         string  `db:"date" json:"date"`
	WeekEnding   string  `db:"week_ending" json:"week_ending"`
	TSID         string  `db:"tsid" json:"tsid"`
	DivisionCode string  `db:"division_code" json:"division_code"`
	TimeTypeCode string  `db:"time_type_code" json:"time_type_code"`
	Surname      string  `db:"surname" json:"surname"`
	GivenName    string  `db:"given_name" json:"given_name"`
	CategoryName string  `db:"category_name" json:"category_name"`
}

// createGetJobTimeEntriesHandler returns an HTTP handler that fetches the time
// entries for a job with optional filters. Supported query parameters:
//   - division   (division id)
//   - time_type  (time type id)
//   - uid        (user id)
//   - category   (category id)
func createGetJobTimeEntriesHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.Error(http.StatusBadRequest, "id is required", nil)
		}

		q := e.Request.URL.Query()
		division := q.Get("division")
		timeType := q.Get("time_type")
		uid := q.Get("uid")
		category := q.Get("category")

		var rows []JobTimeEntry
		err := app.DB().NewQuery(jobTimeEntriesQuery).Bind(dbx.Params{
			"id":        id,
			"division":  division,
			"time_type": timeType,
			"uid":       uid,
			"category":  category,
		}).All(&rows)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		return e.JSON(http.StatusOK, rows)
	}
}
