package routes

import (
	"encoding/json"
	"net/http"
	"strings"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// parseStringArray parses a JSON array string into a slice of strings
func parseStringArray(jsonStr string) []string {
	result := make([]string, 0)
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

// Wrapper response struct for structured jobs writeback
type jobsWritebackResponse struct {
	Jobs             []jobExportOutput            `json:"jobs"`
	Clients          []clientExportOutput         `json:"clients"`
	ClientContacts   []contactExportOutput        `json:"clientContacts"`
	RateRoles        []rateRoleExportOutput       `json:"rateRoles"`
	RateSheets       []rateSheetExportOutput      `json:"rateSheets"`
	RateSheetEntries []rateSheetEntryExportOutput `json:"rateSheetEntries"`
}

// Rate role export struct
type rateRoleExportOutput struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

// Rate sheet export struct
type rateSheetExportOutput struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	EffectiveDate string `json:"effectiveDate"`
	Revision      int    `json:"revision"`
	Active        bool   `json:"active"`
}

// Rate sheet entry export struct
type rateSheetEntryExportOutput struct {
	Id           string  `json:"id"`
	RoleId       string  `json:"roleId"`
	RateSheetId  string  `json:"rateSheetId"`
	Rate         float64 `json:"rate"`
	OvertimeRate float64 `json:"overtimeRate"`
}

// Client note export struct for notes array within each client
type clientNoteExportOutput struct {
	Id                 string `json:"id"`
	Created            string `json:"created"`
	Updated            string `json:"updated"`
	Note               string `json:"note"`
	Uid                string `json:"uid"`                 // legacy_uid of the author
	JobId              string `json:"jobId,omitempty"`     // PocketBase job ID
	JobNumber          string `json:"jobNumber,omitempty"` // job number for reference
	JobNotApplicable   bool   `json:"jobNotApplicable"`
	JobStatusChangedTo string `json:"jobStatusChangedTo,omitempty"`
}

// Client export struct for separate clients array
type clientExportOutput struct {
	Id                      string                   `json:"id"`
	Name                    string                   `json:"name"`
	BusinessDevelopmentLead string                   `json:"businessDevelopmentLead,omitempty"` // legacy_uid of the user
	Notes                   []clientNoteExportOutput `json:"notes,omitempty"`
}

// Client contact export struct for separate clientContacts array
type contactExportOutput struct {
	Id        string `json:"id"`
	Surname   string `json:"surname"`
	GivenName string `json:"givenName"`
	Email     string `json:"email,omitempty"`
	ClientId  string `json:"clientId"`
}

// Internal struct for DB scanning - simplified to only include ID references for relationships
type jobExportDBRow struct {
	Id                        string  `db:"id"`
	Number                    string  `db:"number"`
	Description               string  `db:"description"`
	Status                    string  `db:"status"`
	Parent                    string  `db:"parent"`
	ProposalNumber            string  `db:"proposal_number"`
	ProposalId                string  `db:"proposal_id"`
	FnAgreement               bool    `db:"fn_agreement"`
	ProjectAwardDate          string  `db:"project_award_date"`
	ProposalOpeningDate       string  `db:"proposal_opening_date"`
	ProposalSubmissionDueDate string  `db:"proposal_submission_due_date"`
	ProposalValue             float64 `db:"proposal_value"`
	TimeAndMaterials          bool    `db:"time_and_materials"`
	Location                  string  `db:"location"`
	OutstandingBalance        float64 `db:"outstanding_balance"`
	OutstandingBalanceDate    string  `db:"outstanding_balance_date"`
	AuthorizingDocument       string  `db:"authorizing_document"`
	ClientPo                  string  `db:"client_po"`
	ClientReferenceNumber     string  `db:"client_reference_number"`
	Created                   string  `db:"created"`
	Updated                   string  `db:"updated"`
	CategoriesJSON            string  `db:"categories_json"`
	DivisionsJSON             string  `db:"divisions_json"`
	TimeAllocationsJSON       string  `db:"time_allocations_json"`
	// Simple ID references (full data is in separate clients/contacts arrays)
	ClientID           string `db:"client_id"`
	ClientName         string `db:"client_name"`
	ContactID          string `db:"contact_id"`
	ContactDisplayName string `db:"contact_display_name"`
	ManagerUid         string `db:"manager_uid"`
	ManagerDisplayName string `db:"manager_display_name"`
	AltManagerUid      string `db:"alt_manager_uid"`
	AltManagerDisplay  string `db:"alt_manager_display"`
	JobOwnerID         string `db:"job_owner_id"`
	JobOwnerName       string `db:"job_owner_name"`
	BranchCode         string `db:"branch_code"`
	RateSheetID        string `db:"rate_sheet"`
}

// Internal struct for DB scanning - clients query
type clientExportDBRow struct {
	Id                      string `db:"id"`
	Name                    string `db:"name"`
	BusinessDevelopmentLead string `db:"business_development_lead"` // legacy_uid from admin_profiles
}

// Internal struct for DB scanning - contacts query
type contactExportDBRow struct {
	Id        string `db:"id"`
	Surname   string `db:"surname"`
	GivenName string `db:"given_name"`
	Email     string `db:"email"`
	ClientId  string `db:"client_id"`
}

// Internal struct for DB scanning - client notes query
type clientNoteExportDBRow struct {
	Id                 string `db:"id"`
	Created            string `db:"created"`
	Updated            string `db:"updated"`
	Note               string `db:"note"`
	ClientId           string `db:"client_id"`
	JobId              string `db:"job_id"`
	JobNumber          string `db:"job_number"`
	Uid                string `db:"uid"` // legacy_uid from admin_profiles
	JobNotApplicable   bool   `db:"job_not_applicable"`
	JobStatusChangedTo string `db:"job_status_changed_to"`
}

// Internal struct for DB scanning - rate roles query
type rateRoleExportDBRow struct {
	Id   string `db:"id"`
	Name string `db:"name"`
}

// Internal struct for DB scanning - rate sheets query
type rateSheetExportDBRow struct {
	Id            string `db:"id"`
	Name          string `db:"name"`
	EffectiveDate string `db:"effective_date"`
	Revision      int    `db:"revision"`
	Active        bool   `db:"active"`
}

// Internal struct for DB scanning - rate sheet entries query
type rateSheetEntryExportDBRow struct {
	Id           string  `db:"id"`
	RoleId       string  `db:"role"`
	RateSheetId  string  `db:"rate_sheet"`
	Rate         float64 `db:"rate"`
	OvertimeRate float64 `db:"overtime_rate"`
}

// Output struct matching legacy Firestore format with ID references instead of _row objects
type jobExportOutput struct {
	// Legacy-compatible top-level fields
	ImmutableID                 string  `json:"immutableID"`
	Number                      string  `json:"number"`
	Description                 string  `json:"description"`
	Status                      string  `json:"status"`
	Client                      string  `json:"client"`
	ClientContact               string  `json:"clientContact"`
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
	ProposalValue               float64 `json:"proposalValue"`
	TimeAndMaterials            bool    `json:"timeAndMaterials"`
	Location                    string  `json:"location,omitempty"`
	OutstandingBalance          float64 `json:"outstandingBalance"`
	OutstandingBalanceDate      string  `json:"outstandingBalanceDate,omitempty"`
	AuthorizingDocument         string  `json:"authorizingDocument,omitempty"`
	ClientPo                    string  `json:"clientPo,omitempty"`
	ClientReferenceNumber       string  `json:"clientReferenceNumber,omitempty"`
	Created                     string  `json:"created"`
	Updated                     string  `json:"updated"`

	// Legacy-compatible array fields
	Categories         []string           `json:"categories,omitempty"`
	Divisions          []string           `json:"divisions"`
	JobTimeAllocations map[string]float64 `json:"jobTimeAllocations"`

	// ID references to separate clients/contacts arrays (replaces _row objects)
	ClientId        string `json:"clientId"`
	JobOwnerId      string `json:"jobOwnerId,omitempty"`
	ClientContactId string `json:"clientContactId,omitempty"`
	ParentId        string `json:"parentId,omitempty"`
	ProposalId      string `json:"proposalId,omitempty"`
	RateSheetId     string `json:"rateSheetId,omitempty"`
}

func createJobsExportLegacyHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Try machine auth first (Bearer token matching any unexpired legacy_writeback secret)
		authorized := false
		authHeader := e.Request.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			// TrimSpace handles trailing newlines from secret managers
			token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			if utilities.ValidateMachineToken(app, token, "legacy_writeback") {
				authorized = true
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

		// Query 1: Jobs with simplified fields (ID references instead of denormalized data)
		jobsQuery := `
			SELECT j.id, 
			  j.number, 
			  j.description,
			  j.status,
			  j.fn_agreement,
			  COALESCE(j.project_award_date, '') AS project_award_date,
			  COALESCE(j.proposal_opening_date, '') AS proposal_opening_date,
			  COALESCE(j.proposal_submission_due_date, '') AS proposal_submission_due_date,
			  COALESCE(j.proposal_value, 0) AS proposal_value,
			  COALESCE(j.time_and_materials, 0) AS time_and_materials,
			  COALESCE(j.location, '') AS location,
			  j.outstanding_balance,
			  COALESCE(j.outstanding_balance_date, '') AS outstanding_balance_date,
			  COALESCE(j.authorizing_document, '') AS authorizing_document,
			  COALESCE(j.client_po, '') AS client_po,
			  COALESCE(j.client_reference_number, '') AS client_reference_number,
			  j.created,
			  j.updated,
			  COALESCE((SELECT json_group_array(cat.name) FROM categories cat WHERE cat.job = j.id ORDER BY cat.name), '[]') AS categories_json,
			  COALESCE((SELECT json_group_array(d.code) FROM job_time_allocations jta JOIN divisions d ON jta.division = d.id WHERE jta.job = j.id ORDER BY d.code), '[]') AS divisions_json,
			  COALESCE((SELECT json_group_object(d.code, jta.hours) FROM job_time_allocations jta JOIN divisions d ON jta.division = d.id WHERE jta.job = j.id), '{}') AS time_allocations_json,
			  COALESCE(j.parent, '') AS parent,
			  COALESCE(jp.number, '') AS proposal_number,
			  COALESCE(j.proposal, '') AS proposal_id,
			  COALESCE(j.client, '') AS client_id,
			  COALESCE(c.name, '') AS client_name,
			  COALESCE(j.contact, '') AS contact_id,
			  COALESCE(co.given_name || ' ' || co.surname, '') AS contact_display_name,
			  COALESCE(apm.legacy_uid, '') AS manager_uid,
			  COALESCE(p.given_name || ' ' || p.surname, '') AS manager_display_name,
			  COALESCE(apam.legacy_uid, '') AS alt_manager_uid,
			  COALESCE(pa.given_name || ' ' || pa.surname, '') AS alt_manager_display,
			  COALESCE(j.job_owner, '') AS job_owner_id,
			  COALESCE(jo.name, '') AS job_owner_name,
			  b.code AS branch_code,
			  j.rate_sheet
			FROM jobs j
			LEFT JOIN clients c ON j.client = c.id
			LEFT JOIN client_contacts co ON j.contact = co.id
			LEFT JOIN profiles p ON j.manager = p.uid
			LEFT JOIN admin_profiles apm ON j.manager = apm.uid
			LEFT JOIN profiles pa ON j.alternate_manager = pa.uid
			LEFT JOIN admin_profiles apam ON j.alternate_manager = apam.uid
			LEFT JOIN clients jo ON j.job_owner = jo.id
			LEFT JOIN branches b ON j.branch = b.id
			LEFT JOIN jobs jp ON j.proposal = jp.id
			WHERE j.updated >= {:updatedAfter} AND j._imported = 0
		`

		var jobRows []jobExportDBRow
		if err := app.DB().NewQuery(jobsQuery).Bind(dbx.Params{
			"updatedAfter": updatedAfter,
		}).All(&jobRows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query jobs: "+err.Error(), nil)
		}

		// Query 2: Unique clients (union of client and job_owner from matched jobs)
		// Convert business_development_lead to legacy_uid via admin_profiles join
		clientsQuery := `
			SELECT DISTINCT cl.id, cl.name, COALESCE(ap.legacy_uid, '') AS business_development_lead
			FROM (
				SELECT j.client AS client_id FROM jobs j 
				WHERE j.updated >= {:updatedAfter} AND j._imported = 0 AND j.client IS NOT NULL AND j.client != ''
				UNION
				SELECT j.job_owner AS client_id FROM jobs j 
				WHERE j.updated >= {:updatedAfter} AND j._imported = 0 AND j.job_owner IS NOT NULL AND j.job_owner != ''
			) AS refs
			JOIN clients cl ON refs.client_id = cl.id
			LEFT JOIN admin_profiles ap ON cl.business_development_lead = ap.uid
		`

		var clientRows []clientExportDBRow
		if err := app.DB().NewQuery(clientsQuery).Bind(dbx.Params{
			"updatedAfter": updatedAfter,
		}).All(&clientRows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query clients: "+err.Error(), nil)
		}

		// Query 3: Unique client contacts referenced by matched jobs
		contactsQuery := `
			SELECT DISTINCT co.id, co.surname, co.given_name, COALESCE(co.email, '') AS email, COALESCE(co.client, '') AS client_id
			FROM jobs j
			JOIN client_contacts co ON j.contact = co.id
			WHERE j.updated >= {:updatedAfter} AND j._imported = 0
		`

		var contactRows []contactExportDBRow
		if err := app.DB().NewQuery(contactsQuery).Bind(dbx.Params{
			"updatedAfter": updatedAfter,
		}).All(&contactRows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query contacts: "+err.Error(), nil)
		}

		// Query 4: All client_notes for matched clients (full fidelity - no _imported filtering)
		// Join with jobs to get job number, admin_profiles to get legacy_uid for author
		clientNotesQuery := `
			SELECT 
				cn.id,
				cn.created,
				cn.updated,
				cn.note,
				cn.client AS client_id,
				COALESCE(cn.job, '') AS job_id,
				COALESCE(j.number, '') AS job_number,
				COALESCE(ap.legacy_uid, '') AS uid,
				cn.job_not_applicable,
				COALESCE(cn.job_status_changed_to, '') AS job_status_changed_to
			FROM client_notes cn
			LEFT JOIN jobs j ON cn.job = j.id
			LEFT JOIN admin_profiles ap ON cn.uid = ap.uid
			WHERE cn.client IN (
				SELECT j.client FROM jobs j 
				WHERE j.updated >= {:updatedAfter} AND j._imported = 0 AND j.client IS NOT NULL AND j.client != ''
				UNION
				SELECT j.job_owner FROM jobs j 
				WHERE j.updated >= {:updatedAfter} AND j._imported = 0 AND j.job_owner IS NOT NULL AND j.job_owner != ''
			)
			ORDER BY cn.client, cn.created DESC
		`

		var noteRows []clientNoteExportDBRow
		if err := app.DB().NewQuery(clientNotesQuery).Bind(dbx.Params{
			"updatedAfter": updatedAfter,
		}).All(&noteRows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query client notes: "+err.Error(), nil)
		}

		// Query 5: All rate roles (small table, full sync for consistency)
		rateRolesQuery := `SELECT id, name FROM rate_roles ORDER BY name`

		var rateRoleRows []rateRoleExportDBRow
		if err := app.DB().NewQuery(rateRolesQuery).All(&rateRoleRows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query rate roles: "+err.Error(), nil)
		}

		// Query 6: Rate sheets referenced by matched jobs
		rateSheetsQuery := `
			SELECT DISTINCT rs.id, rs.name, rs.effective_date, rs.revision, rs.active
			FROM rate_sheets rs
			WHERE rs.id IN (
				SELECT j.rate_sheet FROM jobs j
				WHERE j.updated >= {:updatedAfter} AND j._imported = 0 AND j.rate_sheet IS NOT NULL AND j.rate_sheet != ''
			)
		`

		var rateSheetRows []rateSheetExportDBRow
		if err := app.DB().NewQuery(rateSheetsQuery).Bind(dbx.Params{
			"updatedAfter": updatedAfter,
		}).All(&rateSheetRows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query rate sheets: "+err.Error(), nil)
		}

		// Query 7: Rate sheet entries for matched rate sheets
		rateSheetEntriesQuery := `
			SELECT rse.id, rse.role, rse.rate_sheet, rse.rate, rse.overtime_rate
			FROM rate_sheet_entries rse
			WHERE rse.rate_sheet IN (
				SELECT j.rate_sheet FROM jobs j
				WHERE j.updated >= {:updatedAfter} AND j._imported = 0 AND j.rate_sheet IS NOT NULL AND j.rate_sheet != ''
			)
		`

		var rateSheetEntryRows []rateSheetEntryExportDBRow
		if err := app.DB().NewQuery(rateSheetEntriesQuery).Bind(dbx.Params{
			"updatedAfter": updatedAfter,
		}).All(&rateSheetEntryRows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query rate sheet entries: "+err.Error(), nil)
		}

		// Convert job DB rows to output format
		jobs := make([]jobExportOutput, len(jobRows))
		for i, r := range jobRows {
			jobs[i] = jobExportOutput{
				// Legacy-compatible fields
				ImmutableID:                 r.Id,
				Number:                      r.Number,
				Description:                 r.Description,
				Status:                      r.Status,
				Client:                      r.ClientName,
				ClientContact:               strings.TrimSpace(r.ContactDisplayName),
				ManagerDisplayName:          strings.TrimSpace(r.ManagerDisplayName),
				ManagerUid:                  r.ManagerUid,
				Branch:                      r.BranchCode,
				JobOwner:                    r.JobOwnerName,
				AlternateManagerUid:         r.AltManagerUid,
				AlternateManagerDisplayName: strings.TrimSpace(r.AltManagerDisplay),
				Proposal:                    r.ProposalNumber,
				FnAgreement:                 r.FnAgreement,
				ProjectAwardDate:            r.ProjectAwardDate,
				ProposalOpeningDate:         r.ProposalOpeningDate,
				ProposalSubmissionDueDate:   r.ProposalSubmissionDueDate,
				ProposalValue:               r.ProposalValue,
				TimeAndMaterials:            r.TimeAndMaterials,
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

				// ID references to separate arrays
				ClientId:        r.ClientID,
				JobOwnerId:      r.JobOwnerID,
				ClientContactId: r.ContactID,
				ParentId:        r.Parent,
				ProposalId:      r.ProposalId,
				RateSheetId:     r.RateSheetID,
			}
		}

		// Group notes by client_id for efficient lookup
		notesByClient := make(map[string][]clientNoteExportOutput)
		for _, n := range noteRows {
			notesByClient[n.ClientId] = append(notesByClient[n.ClientId], clientNoteExportOutput{
				Id:                 n.Id,
				Created:            n.Created,
				Updated:            n.Updated,
				Note:               n.Note,
				Uid:                n.Uid,
				JobId:              n.JobId,
				JobNumber:          n.JobNumber,
				JobNotApplicable:   n.JobNotApplicable,
				JobStatusChangedTo: n.JobStatusChangedTo,
			})
		}

		// Convert client DB rows to output format, attaching notes
		clients := make([]clientExportOutput, len(clientRows))
		for i, r := range clientRows {
			clients[i] = clientExportOutput{
				Id:                      r.Id,
				Name:                    r.Name,
				BusinessDevelopmentLead: r.BusinessDevelopmentLead,
				Notes:                   notesByClient[r.Id], // may be nil if no notes, omitted from JSON
			}
		}

		// Convert contact DB rows to output format
		contacts := make([]contactExportOutput, len(contactRows))
		for i, r := range contactRows {
			contacts[i] = contactExportOutput{
				Id:        r.Id,
				Surname:   r.Surname,
				GivenName: r.GivenName,
				Email:     r.Email,
				ClientId:  r.ClientId,
			}
		}

		// Convert rate role DB rows to output format
		rateRoles := make([]rateRoleExportOutput, len(rateRoleRows))
		for i, r := range rateRoleRows {
			rateRoles[i] = rateRoleExportOutput{
				Id:   r.Id,
				Name: r.Name,
			}
		}

		// Convert rate sheet DB rows to output format
		rateSheets := make([]rateSheetExportOutput, len(rateSheetRows))
		for i, r := range rateSheetRows {
			rateSheets[i] = rateSheetExportOutput{
				Id:            r.Id,
				Name:          r.Name,
				EffectiveDate: r.EffectiveDate,
				Revision:      r.Revision,
				Active:        r.Active,
			}
		}

		// Convert rate sheet entry DB rows to output format
		rateSheetEntries := make([]rateSheetEntryExportOutput, len(rateSheetEntryRows))
		for i, r := range rateSheetEntryRows {
			rateSheetEntries[i] = rateSheetEntryExportOutput{
				Id:           r.Id,
				RoleId:       r.RoleId,
				RateSheetId:  r.RateSheetId,
				Rate:         r.Rate,
				OvertimeRate: r.OvertimeRate,
			}
		}

		// Return structured response with all arrays
		return e.JSON(http.StatusOK, jobsWritebackResponse{
			Jobs:             jobs,
			Clients:          clients,
			ClientContacts:   contacts,
			RateRoles:        rateRoles,
			RateSheets:       rateSheets,
			RateSheetEntries: rateSheetEntries,
		})
	}
}
