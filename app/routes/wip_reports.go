package routes

import (
	_ "embed"
	"net/http"
	"time"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed wip_reports.sql
var wipReportScopeQuery string

type WIPReportRow struct {
	JobWIP
	ID          string `db:"job_id" json:"id"`
	Number      string `db:"number" json:"number"`
	Description string `db:"description" json:"description"`
	Client      string `db:"client" json:"client"`
	Manager     string `db:"manager" json:"manager"`
	BranchID    string `db:"branch_id" json:"branch_id"`
	Branch      string `db:"branch" json:"branch"`
}

type WIPReport struct {
	Items                []WIPReportRow `json:"items"`
	ExcludedProjectValue int            `json:"excluded_project_value"`
	AsOf                 string         `json:"as_of"`
}

func canViewManagementWIP(app core.App, auth *core.Record) (bool, error) {
	for _, claim := range []string{"kpi", "admin"} {
		hasClaim, err := utilities.HasClaim(app, auth, claim)
		if err != nil || hasClaim {
			return hasClaim, err
		}
	}
	branches, err := app.FindRecordsByFilter("branches", "manager = {:uid}", "", 1, 0, dbx.Params{"uid": auth.Id})
	return len(branches) > 0, err
}

func createGetWIPReportHandler(app core.App, mode string) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if mode != "my" {
			allowed, err := canViewManagementWIP(app, e.Auth)
			if err != nil {
				return e.InternalServerError("Failed to check WIP access", err)
			}
			if !allowed {
				return e.ForbiddenError("You need to be a branch manager or hold the kpi or admin claim to view this report", nil)
			}
		}
		q := e.Request.URL.Query()
		branch, division := q.Get("branch"), q.Get("division")
		if mode == "division" && division == "" {
			return e.BadRequestError("Select a division", nil)
		}
		for collection, id := range map[string]string{"branches": branch, "divisions": division} {
			if (mode == "branch" && collection == "branches" || mode == "division" && collection == "divisions") && id != "" {
				count, err := app.CountRecords(collection, dbx.HashExp{"id": id})
				if err != nil {
					return e.InternalServerError("Failed to check report selection", err)
				}
				if count == 0 {
					return e.BadRequestError("Unknown report selection", nil)
				}
			}
		}
		requireTime := q.Get("require_time")
		if mode == "division" && requireTime != "" && requireTime != "true" && requireTime != "false" {
			return e.BadRequestError("require_time must be true or false", nil)
		}
		result := WIPReport{Items: []WIPReportRow{}, AsOf: time.Now().UTC().Format("2006-01-02")}
		rows := []WIPReportRow{}
		// Aggregate all selected jobs in one query. Division membership changes
		// selection only; each row retains all of that job's WIP factors.
		err := app.DB().NewQuery(wipReportScopeQuery + jobPricedTimeEntriesQuery + jobWIPQuery + `
			SELECT w.*, j.number, j.description, COALESCE(c.name, '') AS client,
			TRIM(COALESCE(p.given_name, '') || ' ' || COALESCE(p.surname, '')) AS manager,
			COALESCE(j.branch, '') AS branch_id, COALESCE(b.name, '') AS branch
			FROM wip_values w JOIN selected_jobs j ON j.id = w.job_id
			LEFT JOIN clients c ON c.id = j.client
			LEFT JOIN profiles p ON p.uid = j.manager
			LEFT JOIN branches b ON b.id = j.branch
			ORDER BY j.number, j.id
		`).Bind(dbx.Params{
			"mode": mode, "uid": e.Auth.Id, "branch": branch, "division": division,
			"require_time": requireTime != "false", "start_date": "", "end_date": result.AsOf,
		}).All(&rows)
		if err != nil {
			return e.InternalServerError("Failed to load WIP report", err)
		}
		for _, row := range rows {
			if row.ProjectValue <= 0 {
				result.ExcludedProjectValue++
			} else {
				result.Items = append(result.Items, row)
			}
		}
		return e.JSON(http.StatusOK, result)
	}
}
