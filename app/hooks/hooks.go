package hooks

import (
	"errors"
	"fmt"
	"strings"
	"tybalt/constants"
	"tybalt/errs"
	"tybalt/notifications"

	"github.com/pocketbase/dbx"
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
	// ensure an admin_profiles record exists for each authenticated user
	app.OnRecordAuthRequest("users").BindFunc(func(e *core.RecordAuthRequestEvent) error {
		uid := e.Record.Id

		// if the admin_profiles record already exists, nothing to do
		if _, err := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": uid}); err == nil {
			return e.Next()
		}

		// create the admin_profiles record with sensible defaults
		adminProfiles, err := app.FindCollectionByNameOrId("admin_profiles")
		if err != nil {
			return err
		}
		rec := core.NewRecord(adminProfiles)
		rec.Set("uid", uid)
		rec.Set("work_week_hours", constants.DEFAULT_WORK_WEEK_HOURS)
		rec.Set("default_charge_out_rate", constants.DEFAULT_CHARGE_OUT_RATE)
		rec.Set("skip_min_time_check", "no")
		rec.Set("salary", false)
		rec.Set("untracked_time_off", false)
		rec.Set("time_sheet_expected", false)
		rec.Set("default_branch", constants.DEFAULT_BRANCH_ID)
		// payroll_id must match ^(?:[1-9]\d*|CMS[0-9]{1,2})$
		rec.Set("payroll_id", "999999")

		if err := app.Save(rec); err != nil {
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
			return err
		}
		return e.Next()
	})
	app.OnRecordUpdateRequest("time_entries").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := ProcessTimeEntry(app, e); err != nil {
			return err
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
}
