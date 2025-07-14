package routes

import (
	_ "embed" // for go:embed
	"encoding/json"
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// NOTE: Using PocketBase expand "client_contacts_via_client" incurs N+1 queries.
// This endpoint delivers clients together with their contacts in a single SQL.

//go:embed clients.sql
var clientsQuery string

// Contact is a minimal subset of fields we need on the client list page.
type Contact struct {
	ID        string `json:"id"`
	GivenName string `json:"given_name"`
	Surname   string `json:"surname"`
	Email     string `json:"email"`
}

type clientRow struct {
	ID                   string `db:"id"`
	Name                 string `db:"name"`
	ContactsJSON         string `db:"contacts_json"`
	ReferencingJobsCount int    `db:"referencing_jobs_count"`
}

type Client struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Contacts             []Contact `json:"contacts"`
	ReferencingJobsCount int       `json:"referencing_jobs_count"`
}

func createGetClientsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")

		var rows []clientRow
		if err := app.DB().NewQuery(clientsQuery).Bind(dbx.Params{"id": id}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		toClient := func(r clientRow) Client {
			var contacts []Contact
			_ = json.Unmarshal([]byte(r.ContactsJSON), &contacts)
			return Client{ID: r.ID, Name: r.Name, Contacts: contacts, ReferencingJobsCount: r.ReferencingJobsCount}
		}

		if id != "" {
			if len(rows) == 0 {
				return e.Error(http.StatusNotFound, "client not found", nil)
			}
			return e.JSON(http.StatusOK, toClient(rows[0]))
		}

		resp := make([]Client, len(rows))
		for i, r := range rows {
			resp[i] = toClient(r)
		}
		return e.JSON(http.StatusOK, resp)
	}
}
