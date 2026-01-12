package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const JOBS_MACHINE_SECRET_ID = "legacy_jobs_writeback"

type jobExportRow struct {
	Id                              string `db:"id" json:"id"`
	Number                          string `db:"number" json:"number"`
	Description                     string `db:"description" json:"description"`
	Parent                          string `db:"parent" json:"parent_id"`
	Proposal                        string `db:"proposal" json:"proposal_id"`
	ClientName                      string `db:"client" json:"client"`
	ClientID                        string `db:"client_id" json:"client_id"`
	ClientBusinessDevelopmentLead   string `db:"client_business_development_lead" json:"client_business_development_lead"`
	ContactSurname                  string `db:"contact_surname" json:"contact_surname"`
	ContactGivenName                string `db:"contact_given_name" json:"contact_given_name"`
	ContactEmail                    string `db:"contact_email" json:"contact_email"`
	ContactID                       string `db:"contact_id" json:"contact_id"`
	ManagerGivenName                string `db:"manager_given_name" json:"manager_given_name"`
	ManagerSurname                  string `db:"manager_surname" json:"manager_surname"`
	AlternateManagerGivenName       string `db:"alternate_manager_given_name" json:"alternate_manager_given_name"`
	AlternateManagerSurname         string `db:"alternate_manager_surname" json:"alternate_manager_surname"`
	JobOwnerName                    string `db:"job_owner" json:"job_owner"`
	JobOwnerID                      string `db:"job_owner_id" json:"job_owner_id"`
	JobOwnerBusinessDevelopmentLead string `db:"job_owner_business_development_lead" json:"job_owner_business_development_lead"`
	BranchCode                      string `db:"branch" json:"branch"`
}

func createJobsExportLegacyHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Try machine auth first (Bearer token)
		authorized := false
		authHeader := e.Request.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if record, err := app.FindRecordById("machine_secrets", JOBS_MACHINE_SECRET_ID); err == nil {
				salt := record.GetString("salt")
				storedHash := record.GetString("sha256_hash")
				h := sha256.New()
				h.Write([]byte(salt + token))
				if hex.EncodeToString(h.Sum(nil)) == storedHash {
					authorized = true
				}
			}
		}

		// Fall back to user auth with report claim
		if !authorized {
			if hasReport, _ := utilities.HasClaim(app, e.Auth, "report"); hasReport {
				authorized = true
			}
		}

		if !authorized {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		updatedAfter := e.Request.PathValue("updatedAfter")
		if updatedAfter == "" {
			return e.Error(http.StatusBadRequest, "updatedAfter is required", nil)
		}

		query := `
			SELECT j.id, 
			  j.number, 
			  j.description,
			  COALESCE(j.parent, '') AS parent,
			  COALESCE(j.proposal, '') AS proposal,
			  c.name AS client,
			  c.id AS client_id,
			  COALESCE(c.business_development_lead, '') AS client_business_development_lead,
			  COALESCE(co.surname, '') AS contact_surname,
			  COALESCE(co.given_name, '') AS contact_given_name,
			  COALESCE(co.email, '') AS contact_email,
			  COALESCE(co.id, '') AS contact_id,
			  COALESCE(p.given_name, '') AS manager_given_name,
			  COALESCE(p.surname, '') AS manager_surname,
			  COALESCE(pa.given_name, '') AS alternate_manager_given_name,
			  COALESCE(pa.surname, '') AS alternate_manager_surname,
			  COALESCE(jo.name, '') AS job_owner,
			  COALESCE(jo.id, '') AS job_owner_id,
			  COALESCE(jo.business_development_lead, '') AS job_owner_business_development_lead,
			  b.code AS branch
			FROM jobs j
			LEFT JOIN clients c ON j.client = c.id
			LEFT JOIN client_contacts co ON j.contact = co.id
			LEFT JOIN profiles p ON j.manager = p.uid
			LEFT JOIN profiles pa ON j.alternate_manager = pa.uid
			LEFT JOIN clients jo ON j.job_owner = jo.id
			LEFT JOIN branches b ON j.branch = b.id
			WHERE j.updated >= {:updatedAfter} AND j._imported = 0
		`

		var rows []jobExportRow
		if err := app.DB().NewQuery(query).Bind(dbx.Params{
			"updatedAfter": updatedAfter,
		}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query jobs: "+err.Error(), nil)
		}

		return e.JSON(http.StatusOK, rows)
	}
}
