package routes

import (
	"database/sql"
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type noteAuthor struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	GivenName string `json:"given_name"`
	Surname   string `json:"surname"`
}

type noteJob struct {
	ID          string `json:"id"`
	Number      string `json:"number"`
	Description string `json:"description"`
}

type clientNote struct {
	ID      string     `json:"id"`
	Created string     `json:"created"`
	Note    string     `json:"note"`
	Job     *noteJob   `json:"job"`
	Author  noteAuthor `json:"author"`
}

func createGetClientNotesHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		clientID := e.Request.PathValue("id")
		if clientID == "" {
			return e.Error(http.StatusBadRequest, "client id is required", nil)
		}

		query := `SELECT
			n.id,
			n.created,
			n.note,
			n.job,
			j.number AS job_number,
			j.description AS job_description,
			u.id AS user_id,
			u.email AS user_email,
			COALESCE(p.given_name, '') AS given_name,
			COALESCE(p.surname, '') AS surname
		FROM client_notes n
		LEFT JOIN jobs j ON j.id = n.job
		LEFT JOIN users u ON u.id = n.uid
		LEFT JOIN profiles p ON p.uid = n.uid
		WHERE n.client = {:client}
		ORDER BY n.created DESC`

		var rows []struct {
			ID             string         `db:"id"`
			Created        string         `db:"created"`
			Note           string         `db:"note"`
			JobID          sql.NullString `db:"job"`
			JobNumber      sql.NullString `db:"job_number"`
			JobDescription sql.NullString `db:"job_description"`
			UserID         string         `db:"user_id"`
			UserEmail      string         `db:"user_email"`
			GivenName      string         `db:"given_name"`
			Surname        string         `db:"surname"`
		}

		if err := app.DB().NewQuery(query).Bind(dbx.Params{"client": clientID}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load client notes", err)
		}

		notes := make([]clientNote, len(rows))
		for i, row := range rows {
			var job *noteJob
			if row.JobID.Valid {
				job = &noteJob{
					ID:          row.JobID.String,
					Number:      row.JobNumber.String,
					Description: row.JobDescription.String,
				}
			}
			notes[i] = clientNote{
				ID:      row.ID,
				Created: row.Created,
				Note:    row.Note,
				Job:     job,
				Author: noteAuthor{
					ID:        row.UserID,
					Email:     row.UserEmail,
					GivenName: row.GivenName,
					Surname:   row.Surname,
				},
			}
		}

		return e.JSON(http.StatusOK, notes)
	}
}

func createGetJobNotesHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		jobID := e.Request.PathValue("id")
		if jobID == "" {
			return e.Error(http.StatusBadRequest, "job id is required", nil)
		}

		query := `SELECT
			n.id,
			n.created,
			n.note,
			u.id AS user_id,
			u.email AS user_email,
			COALESCE(p.given_name, '') AS given_name,
			COALESCE(p.surname, '') AS surname
		FROM client_notes n
		LEFT JOIN users u ON u.id = n.uid
		LEFT JOIN profiles p ON p.uid = n.uid
		WHERE n.job = {:job}
		ORDER BY n.created DESC`

		var rows []struct {
			ID        string `db:"id"`
			Created   string `db:"created"`
			Note      string `db:"note"`
			UserID    string `db:"user_id"`
			UserEmail string `db:"user_email"`
			GivenName string `db:"given_name"`
			Surname   string `db:"surname"`
		}

		if err := app.DB().NewQuery(query).Bind(dbx.Params{"job": jobID}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load job notes", err)
		}

		notes := make([]clientNote, len(rows))
		for i, row := range rows {
			notes[i] = clientNote{
				ID:      row.ID,
				Created: row.Created,
				Note:    row.Note,
				Job:     nil,
				Author: noteAuthor{
					ID:        row.UserID,
					Email:     row.UserEmail,
					GivenName: row.GivenName,
					Surname:   row.Surname,
				},
			}
		}

		return e.JSON(http.StatusOK, notes)
	}
}
