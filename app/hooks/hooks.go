package hooks

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

// CodeError is a custom error type that includes a code
type CodeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
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
	app.OnRecordBeforeCreateRequest("purchase_orders").Add(func(e *core.RecordCreateEvent) error {
		if err := ProcessPurchaseOrder(app, e.Record, e.HttpContext); err != nil {
			return err
		}
		return nil
	})
	app.OnRecordBeforeUpdateRequest("purchase_orders").Add(func(e *core.RecordUpdateEvent) error {
		if err := ProcessPurchaseOrder(app, e.Record, e.HttpContext); err != nil {
			return err
		}
		return nil
	})
	app.OnRecordBeforeCreateRequest("expenses").Add(func(e *core.RecordCreateEvent) error {
		if err := ProcessExpense(app, e.Record, e.HttpContext); err != nil {
			// Check if the error is a CodeError and return the appropriate JSON response
			if codeError, ok := err.(*CodeError); ok {
				return e.HttpContext.JSON(http.StatusBadRequest, map[string]interface{}{
					"message": codeError.Message,
					"code":    codeError.Code,
				})
			}
			return err
		}
		return nil
	})
	app.OnRecordBeforeUpdateRequest("expenses").Add(func(e *core.RecordUpdateEvent) error {
		if err := ProcessExpense(app, e.Record, e.HttpContext); err != nil {
			// Check if the error is a CodeError and return the appropriate JSON response
			if codeError, ok := err.(*CodeError); ok {
				return e.HttpContext.JSON(http.StatusBadRequest, map[string]interface{}{
					"message": codeError.Message,
					"code":    codeError.Code,
				})
			}
			return err
		}
		return nil
	})
}
