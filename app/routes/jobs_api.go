package routes

import (
	_ "embed" // Needed for //go:embed
	"net/http"

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
	ID          string `db:"id" json:"id"`
	Number      string `db:"number" json:"number"`
	Description string `db:"description" json:"description"`
	ClientID    string `db:"client_id" json:"client_id"`
	Client      string `db:"client" json:"client"`
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
