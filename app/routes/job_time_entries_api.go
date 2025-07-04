package routes

import (
	_ "embed" // Needed for //go:embed
	"net/http"
	"strconv"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed job_time_entries.sql
var jobTimeEntriesQuery string

//go:embed job_time_entries_count.sql
var jobTimeEntriesCountQuery string

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

// PaginatedJobTimeEntriesResponse represents the paginated response structure
type PaginatedJobTimeEntriesResponse struct {
	Data       []JobTimeEntry `json:"data"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	Total      int            `json:"total"`
	TotalPages int            `json:"total_pages"`
}

// createGetJobTimeEntriesHandler returns an HTTP handler that fetches the time
// entries for a job with optional filters and pagination. Supported query parameters:
//   - division   (division id)
//   - time_type  (time type id)
//   - uid        (user id)
//   - category   (category id)
//   - page       (page number, default: 1)
//   - limit      (page size, default: 50, max: 200)
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

		// Parse pagination parameters
		page := 1
		if pageStr := q.Get("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		limit := 50 // default page size
		if limitStr := q.Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				if l > 200 {
					l = 200 // max page size
				}
				limit = l
			}
		}

		offset := (page - 1) * limit

		params := dbx.Params{
			"id":        id,
			"division":  division,
			"time_type": timeType,
			"uid":       uid,
			"category":  category,
			"limit":     limit,
			"offset":    offset,
		}

		// Get total count for pagination metadata
		var totalCount int
		err := app.DB().NewQuery(jobTimeEntriesCountQuery).Bind(params).Row(&totalCount)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute count query: "+err.Error(), err)
		}

		// Get paginated results
		var rows []JobTimeEntry
		err = app.DB().NewQuery(jobTimeEntriesQuery).Bind(params).All(&rows)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		totalPages := (totalCount + limit - 1) / limit // ceiling division

		response := PaginatedJobTimeEntriesResponse{
			Data:       rows,
			Page:       page,
			Limit:      limit,
			Total:      totalCount,
			TotalPages: totalPages,
		}

		return e.JSON(http.StatusOK, response)
	}
}
