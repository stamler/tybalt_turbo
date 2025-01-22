package hooks

import (
	"errors"

	"github.com/pocketbase/pocketbase/core"
)

type HookError struct {
	Status  int                  `json:"status"` // Represents the HTTP status code
	Message string               `json:"message"`
	Data    map[string]CodeError `json:"data"`
}

func (e *HookError) Error() string {
	return e.Message
}

func AnnotateHookError(app core.App, e *core.RecordRequestEvent, err error) error {
	var hookErr *HookError
	if errors.As(err, &hookErr) {
		e.JSON(hookErr.Status, hookErr)
	}
	return err
}

// CodeError is a custom error type that includes a code
type CodeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func (e *CodeError) Error() string {
	return e.Message
}

// This file exports the hooks that are available to the PocketBase application.
// The hooks are called at various points in the application lifecycle.

// To begin we have a series of OnRecordBeforeCreateRequest and
// OnRecordBeforeUpdateRequest hooks for the time_entries model. These hooks are
// called before a record is created or updated in the time_entries collection.

func AddHooks(app core.App) {
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
}
