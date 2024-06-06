package hooks

import (
	"encoding/json"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/list"
	"github.com/pocketbase/pocketbase/tools/types"
)

// This file exports the hooks that are available to the PocketBase application.
// The hooks are called at various points in the application lifecycle.

// To begin we have a series of OnRecordBeforeCreateRequest and
// OnRecordBeforeUpdateRequest hooks for the time_entries model. These hooks are
// called before a record is created or updated in the time_entries collection.

// The cleanTimeEntry function is used to remove properties from the time_entry
// record that are not allowed to be set based on the value of the record's
// time_type property. This is intended to reduce round trips to the database
// and to ensure that the record is in a valid state before it is created or
// updated. It is called by validateTimeEntry to reduce the number of fields
// that need to be validated.
func cleanTimeEntry(app *pocketbase.PocketBase, timeEntryRecord *models.Record) error {
	timeTypeId := timeEntryRecord.GetString("time_type")

	// Load the allowed fields for the time_type from the time_types collection in
	// PocketBase. They are stored in the allowed_fields property as a JSON array
	// of strings.
	timeTypeRecord, findError := app.Dao().FindRecordById("time_types", timeTypeId)
	if findError != nil {
		return findError
	}

	// Get the allowed fields from the time_type record with type assertion
	var allowedFields []string
	allowedFieldsJson := timeTypeRecord.Get("allowed_fields").(types.JsonRaw)

	// use json.Unmarshal to convert the JSON array to a Go slice of strings
	if unmarshalError := json.Unmarshal(allowedFieldsJson, &allowedFields); unmarshalError != nil {
		log.Fatalf("Error unmarshalling JSON: %v", unmarshalError)
		return unmarshalError
	}

	// Certain fields are always allowed to be set. We add them to the list of
	// allowed fields here.
	allowedFields = append(allowedFields, "id", "uid", "created", "updated")

	// remove any fields from the time_entry record that are not in allowedFields.
	// I'm not sure if this is the best way to do this but let's try it.
	for key := range timeEntryRecord.ColumnValueMap() {
		if !list.ExistInSlice(key, allowedFields) {
			log.Println("Removing field: ", key)
			timeEntryRecord.Set(key, nil)
		}
	}

	return nil
}

// The ValidateTimeEntry function is used to validate the time_entry record
// before it is created or updated. A lot of the work is done by PocketBase
// itself so this is for cross-field validation. If the time_entry record is
// invalid this function throws an error explaining which field(s) are invalid
// and why.
func ValidateTimeEntry(app *pocketbase.PocketBase, record *models.Record) error {
	// set properties to nil if they are not allowed to be set based on the
	// time_type
	if cleanErr := cleanTimeEntry(app, record); cleanErr != nil {
		return cleanErr
	}

	// Perform validation here
	return nil
}

func AddHooks(app *pocketbase.PocketBase) {
	// hooks for time_entries model
	app.OnRecordBeforeCreateRequest("time_entries").Add(func(e *core.RecordCreateEvent) error {
		if err := ValidateTimeEntry(app, e.Record); err != nil {
			// return the error to the client
			return err
		}
		return nil
	})
	app.OnRecordBeforeUpdateRequest("time_entries").Add(func(e *core.RecordUpdateEvent) error {
		if err := ValidateTimeEntry(app, e.Record); err != nil {
			// return the error to the client
			return err
		}
		return nil
	})
}
