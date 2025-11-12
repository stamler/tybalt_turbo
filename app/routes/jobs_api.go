package routes

import (
	_ "embed" // Needed for //go:embed
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// NOTE: When you use PocketBase's `expand:"categories_via_job,client"` the
// generated SQL executes one separate query per parent record for each
// back-reference (the *_via_* relations). With thousands of jobs this turns into
// an N+1 query pattern that kills performance. Consolidating the data in a
// single SQL below eliminates that overhead.

//go:embed jobs.sql
var jobsQuery string

//go:embed jobs_latest.sql
var jobsLatestQuery string

//go:embed jobs_unused.sql
var jobsUnusedQuery string

//go:embed jobs_stale.sql
var jobsStaleQuery string

// JobWithRelations models the JSON returned by /api/jobs
// The Categories field is unmarshalled from the JSON returned by SQLite.
// We keep field names in snake_case to stay consistent with SQL aliases and
// existing frontend expectations.
// We return the client name as "client" plus its id as "client_id" so the UI
// doesn't need a second request or expand.
//
// The struct purposefully omits PocketBase system fields because the frontend
// doesn't need them for listing/searching.

type Job struct {
	ID                     string  `db:"id" json:"id"`
	Number                 string  `db:"number" json:"number"`
	Description            string  `db:"description" json:"description"`
	Location               string  `db:"location" json:"location"`
	ClientID               string  `db:"client_id" json:"client_id"`
	Client                 string  `db:"client" json:"client"`
	OutstandingBalance     float64 `db:"outstanding_balance" json:"outstanding_balance"`
	OutstandingBalanceDate string  `db:"outstanding_balance_date" json:"outstanding_balance_date"`
}

// latestJobsLimit controls how many proposals and how many projects are returned.
const latestJobsLimit = 20

// LatestJob models a latest job row with grouping label.
type LatestJob struct {
	Job
	GroupName string `db:"group_name" json:"group_name"`
}

// StaleJob augments Job with last_reference date for the stale endpoint
type StaleJob struct {
	Job
	LastReference string `db:"last_reference" json:"last_reference"`
	LastRefType   string `db:"last_reference_type" json:"last_reference_type"`
}

func createGetJobsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id") // may be "" when listing

		// Execute the SQL with optional :id binding
		var rows []Job
		if err := app.DB().NewQuery(jobsQuery).Bind(dbx.Params{"id": id}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		// Convert to response structs
		makeResp := func(r Job) Job { return r }

		if id != "" {
			if len(rows) == 0 {
				return e.Error(http.StatusNotFound, "job not found", nil)
			}
			return e.JSON(http.StatusOK, makeResp(rows[0]))
		}

		resp := make([]Job, len(rows))
		for i, r := range rows {
			resp[i] = makeResp(r)
		}
		return e.JSON(http.StatusOK, resp)
	}
}

// createGetLatestJobsHandler returns the latest proposals and projects.
func createGetLatestJobsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Authorization: must hold the 'job' claim
		hasJobClaim, err := utilities.HasClaim(app, e.Auth, "job")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "error checking claims", err)
		}
		if !hasJobClaim {
			return e.Error(http.StatusForbidden, "you are not authorized to view latest jobs", nil)
		}

		var rows []LatestJob
		if err := app.DB().NewQuery(jobsLatestQuery).Bind(dbx.Params{
			"limit": latestJobsLimit,
		}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		return e.JSON(http.StatusOK, rows)
	}
}

// createGetUnusedJobsHandler returns zero-use jobs matching a job number prefix
func createGetUnusedJobsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Prefer query param ?prefix=..., fallback to optional path value if present
		prefix := strings.TrimSpace(e.Request.URL.Query().Get("prefix"))
		if prefix == "" {
			prefix = strings.TrimSpace(e.Request.PathValue("prefix"))
		}
		if prefix == "" {
			// Default to current year's proposal prefix e.g. P25-
			yy := time.Now().UTC().Year() % 100
			prefix = fmt.Sprintf("P%02d", yy)
		}

		limit := 50
		if q := e.Request.URL.Query().Get("limit"); q != "" {
			if v, err := strconv.Atoi(q); err == nil && v > 0 {
				limit = v
			}
		}

		// Reuse the stale query with age=0 to return unused jobs (no references)
		var rows []StaleJob
		if err := app.DB().NewQuery(jobsStaleQuery).Bind(dbx.Params{
			"prefix": prefix,
			"age":    0,
			"limit":  limit,
		}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}
		return e.JSON(http.StatusOK, rows)
	}
}

// createGetStaleJobsHandler returns stale jobs matching a job number prefix and age (days)
func createGetStaleJobsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// prefix default to current PYY if not provided
		prefix := strings.TrimSpace(e.Request.URL.Query().Get("prefix"))
		if prefix == "" {
			prefix = strings.TrimSpace(e.Request.PathValue("prefix"))
		}
		if prefix == "" {
			yy := time.Now().UTC().Year() % 100
			prefix = fmt.Sprintf("P%02d", yy)
		}

		// age in days, default 180
		age := 180
		if q := e.Request.URL.Query().Get("age"); q != "" {
			if v, err := strconv.Atoi(q); err == nil && v > 0 {
				age = v
			}
		}

		limit := 50
		if q := e.Request.URL.Query().Get("limit"); q != "" {
			if v, err := strconv.Atoi(q); err == nil && v > 0 {
				limit = v
			}
		}

		// Execute
		var rows []StaleJob
		if err := app.DB().NewQuery(jobsStaleQuery).Bind(dbx.Params{
			"prefix": prefix,
			"age":    age,
			"limit":  limit,
		}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}
		return e.JSON(http.StatusOK, rows)
	}
}
