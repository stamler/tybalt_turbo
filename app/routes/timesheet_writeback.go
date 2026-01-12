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

const MACHINE_SECRET_ID = "machinekey00001"

type timeEntryExport struct {
	Id                  string  `db:"id" json:"id"`
	Uid                 string  `db:"uid" json:"uid"`
	Job                 string  `db:"job" json:"job,omitempty"`
	JobDescription      string  `db:"job_description" json:"jobDescription,omitempty"`
	Division            string  `db:"division" json:"division,omitempty"`
	DivisionName        string  `db:"division_name" json:"divisionName,omitempty"`
	TimeType            string  `db:"time_type" json:"timetype"`
	TimeTypeName        string  `db:"time_type_name" json:"timetypeName"`
	Date                string  `db:"date" json:"date"`
	Hours               float64 `db:"hours" json:"-"` // exported conditionally in MarshalJSON
	MealsHours          float64 `db:"meals_hours" json:"mealsHours,omitempty"`
	PayoutRequestAmount float64 `db:"payout_request_amount" json:"payoutRequestAmount,omitempty"`
	WorkRecord          string  `db:"work_record" json:"workrecord,omitempty"`
	Description         string  `db:"description" json:"workDescription,omitempty"`
	Category            string  `db:"category_name" json:"category,omitempty"`
	WeekEnding          string  `db:"week_ending" json:"weekEnding"`
	ClientName          string  `db:"client_name" json:"client,omitempty"`
	JobOwnerName        string  `db:"job_owner_name" json:"-"`
	ClientContact       string  `db:"client_contact" json:"-"`
	BranchCode          string  `db:"branch_code" json:"-"`
	ProposalNumber      string  `db:"proposal_number" json:"-"`
	JobStatus           string  `db:"job_status" json:"-"`
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

type workHoursTally struct {
	JobHours    float64 `json:"jobHours"`
	NoJobNumber float64 `json:"noJobNumber"`
	Hours       float64 `json:"hours"`
}

type jobTallyEntry struct {
	Branch        string `json:"branch,omitempty"`
	Client        string `json:"client"`
	JobOwner      string `json:"jobOwner,omitempty"`
	ClientContact string `json:"clientContact,omitempty"`
	Description   string `json:"description"`
	Status        string `json:"status"`
	Proposal      string `json:"proposal,omitempty"`
}

type timesheetExportRow struct {
	Id                   string                   `db:"id" json:"id"`
	WorkWeekHours        float64                  `db:"work_week_hours" json:"workWeekHours"`
	Salary               bool                     `db:"salary" json:"salary"`
	Uid                  string                   `db:"uid" json:"uid"`
	WeekEnding           string                   `db:"week_ending" json:"weekEnding"`
	GivenName            string                   `db:"given_name" json:"givenName"`
	Surname              string                   `db:"surname" json:"surname"`
	Manager              string                   `db:"approver" json:"managerUid"`
	ManagerName          string                   `db:"manager_name" json:"managerName"`
	DisplayName          string                   `db:"display_name" json:"displayName"`
	PayrollId            string                   `db:"payroll_id" json:"payrollId"`
	Locked               bool                     `json:"locked"`
	Approved             bool                     `json:"approved"`
	Rejected             bool                     `json:"rejected"`
	Submitted            bool                     `json:"submitted"`
	RejectionReason      string                   `json:"rejectionReason"`
	Entries              []timeEntryExport        `json:"entries"`
	BankedHours          float64                  `json:"bankedHours"`
	MealsHoursTally      float64                  `json:"mealsHoursTally"`
	Divisions            []string                 `json:"divisions"`
	DivisionsTally       map[string]string        `json:"divisionsTally"`
	JobNumbers           []string                 `json:"jobNumbers"`
	JobsTally            map[string]jobTallyEntry `json:"jobsTally"`
	NonWorkHoursTally    map[string]float64       `json:"nonWorkHoursTally"`
	OffRotationDaysTally int                      `json:"offRotationDaysTally"`
	OffWeekTally         int                      `json:"offWeekTally"`
	PayoutRequest        float64                  `json:"payoutRequest"`
	Timetypes            []string                 `json:"timetypes"`
	WorkHoursTally       workHoursTally           `json:"workHoursTally"`
}

func keys[K comparable, V any](m map[K]V) []K {
	result := make([]K, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

func createTimesheetExportLegacyHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Try machine auth first (Bearer token)
		authorized := false
		authHeader := e.Request.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if record, err := app.FindRecordById("machine_secrets", MACHINE_SECRET_ID); err == nil {
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
							 COALESCE(jp.number, '') AS proposal_number,
							 COALESCE(j.description, '') AS job_description,
				       COALESCE(d.code, '') AS division,
							 COALESCE(d.name, '') AS division_name,
				       tt.code AS time_type,
							 tt.name AS time_type_name,
				       te.date, te.hours, te.meals_hours,
							 te.payout_request_amount,
				       te.work_record, te.description,
							 te.week_ending,
							 COALESCE(c.name, '') AS client_name,
							 COALESCE(cc.given_name || ' ' || cc.surname, '') AS client_contact,
							 COALESCE(jo.name, '') AS job_owner_name,
							 COALESCE(ca.name, '') AS category_name,
							 COALESCE(b.code, '') AS branch_code,
							 COALESCE(j.status, '') AS job_status
				FROM time_entries te
				LEFT JOIN admin_profiles ap ON te.uid = ap.uid
				LEFT JOIN time_types tt ON te.time_type = tt.id
				LEFT JOIN divisions d ON te.division = d.id
				LEFT JOIN jobs j ON te.job = j.id
				LEFT JOIN jobs jp ON j.proposal = jp.id
				LEFT JOIN clients c ON j.client = c.id
				LEFT JOIN clients jo ON j.job_owner = jo.id
				LEFT JOIN client_contacts cc ON j.contact = cc.id
				LEFT JOIN categories ca ON te.category = ca.id
				LEFT JOIN branches b ON te.branch = b.id
				WHERE tsid = {:tsid}
			`
			if err := app.DB().NewQuery(entriesQuery).Bind(dbx.Params{
				"tsid": rows[i].Id,
			}).All(&entries); err != nil {
				return e.Error(http.StatusInternalServerError, "failed to query time entries: "+err.Error(), nil)
			}
			rows[i].Entries = entries

			// Initialize tally structures
			divSet := make(map[string]string)        // division code -> name
			jobSet := make(map[string]jobTallyEntry) // job numbers -> job details
			ttSet := make(map[string]struct{})       // time type codes
			orDates := make(map[string]struct{})     // unique OR dates
			owDates := make(map[string]struct{})     // unique OW dates
			nonWork := make(map[string]float64)      // time type -> hours
			var wht workHoursTally
			var mealsTotal, payoutTotal, bankedTotal float64

			for _, e := range entries {
				ttSet[e.TimeType] = struct{}{}

				switch e.TimeType {
				case "OR":
					orDates[e.Date] = struct{}{}
				case "OW":
					owDates[e.Date] = struct{}{}
				case "OTO":
					payoutTotal += e.PayoutRequestAmount
				case "RB":
					bankedTotal += e.Hours
				case "R", "RT":
					if e.Division != "" {
						divSet[e.Division] = e.DivisionName
					}
					if e.Job != "" {
						jobSet[e.Job] = jobTallyEntry{
							Branch:        e.BranchCode,
							Client:        e.ClientName,
							JobOwner:      e.JobOwnerName,
							ClientContact: e.ClientContact,
							Description:   e.JobDescription,
							Status:        e.JobStatus,
							Proposal:      e.ProposalNumber,
						}
						wht.JobHours += e.Hours
					} else {
						wht.NoJobNumber += e.Hours
						wht.Hours += e.Hours
					}
					mealsTotal += e.MealsHours
				default:
					nonWork[e.TimeType] += e.Hours
				}
			}

			// Convert sets to slices and assign
			rows[i].DivisionsTally = divSet
			rows[i].Divisions = keys(divSet)
			rows[i].JobNumbers = keys(jobSet)
			rows[i].JobsTally = jobSet
			rows[i].Timetypes = keys(ttSet)
			rows[i].OffRotationDaysTally = len(orDates)
			rows[i].OffWeekTally = len(owDates)
			rows[i].NonWorkHoursTally = nonWork
			rows[i].BankedHours = bankedTotal
			rows[i].MealsHoursTally = mealsTotal
			rows[i].PayoutRequest = payoutTotal
			rows[i].WorkHoursTally = wht
		}

		return e.JSON(http.StatusOK, rows)
	}
}
