package hooks

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// This file exports the hooks that are available to the PocketBase application.
// The hooks are called at various points in the application lifecycle.

// To begin we have a series of OnRecordBeforeCreateRequest and
// OnRecordBeforeUpdateRequest hooks for the time_entries model. These hooks are
// called before a record is created or updated in the time_entries collection.

func AddHooks(app *pocketbase.PocketBase) {
	// hooks for time_entries model
	app.OnRecordBeforeCreateRequest("time_entries").Add(func(e *core.RecordCreateEvent) error {
		if err := ProcessTimeEntry(app, e.Record, e.HttpContext); err != nil {
			return err
		}
		return nil
	})
	app.OnRecordBeforeUpdateRequest("time_entries").Add(func(e *core.RecordUpdateEvent) error {
		if err := ProcessTimeEntry(app, e.Record, e.HttpContext); err != nil {
			return err
		}
		return nil
	})
}
