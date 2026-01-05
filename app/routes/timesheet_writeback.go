package routes

import (
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type timesheetExportRow struct {
	Id              string  `db:"id" json:"id"`
	WorkWeekHours   float64 `db:"work_week_hours" json:"workWeekHours"`
	Salary          bool    `db:"salary" json:"salary"`
	Uid             string  `db:"uid" json:"uid"`
	WeekEnding      string  `db:"week_ending" json:"weekEnding"`
	GivenName       string  `db:"given_name" json:"givenName"`
	Surname         string  `db:"surname" json:"surname"`
	Manager         string  `db:"approver" json:"managerUid"`
	ManagerName     string  `db:"manager_name" json:"managerName"`
	DisplayName     string  `db:"display_name" json:"displayName"`
	PayrollId       string  `db:"payroll_id" json:"payrollId"`
	Locked          bool    `json:"locked"`
	Approved        bool    `json:"approved"`
	Rejected        bool    `json:"rejected"`
	Submitted       bool    `json:"submitted"`
	RejectionReason string  `json:"rejectionReason"`
}

func createTimesheetExportLegacyHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		weekEnding := e.Request.PathValue("weekEnding")
		if weekEnding == "" {
			return e.Error(http.StatusBadRequest, "weekEnding is required", nil)
		}

		query := `
			SELECT ts.id, ts.uid, ts.week_ending, ts.work_week_hours, ts.salary, ts.approver,
				m.given_name || ' ' || m.surname AS manager_name,
				p.given_name, p.surname,
				p.given_name || ' ' || p.surname AS display_name,
				ap.payroll_id
			FROM time_sheets ts 
			LEFT JOIN profiles p ON ts.uid = p.uid
			LEFT JOIN admin_profiles ap ON ts.uid = ap.uid
			LEFT JOIN profiles m ON ts.approver = m.uid
			WHERE ts.week_ending = {:weekEnding}
			  AND ts.committed != ''
		`

		var rows []timesheetExportRow
		if err := app.DB().NewQuery(query).Bind(dbx.Params{
			"weekEnding": weekEnding,
		}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query time sheets: "+err.Error(), nil)
		}

		// Set constants for all rows
		for i := range rows {
			rows[i].Locked = true
			rows[i].Approved = true
			rows[i].Rejected = false
			rows[i].Submitted = true
			rows[i].RejectionReason = ""
		}

		// TODO: Insert entries and their corresponding tallies for each row

		return e.JSON(http.StatusOK, rows)
	}
}
