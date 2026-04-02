package routes

import (
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type TimesheetApprover struct {
	ApproverName  string `json:"approver_name"`
	ApprovedDate  string `json:"approved_date"`
	CommitterName string `json:"committer_name"`
	CommittedDate string `json:"committed_date"`
	RejectorName  string `json:"rejector_name"`
	RejectedDate  string `json:"rejected_date"`
}

func getTimesheetApproverInfo(app core.App, timesheetId string) (TimesheetApprover, error) {
	query := `
		SELECT 
			COALESCE(p.given_name || ' ' || p.surname, '')  AS approver_name,
			ts.approved                                         AS approved_date,
			COALESCE(cp.given_name || ' ' || cp.surname, '') AS committer_name,
			ts.committed                                        AS committed_date,
			COALESCE(rp.given_name || ' ' || rp.surname, '') AS rejector_name,
			ts.rejected                                         AS rejected_date
		FROM time_sheets ts
		LEFT JOIN profiles p  ON ts.approver = p.uid
		LEFT JOIN profiles cp ON ts.committer = cp.uid
		LEFT JOIN profiles rp ON ts.rejector  = rp.uid
		WHERE ts.id = {:timesheetId}
	`

	var result TimesheetApprover
	err := app.DB().NewQuery(query).Bind(dbx.Params{
		"timesheetId": timesheetId,
	}).One(&result)
	if err != nil {
		return TimesheetApprover{
			ApproverName:  "",
			ApprovedDate:  "",
			CommitterName: "",
			CommittedDate: "",
			RejectorName:  "",
			RejectedDate:  "",
		}, nil
	}

	return result, nil
}

// createTimesheetApproverHandler returns a function that gets approver info for a specific timesheet
func createTimesheetApproverHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		timesheetId := e.Request.PathValue("id")
		if timesheetId == "" {
			return e.Error(http.StatusBadRequest, "timesheet id is required", nil)
		}
		result, err := getTimesheetApproverInfo(app, timesheetId)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load approver info", err)
		}

		return e.JSON(http.StatusOK, result)
	}
}
