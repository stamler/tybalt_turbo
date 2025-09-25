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

//go:embed client_details.sql
var clientDetailsQuery string

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

type clientDetailsRow struct {
	ID                     string  `db:"id"`
	Name                   string  `db:"name"`
	BusinessDevelopmentUID string  `db:"business_development_lead"`
	OutstandingBalance     float64 `db:"outstanding_balance"`
	OutstandingBalanceDate string  `db:"outstanding_balance_date"`
	LeadGivenName          string  `db:"lead_given_name"`
	LeadSurname            string  `db:"lead_surname"`
	LeadEmail              string  `db:"lead_email"`
	ReferencingJobsCount   int     `db:"referencing_jobs_count"`
}

type ClientDetails struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	BusinessDevelopmentUID string    `json:"business_development_lead"`
	LeadGivenName          string    `json:"lead_given_name"`
	LeadSurname            string    `json:"lead_surname"`
	LeadEmail              string    `json:"lead_email"`
	OutstandingBalance     float64   `json:"outstanding_balance"`
	OutstandingBalanceDate string    `json:"outstanding_balance_date"`
	Contacts               []Contact `json:"contacts"`
	ReferencingJobsCount   int       `json:"referencing_jobs_count"`
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
			details, err := queryClientDetails(app, id)
			if err != nil {
				return e.Error(http.StatusInternalServerError, "failed to load client details", err)
			}
			return e.JSON(http.StatusOK, details)
		}

		resp := make([]Client, len(rows))
		for i, r := range rows {
			resp[i] = toClient(r)
		}
		return e.JSON(http.StatusOK, resp)
	}
}

func queryClientDetails(app core.App, id string) (*ClientDetails, error) {
	var row struct {
		clientDetailsRow
		ContactsJSON string `db:"contacts_json"`
	}

	if err := app.DB().NewQuery(clientDetailsQuery).Bind(dbx.Params{"id": id}).One(&row); err != nil {
		return nil, err
	}

	var contacts []Contact
	if row.ContactsJSON != "" {
		if err := json.Unmarshal([]byte(row.ContactsJSON), &contacts); err != nil {
			return nil, err
		}
	}

	// When no contacts exist we still want an empty slice in the JSON response, not null.
	if contacts == nil {
		contacts = []Contact{}
	}

	return &ClientDetails{
		ID:                     row.ID,
		Name:                   row.Name,
		BusinessDevelopmentUID: row.BusinessDevelopmentUID,
		LeadGivenName:          row.LeadGivenName,
		LeadSurname:            row.LeadSurname,
		LeadEmail:              row.LeadEmail,
		OutstandingBalance:     row.OutstandingBalance,
		OutstandingBalanceDate: row.OutstandingBalanceDate,
		Contacts:               contacts,
		ReferencingJobsCount:   row.ReferencingJobsCount,
	}, nil
}
