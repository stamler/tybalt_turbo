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

	// Request hooks return these errors with their field codes; the model hooks
	// also guard saves made outside requests, such as from the dashboard or code.
	guard := func(check func(core.App, *core.Record) error, collection string, create bool) {
		requestGuard := func(e *core.RecordRequestEvent) error {
			if err := check(e.App, e.Record); err != nil {
				return AnnotateHookError(e.App, e, err)
			}
			return e.Next()
		}
		modelGuard := func(e *core.RecordEvent) error {
			if err := check(e.App, e.Record); err != nil {
				return err
			}
			return e.Next()
		}
		if create {
			app.OnRecordCreateRequest(collection).BindFunc(requestGuard)
			app.OnRecordCreate(collection).BindFunc(modelGuard)
		}
		app.OnRecordUpdateRequest(collection).BindFunc(requestGuard)
		app.OnRecordUpdate(collection).BindFunc(modelGuard)
	}
	guard(utilities.ValidateBillingName, "client_invoicing_information", true)
	guard(validateAliasRemoval, "clients", false)

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

// validateAliasRemoval keeps a client's alias while invoicing profiles bill it,
// so clearing the alias never silently changes the name those profiles bill.
func validateAliasRemoval(app core.App, client *core.Record) error {
	if client.IsNew() || utilities.ClientAlias(client) != "" || utilities.ClientAlias(client.Original()) == "" {
		return nil
	}
	count, err := utilities.CountProfilesBillingAlias(app, client.Id)
	if err != nil {
		return err
	}
	if count == 0 {
		return nil
	}
	message := "An invoicing profile bills this alias. Change it to bill the client name first."
	if count > 1 {
		message = fmt.Sprintf("%d invoicing profiles bill this alias. Change them to bill the client name first.", count)
	}
	return &errs.HookError{
		Status:  http.StatusBadRequest,
		Message: message,
		Data:    map[string]errs.CodeError{"alias": {Code: "alias_in_use", Message: message}},
	}
}
