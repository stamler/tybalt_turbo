package routes

import (
	_ "embed" // Needed for //go:embed
	"net/http"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed tallies.sql
var talliesQuery string

type TimeSheetTally struct {
	Id                  string                    `json:"id"`
	Approved            string                    `json:"approved"`
	BankEntryDates      utilities.JsonStringSlice `json:"bank_entry_dates"`
	DivisionNames       utilities.JsonStringSlice `json:"division_names"`
	Divisions           utilities.JsonStringSlice `json:"divisions"`
	JobNumbers          utilities.JsonStringSlice `json:"job_numbers"`
	MealsHours          float64                   `json:"meals_hours"`
	NonWorkTotalHours   float64                   `json:"non_work_total_hours"`
	ObHours             float64                   `json:"ob_hours"`
	OffRotationDates    utilities.JsonStringSlice `json:"off_rotation_dates"`
	OffWeekDates        utilities.JsonStringSlice `json:"off_week_dates"`
	OpHours             float64                   `json:"op_hours"`
	OsHours             float64                   `json:"os_hours"`
	OvHours             float64                   `json:"ov_hours"`
	PayoutRequestAmount float64                   `json:"payout_request_amount"`
	PayoutRequestDates  utilities.JsonStringSlice `json:"payout_request_dates"`
	Rejected            string                    `json:"rejected"`
	RejectionReason     string                    `json:"rejection_reason"`
	Salary              string                    `json:"salary"`
	TimeTypeNames       utilities.JsonStringSlice `json:"time_type_names"`
	TimeTypes           utilities.JsonStringSlice `json:"time_types"`
	WeekEnding          string                    `json:"week_ending"`
	WorkHours           float64                   `json:"work_hours"`
	WorkJobHours        float64                   `json:"work_job_hours"`
	WorkTotalHours      float64                   `json:"work_total_hours"`
	WorkWeekHours       float64                   `json:"work_week_hours"`
	GivenName           string                    `json:"given_name"`
	Surname             string                    `json:"surname"`
	Approver            string                    `json:"approver"`
	Committer           string                    `json:"committer"`
	Committed           string                    `json:"committed"`
	ApproverName        string                    `json:"approver_name"`
	CommitterName       string                    `json:"committer_name"`
}

// createTimesheetTalliesHandler returns a handler that creates a tally of the
// timesheets based on the supplied role and status filters.
func createTimesheetTalliesHandler(app core.App, role string, pendingOnly, approvedOnly int) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var timeSheetTally []TimeSheetTally
		err := app.DB().NewQuery(talliesQuery).Bind(dbx.Params{
			"uid":          e.Auth.Id,
			"role":         role,
			"pendingOnly":  pendingOnly,
			"approvedOnly": approvedOnly,
		}).All(&timeSheetTally)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		return e.JSON(http.StatusOK, timeSheetTally)
	}
}
