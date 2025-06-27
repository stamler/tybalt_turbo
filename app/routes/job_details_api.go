package routes

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed job_details.sql
var jobDetailsQuery string

// division struct returned in divisions_json
type Division struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type Person struct {
	ID        string `json:"id"`
	GivenName string `json:"given_name"`
	Surname   string `json:"surname"`
}

type jobDetailsRow struct {
	ID                        string         `db:"id"`
	Number                    string         `db:"number"`
	Description               string         `db:"description"`
	Status                    sql.NullString `db:"status"`
	ClientID                  string         `db:"client_id"`
	ClientName                string         `db:"client_name"`
	ContactID                 sql.NullString `db:"contact_id"`
	ContactGivenName          sql.NullString `db:"contact_given_name"`
	ContactSurname            sql.NullString `db:"contact_surname"`
	ManagerID                 sql.NullString `db:"manager_id"`
	ManagerGivenName          sql.NullString `db:"manager_given_name"`
	ManagerSurname            sql.NullString `db:"manager_surname"`
	AlternateManagerID        sql.NullString `db:"alternate_manager_id"`
	AlternateManagerGivenName sql.NullString `db:"alternate_manager_given_name"`
	AlternateManagerSurname   sql.NullString `db:"alternate_manager_surname"`
	JobOwnerID                sql.NullString `db:"job_owner_id"`
	JobOwnerGivenName         sql.NullString `db:"job_owner_given_name"`
	JobOwnerSurname           sql.NullString `db:"job_owner_surname"`
	ProposalID                sql.NullString `db:"proposal_id"`
	ProposalNumber            sql.NullString `db:"proposal_number"`
	FnAgreement               bool           `db:"fn_agreement"`
	ProjectAwardDate          sql.NullString `db:"project_award_date"`
	ProposalOpeningDate       sql.NullString `db:"proposal_opening_date"`
	ProposalSubmissionDueDate sql.NullString `db:"proposal_submission_due_date"`
	DivisionsJSON             string         `db:"divisions_json"`
	ProjectsJSON              string         `db:"projects_json"`
}

type ClientInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type JobRef struct {
	ID     string `json:"id"`
	Number string `json:"number"`
}

type JobDetails struct {
	ID                        string     `json:"id"`
	Number                    string     `json:"number"`
	Description               string     `json:"description"`
	Status                    string     `json:"status"`
	Client                    ClientInfo `json:"client"`
	Contact                   Person     `json:"contact"`
	Manager                   Person     `json:"manager"`
	AlternateManager          Person     `json:"alternate_manager"`
	JobOwner                  Person     `json:"job_owner"`
	ProposalID                string     `json:"proposal_id"`
	ProposalNumber            string     `json:"proposal_number"`
	FnAgreement               bool       `json:"fn_agreement"`
	ProjectAwardDate          string     `json:"project_award_date"`
	ProposalOpeningDate       string     `json:"proposal_opening_date"`
	ProposalSubmissionDueDate string     `json:"proposal_submission_due_date"`
	Divisions                 []Division `json:"divisions"`
	Projects                  []JobRef   `json:"projects"`
}

func createGetJobDetailsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")

		var rows []jobDetailsRow
		if err := app.DB().NewQuery(jobDetailsQuery).Bind(dbx.Params{"id": id}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}
		if len(rows) == 0 {
			return e.Error(http.StatusNotFound, "job not found", nil)
		}
		r := rows[0]

		// parse divisions json
		var divisions []Division
		_ = json.Unmarshal([]byte(r.DivisionsJSON), &divisions)
		var projects []JobRef
		_ = json.Unmarshal([]byte(r.ProjectsJSON), &projects)

		// helper to convert NullString to string
		ns := func(n sql.NullString) string {
			if n.Valid {
				return n.String
			}
			return ""
		}

		jd := JobDetails{
			ID:                        r.ID,
			Number:                    r.Number,
			Description:               r.Description,
			Status:                    ns(r.Status),
			Client:                    ClientInfo{ID: r.ClientID, Name: r.ClientName},
			Contact:                   Person{ID: ns(r.ContactID), GivenName: ns(r.ContactGivenName), Surname: ns(r.ContactSurname)},
			Manager:                   Person{ID: ns(r.ManagerID), GivenName: ns(r.ManagerGivenName), Surname: ns(r.ManagerSurname)},
			AlternateManager:          Person{ID: ns(r.AlternateManagerID), GivenName: ns(r.AlternateManagerGivenName), Surname: ns(r.AlternateManagerSurname)},
			JobOwner:                  Person{ID: ns(r.JobOwnerID), GivenName: ns(r.JobOwnerGivenName), Surname: ns(r.JobOwnerSurname)},
			ProposalID:                ns(r.ProposalID),
			ProposalNumber:            ns(r.ProposalNumber),
			FnAgreement:               r.FnAgreement,
			ProjectAwardDate:          ns(r.ProjectAwardDate),
			ProposalOpeningDate:       ns(r.ProposalOpeningDate),
			ProposalSubmissionDueDate: ns(r.ProposalSubmissionDueDate),
			Divisions:                 divisions,
			Projects:                  projects,
		}

		return e.JSON(http.StatusOK, jd)
	}
}
