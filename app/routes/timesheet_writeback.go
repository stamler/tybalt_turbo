package routes

import (
	"encoding/json"
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type timeEntryExport struct {
	Id             string  `db:"id" json:"id"`
	Uid            string  `db:"uid" json:"uid"`
	Job            string  `db:"job" json:"job,omitempty"`
	JobDescription string  `db:"job_description" json:"jobDescription,omitempty"`
	Division       string  `db:"division" json:"division,omitempty"`
	DivisionName   string  `db:"division_name" json:"divisionName,omitempty"`
	TimeType       string  `db:"time_type" json:"timeType"`
	TimeTypeName   string  `db:"time_type_name" json:"timetypeName"`
	Date           string  `db:"date" json:"date"`
	Hours          float64 `db:"hours" json:"-"` // exported conditionally in MarshalJSON
	MealsHours     float64 `db:"meals_hours" json:"mealsHours,omitempty"`
	WorkRecord     string  `db:"work_record" json:"workRecord,omitempty"`
	Description    string  `db:"description" json:"workDescription,omitempty"`
	Category       string  `db:"category_name" json:"category,omitempty"`
	WeekEnding     string  `db:"week_ending" json:"weekEnding"`
	ClientName     string  `db:"client_name" json:"client,omitempty"`
}

// MarshalJSON implements json.Marshaler to conditionally rename the hours field.
// When Job is not empty, hours is exported as "jobHours"; otherwise as "hours".
// The Alias type prevents infinite recursion: calling json.Marshal(t) directly
// would invoke this method again, but Alias doesn't inherit MarshalJSON.
func (t timeEntryExport) MarshalJSON() ([]byte, error) {
	type Alias timeEntryExport

	if t.Job != "" {
		return json.Marshal(struct {
			Alias
			JobHours float64 `json:"jobHours"`
		}{
			Alias:    Alias(t),
			JobHours: t.Hours,
		})
	}

	return json.Marshal(struct {
		Alias
		Hours float64 `json:"hours"`
	}{
		Alias: Alias(t),
		Hours: t.Hours,
	})
}

type timesheetExportRow struct {
	Id              string            `db:"id" json:"id"`
	WorkWeekHours   float64           `db:"work_week_hours" json:"workWeekHours"`
	Salary          bool              `db:"salary" json:"salary"`
	Uid             string            `db:"uid" json:"uid"`
	WeekEnding      string            `db:"week_ending" json:"weekEnding"`
	GivenName       string            `db:"given_name" json:"givenName"`
	Surname         string            `db:"surname" json:"surname"`
	Manager         string            `db:"approver" json:"managerUid"`
	ManagerName     string            `db:"manager_name" json:"managerName"`
	DisplayName     string            `db:"display_name" json:"displayName"`
	PayrollId       string            `db:"payroll_id" json:"payrollId"`
	Locked          bool              `json:"locked"`
	Approved        bool              `json:"approved"`
	Rejected        bool              `json:"rejected"`
	Submitted       bool              `json:"submitted"`
	RejectionReason string            `json:"rejectionReason"`
	Entries         []timeEntryExport `json:"entries"`
}

func createTimesheetExportLegacyHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		weekEnding := e.Request.PathValue("weekEnding")
		if weekEnding == "" {
			return e.Error(http.StatusBadRequest, "weekEnding is required", nil)
		}

		query := `
			SELECT ts.id, ap.legacy_uid AS uid, ts.week_ending, ts.work_week_hours, ts.salary, apm.legacy_uid AS approver,
				m.given_name || ' ' || m.surname AS manager_name,
				p.given_name, p.surname,
				p.given_name || ' ' || p.surname AS display_name,
				ap.payroll_id
			FROM time_sheets ts 
			LEFT JOIN profiles p ON ts.uid = p.uid
			LEFT JOIN admin_profiles ap ON ts.uid = ap.uid
			LEFT JOIN admin_profiles apm ON ts.approver = apm.uid
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

		// Set constants for all rows and fetch time entries
		for i := range rows {
			rows[i].Locked = true
			rows[i].Approved = true
			rows[i].Rejected = false
			rows[i].Submitted = true
			rows[i].RejectionReason = ""

			// Fetch time entries for this timesheet
			var entries []timeEntryExport
			entriesQuery := `
				SELECT te.id, ap.legacy_uid AS uid, 
				       COALESCE(j.number, '') AS job,
							 COALESCE(j.description, '') AS job_description,
				       COALESCE(d.code, '') AS division,
							 COALESCE(d.name, '') AS division_name,
				       tt.code AS time_type,
							 tt.name AS time_type_name,
				       te.date, te.hours, te.meals_hours, 
				       te.work_record, te.description,
							 te.week_ending,
							 COALESCE(c.name, '') AS client_name,
							 COALESCE(ca.name, '') AS category_name
				FROM time_entries te
				LEFT JOIN admin_profiles ap ON te.uid = ap.uid
				LEFT JOIN time_types tt ON te.time_type = tt.id
				LEFT JOIN divisions d ON te.division = d.id
				LEFT JOIN jobs j ON te.job = j.id
				LEFT JOIN clients c ON j.client = c.id
				LEFT JOIN categories ca ON te.category = ca.id
				WHERE tsid = {:tsid}
			`
			if err := app.DB().NewQuery(entriesQuery).Bind(dbx.Params{
				"tsid": rows[i].Id,
			}).All(&entries); err != nil {
				return e.Error(http.StatusInternalServerError, "failed to query time entries: "+err.Error(), nil)
			}
			rows[i].Entries = entries
		}

		return e.JSON(http.StatusOK, rows)
	}
}
