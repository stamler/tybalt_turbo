package hooks

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"tybalt/constants"
	"tybalt/errs"
	"tybalt/notifications"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func AnnotateHookError(app core.App, e *core.RecordRequestEvent, err error) error {
	var hookErr *errs.HookError
	if errors.As(err, &hookErr) {
		e.JSON(hookErr.Status, hookErr)
	}
	return err
}

// This file exports the hooks that are available to the PocketBase application.
// The hooks are called at various points in the application lifecycle.

// To begin we have a series of OnRecordBeforeCreateRequest and
// OnRecordBeforeUpdateRequest hooks for the time_entries model. These hooks are
// called before a record is created or updated in the time_entries collection.

func AddHooks(app core.App) {
	// OnRecordAuthRequest fires after successful credential verification but before
	// returning the auth token to the client. We use it to:
	// 1. Block inactive accounts from logging in
	// 2. Auto-create admin_profiles for first-time users
	//
	// Control flow: returning an error aborts the auth flow and sends the error to
	// the client (no token issued). Calling e.Next() continues the chain and
	// completes the login (token issued).
	app.OnRecordAuthRequest("users").BindFunc(func(e *core.RecordAuthRequestEvent) error {
		uid := e.Record.Id

		// Try to find existing admin_profiles record
		adminProfile, err := e.App.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": uid})
		if err == nil {
			// Record exists - check if active. Returning error here prevents login
			// even though credentials were valid.
			if !adminProfile.GetBool("active") {
				return apis.NewForbiddenError("this account is inactive", nil)
			}
			return e.Next()
		}

		// No admin_profiles record exists - create one with sensible defaults
		adminProfiles, err := e.App.FindCollectionByNameOrId("admin_profiles")
		if err != nil {
			return err
		}
		rec := core.NewRecord(adminProfiles)
		rec.Set("uid", uid)
		rec.Set("active", true) // New users are active by default
		rec.Set("work_week_hours", constants.DEFAULT_WORK_WEEK_HOURS)
		rec.Set("default_charge_out_rate", constants.DEFAULT_CHARGE_OUT_RATE)
		rec.Set("skip_min_time_check", "no")
		rec.Set("salary", false)
		rec.Set("untracked_time_off", false)
		rec.Set("time_sheet_expected", false)
		rec.Set("default_branch", constants.DEFAULT_BRANCH_ID)
		// payroll_id must match ^(?:[1-9]\d*|CMS[0-9]{1,2})$
		rec.Set("payroll_id", "999999")

		if err := e.App.Save(rec); err != nil {
			// ignore race where another process created it first
			if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return fmt.Errorf("failed to create admin_profile for %s: %w", uid, err)
			}
		}

		return e.Next()
	})

	// hooks for time_entries model
	app.OnRecordCreateRequest("time_entries").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := ProcessTimeEntry(app, e); err != nil {
			return AnnotateHookError(app, e, err)
		}
		return e.Next()
	})
	app.OnRecordUpdateRequest("time_entries").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := ProcessTimeEntry(app, e); err != nil {
			return AnnotateHookError(app, e, err)
		}
		return e.Next()
	})
	// hooks for time_amendments model
	app.OnRecordCreateRequest("time_amendments").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := ProcessTimeAmendment(app, e); err != nil {
			return err
		}
		return e.Next()
	})
	app.OnRecordUpdateRequest("time_amendments").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := ProcessTimeAmendment(app, e); err != nil {
			return err
		}
		return e.Next()
	})
	// hooks for purchase_orders model
	app.OnRecordCreateRequest("purchase_orders").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := ProcessPurchaseOrder(app, e); err != nil {
			return AnnotateHookError(app, e, err)
		}
		return e.Next()
	})
	app.OnRecordUpdateRequest("purchase_orders").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := ProcessPurchaseOrder(app, e); err != nil {
			return AnnotateHookError(app, e, err)
		}
		return e.Next()
	})
	// hooks for jobs model
	app.OnRecordCreateRequest("jobs").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := ProcessJob(app, e); err != nil {
			return AnnotateHookError(app, e, err)
		}
		return e.Next()
	})
	app.OnRecordUpdateRequest("jobs").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := ProcessJob(app, e); err != nil {
			return AnnotateHookError(app, e, err)
		}
		return e.Next()
	})
	// hooks for profiles model
	app.OnRecordCreateRequest("profiles").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := ProcessProfile(app, e); err != nil {
			return AnnotateHookError(app, e, err)
		}
		return e.Next()
	})
	app.OnRecordUpdateRequest("profiles").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := ProcessProfile(app, e); err != nil {
			return AnnotateHookError(app, e, err)
		}
		return e.Next()
	})
	// hooks for clients model
	app.OnRecordCreateRequest("clients").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := ProcessClient(app, e); err != nil {
			return AnnotateHookError(app, e, err)
		}
		return e.Next()
	})
	app.OnRecordUpdateRequest("clients").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := ProcessClient(app, e); err != nil {
			return AnnotateHookError(app, e, err)
		}
		return e.Next()
	})
	// hooks for client_notes model
	app.OnRecordCreateRequest("client_notes").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := ProcessClientNote(app, e); err != nil {
			return AnnotateHookError(app, e, err)
		}
		return e.Next()
	})
	// hooks for po_approver_props model
	app.OnRecordCreateRequest("po_approver_props").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := ProcessPOApproverProps(app, e); err != nil {
			return AnnotateHookError(app, e, err)
		}
		return e.Next()
	})
	app.OnRecordUpdateRequest("po_approver_props").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := ProcessPOApproverProps(app, e); err != nil {
			return AnnotateHookError(app, e, err)
		}
		return e.Next()
	})
	// hooks for expenses model
	app.OnRecordCreateRequest("expenses").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := ProcessExpense(app, e); err != nil {
			return AnnotateHookError(app, e, err)
		}
		return e.Next()
	})
	app.OnRecordUpdateRequest("expenses").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := ProcessExpense(app, e); err != nil {
			return AnnotateHookError(app, e, err)
		}
		return e.Next()
	})

	// For notification hooks, we need to use the RecordEvent instead of the
	// RecordRequestEvent because the notifications model is not exposed to the
	// API.
	app.OnRecordCreate("notifications").BindFunc(func(e *core.RecordEvent) error {
		if err := notifications.WriteStatusUpdated(app, e); err != nil {
			return err
		}
		return e.Next()
	})
	app.OnRecordUpdate("notifications").BindFunc(func(e *core.RecordEvent) error {
		if err := notifications.WriteStatusUpdated(app, e); err != nil {
			return err
		}
		return e.Next()
	})

	// Hook for time_sheet_reviewers: validate that reviewer is an active user
	app.OnRecordCreateRequest("time_sheet_reviewers").BindFunc(func(e *core.RecordRequestEvent) error {
		reviewerUID := e.Record.GetString("reviewer")
		active, err := utilities.IsUserActive(e.App, reviewerUID)
		if err != nil {
			hookErr := &errs.HookError{
				Status:  http.StatusInternalServerError,
				Message: "failed to check reviewer active status",
			}
			return AnnotateHookError(e.App, e, hookErr)
		}
		if !active {
			hookErr := &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "reviewer must be an active user",
				Data: map[string]errs.CodeError{
					"reviewer": {Code: "reviewer_not_active", Message: "the selected reviewer is not an active user"},
				},
			}
			return AnnotateHookError(e.App, e, hookErr)
		}
		return e.Next()
	})

	// Hook for time_sheet_reviewers: when a new reviewer is added, send notification
	// Note: core.RecordEvent doesn't provide auth context, so we can't get the creator
	// directly from the event. We load the timesheet to get the approver (who is the only
	// one who can share), and we also need the timesheet for notification data (week_ending,
	// employee name, etc.) anyway.
	app.OnRecordCreate("time_sheet_reviewers").BindFunc(func(e *core.RecordEvent) error {
		timesheetId := e.Record.GetString("time_sheet")
		reviewerUID := e.Record.GetString("reviewer")

		// Get the timesheet record (needed for approver and notification data)
		timesheet, err := app.FindRecordById("time_sheets", timesheetId)
		if err != nil {
			app.Logger().Error(
				"error finding timesheet for reviewer notification",
				"timesheet_id", timesheetId,
				"error", err,
			)
			// Don't fail the request if notification fails
			return e.Next()
		}

		// Get the approver from the timesheet (only the approver can share, so this is accurate)
		sharerUID := timesheet.GetString("approver")

		// Queue notification for the new reviewer
		if notifErr := notifications.QueueTimesheetSharedNotifications(app, timesheet, sharerUID, []string{reviewerUID}); notifErr != nil {
			app.Logger().Error(
				"error queueing timesheet shared notification",
				"timesheet_id", timesheetId,
				"reviewer_uid", reviewerUID,
				"error", notifErr,
			)
			// Don't fail the request if notification fails
		}

		return e.Next()
	})
}
