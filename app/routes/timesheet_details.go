package routes

import (
	"net/http"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type TimeSheetDetailsResponse struct {
	TimeSheet    *core.Record      `json:"timeSheet"`
	Items        []*core.Record    `json:"items"`
	ApproverInfo TimesheetApprover `json:"approverInfo"`
}

func createGetTimeSheetDetailsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth
		if auth == nil {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		id := e.Request.PathValue("id")
		if id == "" {
			return e.Error(http.StatusBadRequest, "id is required", nil)
		}

		timeSheet, err := app.FindRecordById("time_sheets", id)
		if err != nil {
			return e.Error(http.StatusNotFound, "time sheet not found or not authorized", err)
		}

		allowed, err := canViewTimeSheetDetails(app, auth, timeSheet)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to check time sheet details permissions", err)
		}
		if !allowed {
			return e.Error(http.StatusNotFound, "time sheet not found or not authorized", nil)
		}

		items, err := app.FindRecordsByFilter("time_entries", "tsid={:tsid}", "-date", 0, 0, dbx.Params{
			"tsid": id,
		})
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load time entries", err)
		}

		if errs := app.ExpandRecords(items, []string{"job", "time_type", "division", "category"}, nil); len(errs) > 0 {
			return e.Error(http.StatusInternalServerError, "failed to expand time entries", nil)
		}

		approverInfo, err := getTimesheetApproverInfo(app, id)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load approver info", err)
		}

		return e.JSON(http.StatusOK, TimeSheetDetailsResponse{
			TimeSheet:    timeSheet,
			Items:        items,
			ApproverInfo: approverInfo,
		})
	}
}

func canViewTimeSheetDetails(app core.App, auth *core.Record, timeSheet *core.Record) (bool, error) {
	if auth == nil || timeSheet == nil {
		return false, nil
	}

	if auth.Id == timeSheet.GetString("uid") {
		return true, nil
	}

	hasAdmin, err := utilities.HasClaim(app, auth, "admin")
	if err != nil {
		return false, err
	}
	if hasAdmin && !timeSheet.GetDateTime("committed").IsZero() {
		return true, nil
	}

	if timeSheet.GetBool("submitted") && auth.Id == timeSheet.GetString("approver") {
		return true, nil
	}

	reviewers, err := app.FindRecordsByFilter("time_sheet_reviewers", "time_sheet={:time_sheet} && reviewer={:reviewer}", "", 1, 0, dbx.Params{
		"time_sheet": timeSheet.Id,
		"reviewer":   auth.Id,
	})
	if err != nil {
		return false, err
	}
	if len(reviewers) > 0 && timeSheet.GetBool("submitted") {
		return true, nil
	}

	hasCommit, err := utilities.HasClaim(app, auth, "commit")
	if err != nil {
		return false, err
	}
	if hasCommit && timeSheet.GetBool("submitted") && !timeSheet.GetDateTime("approved").IsZero() {
		return true, nil
	}

	hasReport, err := utilities.HasClaim(app, auth, "report")
	if err != nil {
		return false, err
	}
	if hasReport && !timeSheet.GetDateTime("committed").IsZero() {
		return true, nil
	}

	return false, nil
}
