package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const JOBS_MACHINE_SECRET_ID = "legacy_jobs_writeback"

// formatDisplayName returns "Given Surname" or empty string if both are empty
func formatDisplayName(givenName, surname string) string {
	if givenName == "" && surname == "" {
		return ""
	}
	if givenName == "" {
		return surname
	}
	if surname == "" {
		return givenName
	}
	return givenName + " " + surname
}

// parseStringArray parses a JSON array string into a slice of strings
func parseStringArray(jsonStr string) []string {
	var result []string
	if jsonStr == "" || jsonStr == "[]" {
		return result
	}
	_ = json.Unmarshal([]byte(jsonStr), &result)
	return result
}

// parseFloatMap parses a JSON object string into a map of string to float64
func parseFloatMap(jsonStr string) map[string]float64 {
	result := make(map[string]float64)
	if jsonStr == "" || jsonStr == "{}" {
		return result
	}
	_ = json.Unmarshal([]byte(jsonStr), &result)
	return result
}

// Nested structs for extra Turbo data in job export
type jobExportClientRow struct {
	Id                      string `json:"id"`
	Name                    string `json:"name"`
	BusinessDevelopmentLead string `json:"business_development_lead,omitempty"`
}

type jobExportContactRow struct {
	Id        string `json:"id"`
	Surname   string `json:"surname"`
	GivenName string `json:"given_name"`
	Email     string `json:"email,omitempty"`
}

type jobExportManagerRow struct {
	Uid       string `json:"uid"`
	GivenName string `json:"given_name"`
	Surname   string `json:"surname"`
}

type jobExportJobOwnerRow struct {
	Id                      string `json:"id"`
	Name                    string `json:"name"`
	BusinessDevelopmentLead string `json:"business_development_lead,omitempty"`
}

// Internal struct for DB scanning
type jobExportDBRow struct {
	Id                              string  `db:"id"`
	Number                          string  `db:"number"`
	Description                     string  `db:"description"`
	Status                          string  `db:"status"`
	Parent                          string  `db:"parent"`
	ProposalNumber                  string  `db:"proposal_number"`
	ProposalId                      string  `db:"proposal_id"`
	FnAgreement                     bool    `db:"fn_agreement"`
	ProjectAwardDate                string  `db:"project_award_date"`
	ProposalOpeningDate             string  `db:"proposal_opening_date"`
	ProposalSubmissionDueDate       string  `db:"proposal_submission_due_date"`
	Location                        string  `db:"location"`
	OutstandingBalance              float64 `db:"outstanding_balance"`
	OutstandingBalanceDate          string  `db:"outstanding_balance_date"`
	AuthorizingDocument             string  `db:"authorizing_document"`
	ClientPo                        string  `db:"client_po"`
	ClientReferenceNumber           string  `db:"client_reference_number"`
	Created                         string  `db:"created"`
	Updated                         string  `db:"updated"`
	CategoriesJSON                  string  `db:"categories_json"`
	DivisionsJSON                   string  `db:"divisions_json"`
	TimeAllocationsJSON             string  `db:"time_allocations_json"`
	ClientName                      string  `db:"client_name"`
	ClientID                        string  `db:"client_id"`
	ClientBusinessDevelopmentLead   string  `db:"client_business_development_lead"`
	ContactSurname                  string  `db:"contact_surname"`
	ContactGivenName                string  `db:"contact_given_name"`
	ContactEmail                    string  `db:"contact_email"`
	ContactID                       string  `db:"contact_id"`
	ManagerUid                      string  `db:"manager_uid"`
	ManagerGivenName                string  `db:"manager_given_name"`
	ManagerSurname                  string  `db:"manager_surname"`
	AlternateManagerUid             string  `db:"alternate_manager_uid"`
	AlternateManagerGivenName       string  `db:"alternate_manager_given_name"`
	AlternateManagerSurname         string  `db:"alternate_manager_surname"`
	JobOwnerName                    string  `db:"job_owner_name"`
	JobOwnerID                      string  `db:"job_owner_id"`
	JobOwnerBusinessDevelopmentLead string  `db:"job_owner_business_development_lead"`
	BranchCode                      string  `db:"branch_code"`
}

// Output struct matching legacy Firestore format with _row objects for extra data
type jobExportOutput struct {
	// Legacy-compatible top-level fields
	ImmutableID                 string  `json:"immutableID"`
	Number                      string  `json:"number"`
	Description                 string  `json:"description"`
	Status                      string  `json:"status"`
	Client                      string  `json:"client"`
	ClientContact               string  `json:"clientContact"`
	Manager                     string  `json:"manager"`
	ManagerDisplayName          string  `json:"managerDisplayName"`
	ManagerUid                  string  `json:"managerUid"`
	Branch                      string  `json:"branch"`
	JobOwner                    string  `json:"jobOwner"`
	AlternateManagerUid         string  `json:"alternateManagerUid,omitempty"`
	AlternateManagerDisplayName string  `json:"alternateManagerDisplayName,omitempty"`
	Proposal                    string  `json:"proposal,omitempty"`
	FnAgreement                 bool    `json:"fnAgreement"`
	ProjectAwardDate            string  `json:"projectAwardDate,omitempty"`
	ProposalOpeningDate         string  `json:"proposalOpeningDate,omitempty"`
	ProposalSubmissionDueDate   string  `json:"proposalSubmissionDueDate,omitempty"`
	Location                    string  `json:"location,omitempty"`
	OutstandingBalance          float64 `json:"outstandingBalance"`
	OutstandingBalanceDate      string  `json:"outstandingBalanceDate,omitempty"`
	AuthorizingDocument         string  `json:"authorizingDocument,omitempty"`
	ClientPo                    string  `json:"clientPo,omitempty"`
	ClientReferenceNumber       string  `json:"clientReferenceNumber,omitempty"`
	Created                     string  `json:"created"`
	Updated                     string  `json:"updated"`

	// Legacy-compatible array fields
	Categories         []string           `json:"categories"`
	Divisions          []string           `json:"divisions"`
	JobTimeAllocations map[string]float64 `json:"jobTimeAllocations"`

	// Extra Turbo data scoped in _row objects (for related collections)
	ClientRow           *jobExportClientRow   `json:"client_row"`
	ContactRow          *jobExportContactRow  `json:"contact_row,omitempty"`
	ManagerRow          *jobExportManagerRow  `json:"manager_row,omitempty"`
	AlternateManagerRow *jobExportManagerRow  `json:"alternate_manager_row,omitempty"`
	JobOwnerRow         *jobExportJobOwnerRow `json:"job_owner_row,omitempty"`
	ParentId            string                `json:"parent_id,omitempty"`
	ProposalId          string                `json:"proposal_id,omitempty"`
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
			  j.status,
			  j.fn_agreement,
			  COALESCE(j.project_award_date, '') AS project_award_date,
			  COALESCE(j.proposal_opening_date, '') AS proposal_opening_date,
			  COALESCE(j.proposal_submission_due_date, '') AS proposal_submission_due_date,
			  COALESCE(j.location, '') AS location,
			  j.outstanding_balance,
			  COALESCE(j.outstanding_balance_date, '') AS outstanding_balance_date,
			  COALESCE(j.authorizing_document, '') AS authorizing_document,
			  COALESCE(j.client_po, '') AS client_po,
			  COALESCE(j.client_reference_number, '') AS client_reference_number,
			  j.created,
			  j.updated,
			  COALESCE((SELECT json_group_array(cat.name) FROM categories cat WHERE cat.job = j.id), '[]') AS categories_json,
			  COALESCE((SELECT json_group_array(d.code) FROM job_time_allocations jta JOIN divisions d ON jta.division = d.id WHERE jta.job = j.id), '[]') AS divisions_json,
			  COALESCE((SELECT json_group_object(d.code, jta.hours) FROM job_time_allocations jta JOIN divisions d ON jta.division = d.id WHERE jta.job = j.id), '{}') AS time_allocations_json,
			  COALESCE(j.parent, '') AS parent,
			  COALESCE(jp.number, '') AS proposal_number,
			  COALESCE(j.proposal, '') AS proposal_id,
			  c.name AS client_name,
			  c.id AS client_id,
			  COALESCE(c.business_development_lead, '') AS client_business_development_lead,
			  COALESCE(co.surname, '') AS contact_surname,
			  COALESCE(co.given_name, '') AS contact_given_name,
			  COALESCE(co.email, '') AS contact_email,
			  COALESCE(co.id, '') AS contact_id,
			  COALESCE(j.manager, '') AS manager_uid,
			  COALESCE(p.given_name, '') AS manager_given_name,
			  COALESCE(p.surname, '') AS manager_surname,
			  COALESCE(j.alternate_manager, '') AS alternate_manager_uid,
			  COALESCE(pa.given_name, '') AS alternate_manager_given_name,
			  COALESCE(pa.surname, '') AS alternate_manager_surname,
			  COALESCE(jo.name, '') AS job_owner_name,
			  COALESCE(jo.id, '') AS job_owner_id,
			  COALESCE(jo.business_development_lead, '') AS job_owner_business_development_lead,
			  b.code AS branch_code
			FROM jobs j
			LEFT JOIN clients c ON j.client = c.id
			LEFT JOIN client_contacts co ON j.contact = co.id
			LEFT JOIN profiles p ON j.manager = p.uid
			LEFT JOIN profiles pa ON j.alternate_manager = pa.uid
			LEFT JOIN clients jo ON j.job_owner = jo.id
			LEFT JOIN branches b ON j.branch = b.id
			LEFT JOIN jobs jp ON j.proposal = jp.id
			WHERE j.updated >= {:updatedAfter} AND j._imported = 0
		`

		var dbRows []jobExportDBRow
		if err := app.DB().NewQuery(query).Bind(dbx.Params{
			"updatedAfter": updatedAfter,
		}).All(&dbRows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query jobs: "+err.Error(), nil)
		}

		// Convert DB rows to output format
		output := make([]jobExportOutput, len(dbRows))
		for i, r := range dbRows {
			// Format display names
			managerDisplayName := formatDisplayName(r.ManagerGivenName, r.ManagerSurname)
			altManagerDisplayName := formatDisplayName(r.AlternateManagerGivenName, r.AlternateManagerSurname)
			clientContact := formatDisplayName(r.ContactGivenName, r.ContactSurname)

			output[i] = jobExportOutput{
				// Legacy-compatible fields
				ImmutableID:                 r.Id,
				Number:                      r.Number,
				Description:                 r.Description,
				Status:                      r.Status,
				Client:                      r.ClientName,
				ClientContact:               clientContact,
				Manager:                     managerDisplayName,
				ManagerDisplayName:          managerDisplayName,
				ManagerUid:                  r.ManagerUid,
				Branch:                      r.BranchCode,
				JobOwner:                    r.JobOwnerName,
				AlternateManagerUid:         r.AlternateManagerUid,
				AlternateManagerDisplayName: altManagerDisplayName,
				Proposal:                    r.ProposalNumber,
				FnAgreement:                 r.FnAgreement,
				ProjectAwardDate:            r.ProjectAwardDate,
				ProposalOpeningDate:         r.ProposalOpeningDate,
				ProposalSubmissionDueDate:   r.ProposalSubmissionDueDate,
				Location:                    r.Location,
				OutstandingBalance:          r.OutstandingBalance,
				OutstandingBalanceDate:      r.OutstandingBalanceDate,
				AuthorizingDocument:         r.AuthorizingDocument,
				ClientPo:                    r.ClientPo,
				ClientReferenceNumber:       r.ClientReferenceNumber,
				Created:                     r.Created,
				Updated:                     r.Updated,

				// Categories and divisions (parsed from JSON aggregations)
				Categories:         parseStringArray(r.CategoriesJSON),
				Divisions:          parseStringArray(r.DivisionsJSON),
				JobTimeAllocations: parseFloatMap(r.TimeAllocationsJSON),

				// Extra Turbo data in _row objects (for related collections)
				ClientRow: &jobExportClientRow{
					Id:                      r.ClientID,
					Name:                    r.ClientName,
					BusinessDevelopmentLead: r.ClientBusinessDevelopmentLead,
				},
				ParentId:   r.Parent,
				ProposalId: r.ProposalId,
			}

			// Only include optional _row objects if they have data
			if r.ContactID != "" {
				output[i].ContactRow = &jobExportContactRow{
					Id:        r.ContactID,
					Surname:   r.ContactSurname,
					GivenName: r.ContactGivenName,
					Email:     r.ContactEmail,
				}
			}
			if r.ManagerUid != "" {
				output[i].ManagerRow = &jobExportManagerRow{
					Uid:       r.ManagerUid,
					GivenName: r.ManagerGivenName,
					Surname:   r.ManagerSurname,
				}
			}
			if r.AlternateManagerUid != "" {
				output[i].AlternateManagerRow = &jobExportManagerRow{
					Uid:       r.AlternateManagerUid,
					GivenName: r.AlternateManagerGivenName,
					Surname:   r.AlternateManagerSurname,
				}
			}
			if r.JobOwnerID != "" {
				output[i].JobOwnerRow = &jobExportJobOwnerRow{
					Id:                      r.JobOwnerID,
					Name:                    r.JobOwnerName,
					BusinessDevelopmentLead: r.JobOwnerBusinessDevelopmentLead,
				}
			}
		}

		return e.JSON(http.StatusOK, output)
	}
}
