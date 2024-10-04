// this file implements cleaning and validation rules for the time_entries collection

package hooks

import (
	"encoding/json"
	"log"
	"regexp"
	"tybalt/utilities"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/list"
	"github.com/pocketbase/pocketbase/tools/types"
)

// The cleanTimeEntry function is used to remove properties from the time_entry
// record that are not allowed to be set based on the value of the record's
// time_type property. This is intended to reduce round trips to the database
// and to ensure that the record is in a valid state before it is created or
// updated. It is called by ProcessTimeEntry to reduce the number of fields
// that need to be validated.
func cleanTimeEntry(app *pocketbase.PocketBase, timeEntryRecord *models.Record) ([]string, error) {
	timeTypeId := timeEntryRecord.GetString("time_type")

	// Load the allowed fields for the time_type from the time_types collection in
	// PocketBase. They are stored in the allowed_fields property as a JSON array
	// of strings.
	timeTypeRecord, findError := app.Dao().FindRecordById("time_types", timeTypeId)
	if findError != nil {
		return nil, findError
	}

	// Get the allowed fields from the time_type record with type assertion
	var allowedFields []string
	allowedFieldsJson := timeTypeRecord.Get("allowed_fields").(types.JsonRaw)

	// use json.Unmarshal to convert the JSON array to a Go slice of strings
	if unmarshalErrorAllowed := json.Unmarshal(allowedFieldsJson, &allowedFields); unmarshalErrorAllowed != nil {
		log.Fatalf("Error unmarshalling JSON: %v", unmarshalErrorAllowed)
		return nil, unmarshalErrorAllowed
	}

	// Get the required fields from the time_type record with type assertion
	var requiredFields []string
	requiredFieldsJson := timeTypeRecord.Get("required_fields").(types.JsonRaw)

	// if requiredFieldsJson has a value of null, "{}", or "[]" then all fields
	// are required so we set the requiredFields to the allowedFields.
	if requiredFieldsJson.String() == "null" || requiredFieldsJson.String() == "[]" || requiredFieldsJson.String() == "{}" {
		requiredFields = allowedFields
	} else if unmarshalErrorRequired := json.Unmarshal(requiredFieldsJson, &requiredFields); unmarshalErrorRequired != nil {
		log.Fatalf("Error unmarshalling JSON: %v", unmarshalErrorRequired)
		return nil, unmarshalErrorRequired
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

	return requiredFields, nil
}

// cross-field validation is performed in this function. It is expected that the
// time_entry record has already been cleaned by the cleanTimeEntry function.
// This ensures that only the fields that are allowed to be set are present in
// the record prior to validation. The requiredFields slice is used to validate
// the presence of required fields. The function returns an error if the record
// is invalid, otherwise it returns nil.
func validateTimeEntry(timeEntryRecord *models.Record, requiredFields []string) error {
	jobIsPresent := timeEntryRecord.Get("job") != ""
	totalHours := timeEntryRecord.GetFloat("hours") + timeEntryRecord.GetFloat("meals_hours")

	// validation is performed in two passes. The first pass is to validate the
	// presence of required fields. We do this by using the validation.Required
	// function on each required field from the requiredFields slice using
	// validation.Errors with the field name as the key.
	requiredValidationsErrors := validation.Errors{}
	for _, field := range requiredFields {
		requiredValidationsErrors[field] = validation.Validate(timeEntryRecord.Get(field), validation.Required.Error("Value required"))
	}

	// Check for errors in the first pass
	if err := requiredValidationsErrors.Filter(); err != nil {
		return err
	}

	// The second pass performs everything else (cross-field validation, field
	// values, etc.)
	otherValidationsErrors := validation.Errors{
		"hours":                 validation.Validate(timeEntryRecord.Get("hours"), validation.By(utilities.IsPositiveMultipleOfPointFive())),
		"date":                  validation.Validate(timeEntryRecord.Get("date"), validation.By(utilities.IsValidDate)),
		"global":                validation.Validate(totalHours, validation.Max(18.0).Error("Total hours must not exceed 18")),
		"meals_hours":           validation.Validate(timeEntryRecord.Get("meals_hours"), validation.Max(3.0).Error("Meals Hours must not exceed 3")),
		"description":           validation.Validate(timeEntryRecord.Get("description"), validation.Length(5, 0).Error("must be at least 5 characters")),
		"work_record":           validation.Validate(timeEntryRecord.Get("work_record"), validation.When(jobIsPresent, validation.Match(regexp.MustCompile("^[FKQ][0-9]{2}-[0-9]{3,4}(-[0-9]+)?$")).Error("must be in the correct format")).Else(validation.In("").Error("Work Record must be empty when job is not provided"))),
		"payout_request_amount": validation.Validate(timeEntryRecord.Get("payout_request_amount"), validation.Min(0.0).Exclusive().Error("Amount must be greater than 0"), validation.By(utilities.IsPositiveMultipleOfPointZeroOne())),
	}.Filter()

	return otherValidationsErrors
}

// The ProcessTimeEntry function is used to validate the time_entry record
// before it is created or updated. A lot of the work is done by PocketBase
// itself so this is for cross-field validation. If the time_entry record is
// invalid this function throws an error explaining which field(s) are invalid
// and why.
func ProcessTimeEntry(app *pocketbase.PocketBase, record *models.Record, context echo.Context) error {
	// get the auth record from the context
	authRecord := context.Get(apis.ContextAuthRecordKey).(*models.Record)

	// If the uid property is not equal to the authenticated user's uid, return an
	// error.
	if record.GetString("uid") != authRecord.Id {
		return apis.NewApiError(400, "uid property must be equal to the authenticated user's id", map[string]validation.Error{})
	}

	// set properties to nil if they are not allowed to be set based on the
	// time_type
	requiredFields, cleanErr := cleanTimeEntry(app, record)
	if cleanErr != nil {
		return apis.NewBadRequestError("Error cleaning time_entry record", cleanErr)
	}

	// write the week_ending property to the record. This is derived exclusively
	// from the date property.
	weekEnding, wkEndErr := utilities.GenerateWeekEnding(record.GetString("date"))
	if wkEndErr != nil {
		return apis.NewBadRequestError("Error generating week_ending", wkEndErr)
	}
	record.Set("week_ending", weekEnding)

	// perform the validation for the time_entry record. In this step we also
	// write the uid property to the record so that we can validate it against the
	if validationErr := validateTimeEntry(record, requiredFields); validationErr != nil {
		return apis.NewBadRequestError("Validation error", validationErr)
	}

	return nil
}
