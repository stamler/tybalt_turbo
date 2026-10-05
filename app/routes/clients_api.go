package routes

import (
	"database/sql"
	_ "embed" // for go:embed
	"encoding/json"
	"errors"
	"net/http"

	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// NOTE: Using PocketBase expand "client_contacts_via_client" incurs N+1 queries.
// This endpoint delivers clients together with their contacts in a single SQL.

//go:embed clients.sql
var clientsQuery string

//go:embed client_details.sql
var clientDetailsQuery string

//go:embed busdev_leads.sql
var busdevLeadsQuery string

// Contact is a minimal subset of fields we need on the client list page.
type Contact struct {
	ID        string `json:"id"`
	GivenName string `json:"given_name"`
	Surname   string `json:"surname"`
	Email     string `json:"email"`
}

type clientRow struct {
	ID                   string  `db:"id"`
	Name                 string  `db:"name"`
	ContactsJSON         string  `db:"contacts_json"`
	ReferencingJobsCount int     `db:"referencing_jobs_count"`
	OutstandingBalance   float64 `db:"outstanding_balance"`
}

type Client struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Contacts             []Contact `json:"contacts"`
	ReferencingJobsCount int       `json:"referencing_jobs_count"`
	OutstandingBalance   float64   `json:"outstanding_balance"`
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
	Address                string  `db:"address"`
	City                   string  `db:"city"`
	ProvinceState          string  `db:"province_state"`
	PostalCode             string  `db:"postal_code"`
	Country                string  `db:"country"`
	Phone                  string  `db:"phone"`
}

// ClientContact includes both project use and use through an invoicing profile.
// Each job counts once even when it uses the contact in both roles.
type ClientContact struct {
	Contact
	Client        string `json:"client"`
	Address       string `json:"address"`
	City          string `json:"city"`
	ProvinceState string `json:"province_state"`
	PostalCode    string `json:"postal_code"`
	Country       string `json:"country"`
	Phone         string `json:"phone"`
	JobCount      int    `json:"job_count"`
	ProfileCount  int    `json:"profile_count"`
}

type ClientInvoicingProfile struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	Client                string `json:"client"`
	Contact               string `json:"contact"`
	Fax                   string `json:"fax"`
	InvoicingInstructions string `json:"invoicing_instructions"`
	JobCount              int    `json:"job_count"`
	Creator               string `json:"creator"`
	CreatorName           string `json:"creator_name"`
	Created               string `json:"created"`
}

type ClientDetails struct {
	ID                     string                   `json:"id"`
	Name                   string                   `json:"name"`
	BusinessDevelopmentUID string                   `json:"business_development_lead"`
	LeadGivenName          string                   `json:"lead_given_name"`
	LeadSurname            string                   `json:"lead_surname"`
	LeadEmail              string                   `json:"lead_email"`
	OutstandingBalance     float64                  `json:"outstanding_balance"`
	OutstandingBalanceDate string                   `json:"outstanding_balance_date"`
	Contacts               []ClientContact          `json:"contacts"`
	InvoicingProfiles      []ClientInvoicingProfile `json:"invoicing_profiles"`
	ReferencingJobsCount   int                      `json:"referencing_jobs_count"`
	Address                string                   `json:"address"`
	City                   string                   `json:"city"`
	ProvinceState          string                   `json:"province_state"`
	PostalCode             string                   `json:"postal_code"`
	Country                string                   `json:"country"`
	Phone                  string                   `json:"phone"`
}

type BusdevLead struct {
	ID        string `json:"id"`
	GivenName string `json:"given_name"`
	Surname   string `json:"surname"`
	Email     string `json:"email"`
}

func createGetBusdevLeadsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var rows []BusdevLead
		if err := app.DB().NewQuery(busdevLeadsQuery).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}
		return e.JSON(http.StatusOK, rows)
	}
}

func createGetClientsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id != "" {
			// Details include invoicing profiles, whose collection read rule requires
			// an active account. Keep the same rule for this combined response.
			active, err := utilities.IsUserActive(app, e.Auth.Id)
			if err != nil {
				return e.Error(http.StatusInternalServerError, "failed to check account status", err)
			}
			if !active {
				return e.Error(http.StatusForbidden, "an active account is required to view client details", nil)
			}
			details, err := queryClientDetails(app, id)
			if errors.Is(err, sql.ErrNoRows) {
				return e.Error(http.StatusNotFound, "client not found", nil)
			}
			if err != nil {
				return e.Error(http.StatusInternalServerError, "failed to load client details", err)
			}
			return e.JSON(http.StatusOK, details)
		}

		var rows []clientRow
		if err := app.DB().NewQuery(clientsQuery).Bind(dbx.Params{"id": id}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		toClient := func(r clientRow) Client {
			var contacts []Contact
			_ = json.Unmarshal([]byte(r.ContactsJSON), &contacts)
			return Client{
				ID:                   r.ID,
				Name:                 r.Name,
				Contacts:             contacts,
				ReferencingJobsCount: r.ReferencingJobsCount,
				OutstandingBalance:   r.OutstandingBalance,
			}
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
		ContactsJSON          string `db:"contacts_json"`
		InvoicingProfilesJSON string `db:"invoicing_profiles_json"`
	}

	if err := app.DB().NewQuery(clientDetailsQuery).Bind(dbx.Params{"id": id}).One(&row); err != nil {
		return nil, err
	}

	var contacts []ClientContact
	if row.ContactsJSON != "" {
		if err := json.Unmarshal([]byte(row.ContactsJSON), &contacts); err != nil {
			return nil, err
		}
	}

	// When no contacts exist we still want an empty slice in the JSON response, not null.
	if contacts == nil {
		contacts = []ClientContact{}
	}
	profiles := []ClientInvoicingProfile{}
	if row.InvoicingProfilesJSON != "" {
		if err := json.Unmarshal([]byte(row.InvoicingProfilesJSON), &profiles); err != nil {
			return nil, err
		}
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
		InvoicingProfiles:      profiles,
		ReferencingJobsCount:   row.ReferencingJobsCount,
		Address:                row.Address,
		City:                   row.City,
		ProvinceState:          row.ProvinceState,
		PostalCode:             row.PostalCode,
		Country:                row.Country,
		Phone:                  row.Phone,
	}, nil
}
