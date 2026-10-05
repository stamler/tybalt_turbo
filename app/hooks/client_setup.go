package hooks

import (
	"fmt"
	"net/http"

	"tybalt/errs"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// clientRecordUses lists the records that keep a reference to each client record.
// Deletion is refused while any exist, instead of letting PocketBase clear the
// optional ones. Contacts are not listed under clients: they cascade with their
// client, and their own uses are checked as they are deleted.
var clientRecordUses = map[string][]struct{ collection, field string }{
	"clients": {
		{"jobs", "client"},
		{"jobs", "job_owner"},
		{"client_invoicing_information", "client"},
	},
	"client_contacts": {
		{"jobs", "contact"},
		{"client_invoicing_information", "contact"},
	},
	"client_invoicing_information": {
		{"jobs", "invoicing_information"},
	},
}

// AddClientSetupHooks protects shared client records and their references.
// Who may write these collections is decided by their collection rules.
func AddClientSetupHooks(app core.App) {
	// The creator of an invoicing profile comes from the server, never the form.
	app.OnRecordCreateRequest("client_invoicing_information").BindFunc(func(e *core.RecordRequestEvent) error {
		creator := ""
		if e.Auth != nil && e.Auth.Collection().Name == "users" {
			creator = e.Auth.Id
		}
		e.Record.Set("creator", creator)
		return e.Next()
	})

	app.OnRecordUpdate("client_invoicing_information").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("creator") != e.Record.Original().GetString("creator") {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "the creator of an invoicing profile cannot be changed",
				Data: map[string]errs.CodeError{
					"creator": {Code: "not_editable", Message: "the creator of an invoicing profile cannot be changed"},
				},
			}
		}
		return e.Next()
	})

	validateReferences := func(e *core.RecordEvent) error {
		if err := utilities.ValidateClientReferences(e.App, e.Record); err != nil {
			return err
		}
		return e.Next()
	}
	app.OnRecordCreate("client_contacts", "client_invoicing_information").BindFunc(validateReferences)
	app.OnRecordUpdate("client_contacts", "client_invoicing_information").BindFunc(validateReferences)

	app.OnRecordDelete("clients", "client_contacts", "client_invoicing_information").BindFunc(func(e *core.RecordEvent) error {
		for _, use := range clientRecordUses[e.Record.Collection().Name] {
			count, err := e.App.CountRecords(use.collection, dbx.HashExp{use.field: e.Record.Id})
			if err != nil {
				return err
			}
			if count > 0 {
				return &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: fmt.Sprintf("this record is still used by %s", use.collection),
				}
			}
		}
		return e.Next()
	})
}
