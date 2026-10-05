package utilities

import (
	"fmt"
	"net/http"

	"tybalt/errs"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type clientRelation struct {
	collection string
	field      string
}

// clientRelationsFrom lists the client-owned records each collection points at.
var clientRelationsFrom = map[string][]clientRelation{
	"jobs": {
		{"client_contacts", "contact"},
		{"client_invoicing_information", "invoicing_information"},
	},
	"client_invoicing_information": {
		{"client_contacts", "contact"},
	},
}

// clientRelationsTo lists the records that point at each client-owned collection.
var clientRelationsTo = map[string][]clientRelation{
	"client_contacts": {
		{"jobs", "contact"},
		{"client_invoicing_information", "contact"},
	},
	"client_invoicing_information": {
		{"jobs", "invoicing_information"},
	},
}

func clientReferenceError(field, code, message string) error {
	return &errs.HookError{
		Status:  http.StatusBadRequest,
		Message: message,
		Data:    map[string]errs.CodeError{field: {Code: code, Message: message}},
	}
}

// ValidateClientReferences checks that a job, contact or invoicing profile agrees
// with its related records about the client: the contact and invoicing profile
// it selects, and the jobs and profiles that select it, must all belong to its
// client. Empty relations are left to required-field validation.
func ValidateClientReferences(app core.App, record *core.Record) error {
	collection := record.Collection().Name
	client := record.GetString("client")

	for _, relation := range clientRelationsFrom[collection] {
		id := record.GetString(relation.field)
		if id == "" {
			continue
		}
		related, err := app.FindRecordById(relation.collection, id)
		if err != nil {
			return clientReferenceError(relation.field, "invalid_reference", "The selected record was not found.")
		}
		if related.GetString("client") != client {
			return clientReferenceError(relation.field, "client_mismatch", "The selected record must belong to this client.")
		}
	}

	if record.IsNew() {
		return nil
	}
	for _, relation := range clientRelationsTo[collection] {
		count, err := app.CountRecords(relation.collection, dbx.HashExp{relation.field: record.Id}, dbx.Not(dbx.HashExp{"client": client}))
		if err != nil {
			return err
		}
		if count > 0 {
			return clientReferenceError("client", "client_mismatch", fmt.Sprintf("This record is used by %s for another client.", relation.collection))
		}
	}
	return nil
}
