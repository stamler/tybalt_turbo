package routes

import (
	"net/http"
	"time"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type trackingCountRow struct {
	WeekEnding     string `db:"week_ending" json:"week_ending"`
	SubmittedCount int    `db:"submitted_count" json:"submitted_count"`
	ApprovedCount  int    `db:"approved_count" json:"approved_count"`
	CommittedCount int    `db:"committed_count" json:"committed_count"`
	RejectedCount  int    `db:"rejected_count" json:"rejected_count"`
}

// createTimesheetTrackingCountsHandler returns weekly submitted/approved/committed counts for committers
func createTimesheetTrackingCountsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth
		hasCommit, err := utilities.HasClaim(app, auth, "commit")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "error checking claims", err)
		}
		if !hasCommit {
			return e.Error(http.StatusForbidden, "you are not authorized to view time tracking", nil)
		}

		query := `
            SELECT
                week_ending,
                -- committed are exclusive
                SUM(CASE WHEN committed != '' THEN 1 ELSE 0 END) AS committed_count,
                -- approved but not committed
                SUM(CASE WHEN approved != '' AND committed = '' THEN 1 ELSE 0 END) AS approved_count,
                -- submitted but neither approved nor committed
                SUM(CASE WHEN submitted = 1 AND approved = '' AND committed = '' THEN 1 ELSE 0 END) AS submitted_count,
                -- rejected is non-exclusive (can overlap others)
                SUM(CASE WHEN rejected != '' THEN 1 ELSE 0 END) AS rejected_count
            FROM time_sheets
            GROUP BY week_ending
            ORDER BY week_ending DESC
        `

		var rows []trackingCountRow
		if err := app.DB().NewQuery(query).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query", err)
		}

		return e.JSON(http.StatusOK, rows)
	}
}

type trackingListRow struct {
	ID        string `db:"id" json:"id"`
	Uid       string `db:"uid" json:"uid"`
	GivenName string `db:"given_name" json:"given_name"`
	Surname   string `db:"surname" json:"surname"`
	Submitted bool   `db:"submitted" json:"submitted"`
	Approved  string `db:"approved" json:"approved"`
	Rejected  string `db:"rejected" json:"rejected"`
	Committed string `db:"committed" json:"committed"`
	Approver  string `db:"approver" json:"approver"`
	Committer string `db:"committer" json:"committer"`
}

// createTimesheetTrackingListHandler returns org-wide timesheets for a given week ending
func createTimesheetTrackingListHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth
		hasCommit, err := utilities.HasClaim(app, auth, "commit")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "error checking claims", err)
		}
		if !hasCommit {
			return e.Error(http.StatusForbidden, "you are not authorized to view time tracking", nil)
		}

		weekEnding := e.Request.PathValue("weekEnding")
		if weekEnding == "" {
			return e.Error(http.StatusBadRequest, "weekEnding is required", nil)
		}
		if _, err := time.Parse("2006-01-02", weekEnding); err != nil {
			return e.Error(http.StatusBadRequest, "invalid date format (YYYY-MM-DD)", nil)
		}

		query := `
            SELECT
                ts.id,
                ts.uid,
                COALESCE(p.given_name, '') AS given_name,
                COALESCE(p.surname, '') AS surname,
                ts.submitted,
                ts.approved,
                ts.rejected,
                ts.committed,
                COALESCE(ts.approver, '') AS approver,
                COALESCE(ts.committer, '') AS committer
            FROM time_sheets ts
            LEFT JOIN profiles p ON p.uid = ts.uid
            WHERE ts.week_ending = {:week_ending}
            ORDER BY p.surname, p.given_name
        `

		var rows []trackingListRow
		if err := app.DB().NewQuery(query).Bind(dbx.Params{
			"week_ending": weekEnding,
		}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query", err)
		}

		return e.JSON(http.StatusOK, rows)
	}
}
