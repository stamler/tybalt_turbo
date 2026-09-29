package routes

import (
	"net/http"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type WIPMissingRateRole struct {
	ID    string  `db:"role_id" json:"id"`
	Name  string  `db:"role_name" json:"name"`
	Hours float64 `db:"hours" json:"hours"`
}

type WIPMissingRateSheet struct {
	ID       string `db:"rate_sheet_id" json:"id"`
	Name     string `db:"rate_sheet_name" json:"name"`
	Revision int    `db:"rate_sheet_revision" json:"revision"`
}

type WIPMissingRates struct {
	AsOf      string               `json:"as_of"`
	RateSheet *WIPMissingRateSheet `json:"rate_sheet"`
	Roles     []WIPMissingRateRole `json:"roles"`
}

func createGetJobWIPMissingRatesHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		today := time.Now().UTC().Format("2006-01-02")
		asOf := e.Request.URL.Query().Get("as_of")
		if asOf == "" {
			asOf = today
		}
		if _, err := time.Parse("2006-01-02", asOf); err != nil || asOf > today {
			return e.BadRequestError("as_of must be a valid date no later than today", nil)
		}
		var rows []struct {
			RateSheetID       string  `db:"rate_sheet_id"`
			RateSheetName     string  `db:"rate_sheet_name"`
			RateSheetRevision int     `db:"rate_sheet_revision"`
			RoleID            string  `db:"role_id"`
			RoleName          string  `db:"role_name"`
			Hours             float64 `db:"hours"`
		}
		// Use the same pricing rules and date limit as WIP. The left joins retain
		// jobs with no sheet or no missing rates, without a separate job lookup.
		err := app.DB().NewQuery(singleJobScopeQuery + jobPricedTimeEntriesQuery + `
			SELECT COALESCE(rs.id, '') AS rate_sheet_id,
				COALESCE(rs.name, '') AS rate_sheet_name,
				COALESCE(rs.revision, 0) AS rate_sheet_revision,
				COALESCE(p.role, '') AS role_id,
				COALESCE(NULLIF(r.name, ''), p.role, '') AS role_name,
				COALESCE(SUM(p.hours), 0) AS hours
			FROM selected_jobs j
			LEFT JOIN rate_sheets rs ON rs.id = j.rate_sheet
			LEFT JOIN priced_entries p ON p.job = j.id
				AND COALESCE(j.rate_sheet, '') != ''
				AND COALESCE(p.role, '') != '' AND p.sheet_rate IS NULL
			LEFT JOIN rate_roles r ON r.id = p.role
			GROUP BY j.id, rs.id, p.role
			ORDER BY role_name, role_id
		`).Bind(dbx.Params{
			"job_id": e.Request.PathValue("id"), "start_date": "", "end_date": asOf,
		}).All(&rows)
		if err != nil {
			return e.InternalServerError("Failed to load missing rates", err)
		}
		if len(rows) == 0 {
			return e.NotFoundError("Job not found", nil)
		}
		result := WIPMissingRates{AsOf: asOf, Roles: []WIPMissingRateRole{}}
		if rows[0].RateSheetID != "" {
			result.RateSheet = &WIPMissingRateSheet{ID: rows[0].RateSheetID, Name: rows[0].RateSheetName, Revision: rows[0].RateSheetRevision}
		}
		for _, row := range rows {
			if row.RoleID != "" && row.Hours != 0 {
				result.Roles = append(result.Roles, WIPMissingRateRole{ID: row.RoleID, Name: row.RoleName, Hours: row.Hours})
			}
		}
		return e.JSON(http.StatusOK, result)
	}
}
