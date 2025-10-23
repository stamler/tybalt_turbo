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
		isAuthorized, err := utilities.HasClaim(app, auth, "report")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "error checking claims", err)
		}
		if !isAuthorized {
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
	ID            string `db:"id" json:"id"`
	Uid           string `db:"uid" json:"uid"`
	GivenName     string `db:"given_name" json:"given_name"`
	Surname       string `db:"surname" json:"surname"`
	Submitted     bool   `db:"submitted" json:"submitted"`
	Approved      string `db:"approved" json:"approved"`
	Rejected      string `db:"rejected" json:"rejected"`
	Committed     string `db:"committed" json:"committed"`
	Approver      string `db:"approver" json:"approver"`
	Committer     string `db:"committer" json:"committer"`
	ApproverName  string `db:"approver_name" json:"approver_name"`
	CommitterName string `db:"committer_name" json:"committer_name"`
	RejectorName  string `db:"rejector_name" json:"rejector_name"`
	// Consolidated phase for grouping in UI
	Phase string `db:"phase" json:"phase"`
	// Aggregated totals for convenience in UI
	TotalHoursWorked     float64 `db:"total_hours_worked" json:"total_hours_worked"`
	TotalStat            float64 `db:"total_stat" json:"total_stat"`
	TotalPPTO            float64 `db:"total_ppto" json:"total_ppto"`
	TotalVacation        float64 `db:"total_vacation" json:"total_vacation"`
	TotalSick            float64 `db:"total_sick" json:"total_sick"`
	TotalToBank          float64 `db:"total_to_bank" json:"total_to_bank"`
	TotalBereavement     float64 `db:"total_bereavement" json:"total_bereavement"`
	TotalOTPayoutAmount  float64 `db:"total_ot_payout_request" json:"total_ot_payout_request"`
	TotalDaysOffRotation int     `db:"total_days_off_rotation" json:"total_days_off_rotation"`
}

// createTimesheetTrackingListHandler returns org-wide timesheets for a given week ending
func createTimesheetTrackingListHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth
		isAuthorized, err := utilities.HasClaim(app, auth, "report")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "error checking claims", err)
		}
		if !isAuthorized {
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
                COALESCE(ts.committer, '') AS committer,
                COALESCE(ap.given_name || ' ' || ap.surname, '') AS approver_name,
                COALESCE(cp.given_name || ' ' || cp.surname, '') AS committer_name,
                COALESCE(rp.given_name || ' ' || rp.surname, '') AS rejector_name,
                CASE
                    WHEN ts.committed != '' THEN 'Committed'
                    WHEN ts.approved != '' AND ts.committed = '' THEN 'Approved'
                    WHEN ts.submitted = 1 AND ts.approved = '' AND ts.committed = '' THEN 'Submitted'
                    ELSE 'Unsubmitted'
                END AS phase,
                COALESCE(agg.total_hours_worked, 0) AS total_hours_worked,
                COALESCE(agg.total_stat, 0) AS total_stat,
                COALESCE(agg.total_ppto, 0) AS total_ppto,
                COALESCE(agg.total_vacation, 0) AS total_vacation,
                COALESCE(agg.total_sick, 0) AS total_sick,
                COALESCE(agg.total_to_bank, 0) AS total_to_bank,
                COALESCE(agg.total_bereavement, 0) AS total_bereavement,
                COALESCE(agg.total_ot_payout_request, 0) AS total_ot_payout_request,
                COALESCE(agg.total_days_off_rotation, 0) AS total_days_off_rotation
            FROM time_sheets ts
            LEFT JOIN profiles p ON p.uid = ts.uid
            LEFT JOIN profiles ap ON ap.uid = ts.approver
            LEFT JOIN profiles cp ON cp.uid = ts.committer
            LEFT JOIN profiles rp ON rp.uid = ts.rejector
            LEFT JOIN (
                SELECT
                    te.tsid AS tsid,
                    SUM(CASE WHEN tt.code IN ('R','RT') THEN IFNULL(te.hours,0) ELSE 0 END) AS total_hours_worked,
                    SUM(CASE WHEN tt.code = 'OH' THEN IFNULL(te.hours,0) ELSE 0 END) AS total_stat,
                    SUM(CASE WHEN tt.code = 'OP' THEN IFNULL(te.hours,0) ELSE 0 END) AS total_ppto,
                    SUM(CASE WHEN tt.code = 'OV' THEN IFNULL(te.hours,0) ELSE 0 END) AS total_vacation,
                    SUM(CASE WHEN tt.code = 'OS' THEN IFNULL(te.hours,0) ELSE 0 END) AS total_sick,
                    SUM(CASE WHEN tt.code = 'RB' THEN IFNULL(te.hours,0) ELSE 0 END) AS total_to_bank,
                    SUM(CASE WHEN tt.code = 'OB' THEN IFNULL(te.hours,0) ELSE 0 END) AS total_bereavement,
                    SUM(CASE WHEN tt.code = 'OR' THEN 1 ELSE 0 END) AS total_days_off_rotation,
                    SUM(CASE WHEN tt.code = 'OTO' THEN IFNULL(te.payout_request_amount,0) ELSE 0 END) AS total_ot_payout_request
                FROM time_entries te
                LEFT JOIN time_types tt ON te.time_type = tt.id
								WHERE te.week_ending = {:week_ending}
                GROUP BY te.tsid
            ) agg ON agg.tsid = ts.id
            WHERE ts.week_ending = {:week_ending}
              AND ts.submitted = 1
            ORDER BY
                CASE
                    WHEN ts.committed != '' THEN 3
                    WHEN ts.approved != '' AND ts.committed = '' THEN 1
                    ELSE 2
                END,
                p.surname, p.given_name
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
