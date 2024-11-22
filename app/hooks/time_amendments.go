// this file implements cleaning and validation rules for the time_amendments collection

package hooks

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"tybalt/utilities"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/list"
	"github.com/pocketbase/pocketbase/tools/types"
)

// The cleanTimeAmendment function is used to remove properties from the
// time_amendment record that are not allowed to be set based on the value of
// the record's time_type property. This is intended to reduce round trips to
// the database and to ensure that the record is in a valid state before it is
// created or updated. It is called by ProcessTimeAmendment to reduce the number
// of fields that need to be validated.
func cleanTimeAmendment(app core.App, timeAmendmentRecord *models.Record) ([]string, error) {
	timeTypeId := timeAmendmentRecord.GetString("time_type")

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
	allowedFields = append(allowedFields, "id", "uid", "created", "creator", "updated", "skip_tsid_check")

	// remove any fields from the time_amendment record that are not in
	// allowedFields. I'm not sure if this is the best way to do this but let's
	// try it.
	for key := range timeAmendmentRecord.ColumnValueMap() {
		if !list.ExistInSlice(key, allowedFields) {
			log.Println("Removing field: ", key)
			timeAmendmentRecord.Set(key, nil)
		}
	}

	return requiredFields, nil
}

// cross-field validation is performed in this function. It is expected that the
// time_amendment record has already been cleaned by the cleanTimeAmendment
// function. This ensures that only the fields that are allowed to be set are
// present in the record prior to validation. The requiredFields slice is used
// to validate the presence of required fields. The function returns an error if
// the record is invalid, otherwise it returns nil.
func validateTimeAmendment(timeAmendmentRecord *models.Record, requiredFields []string) error {
	jobIsPresent := timeAmendmentRecord.Get("job") != ""
	totalHours := timeAmendmentRecord.GetFloat("hours") + timeAmendmentRecord.GetFloat("meals_hours")

	// validation is performed in two passes. The first pass is to validate the
	// presence of required fields. We do this by using the validation.Required
	// function on each required field from the requiredFields slice using
	// validation.Errors with the field name as the key.
	requiredValidationsErrors := validation.Errors{}
	for _, field := range requiredFields {
		requiredValidationsErrors[field] = validation.Validate(timeAmendmentRecord.Get(field), validation.Required.Error("Value required"))
	}

	// Check for errors in the first pass
	if err := requiredValidationsErrors.Filter(); err != nil {
		return err
	}

	// The second pass performs everything else (cross-field validation, field
	// values, etc.)
	otherValidationsErrors := validation.Errors{
		"hours":                 validation.Validate(timeAmendmentRecord.Get("hours"), validation.By(utilities.IsPositiveMultipleOfPointFive())),
		"date":                  validation.Validate(timeAmendmentRecord.Get("date"), validation.By(utilities.IsValidDate)),
		"global":                validation.Validate(totalHours, validation.Max(18.0).Error("Total hours must not exceed 18")),
		"meals_hours":           validation.Validate(timeAmendmentRecord.Get("meals_hours"), validation.Max(3.0).Error("Meals Hours must not exceed 3")),
		"description":           validation.Validate(timeAmendmentRecord.Get("description"), validation.Length(5, 0).Error("must be at least 5 characters")),
		"work_record":           validation.Validate(timeAmendmentRecord.Get("work_record"), validation.When(jobIsPresent, validation.Match(regexp.MustCompile("^[FKQ][0-9]{2}-[0-9]{3,4}(-[0-9]+)?$")).Error("must be in the correct format")).Else(validation.In("").Error("Work Record must be empty when job is not provided"))),
		"payout_request_amount": validation.Validate(timeAmendmentRecord.Get("payout_request_amount"), validation.Min(0.0).Exclusive().Error("Amount must be greater than 0"), validation.By(utilities.IsPositiveMultipleOfPointZeroOne())),
	}.Filter()

	return otherValidationsErrors
}

// The ProcessTimeAmendment function is used to validate the time_amendment
// record before it is created or updated. A lot of the work is done by
// PocketBase itself so this is for cross-field validation. If the
// time_amendment record is invalid this function throws an error explaining
// which field(s) are invalid and why.
func ProcessTimeAmendment(app core.App, record *models.Record, context echo.Context) error {
	// get the auth record from the context
	authRecord := context.Get(apis.ContextAuthRecordKey).(*models.Record)

	// If the creator property is not equal to the authenticated user's id, return
	// an error.
	if record.GetString("creator") != authRecord.Id {
		return apis.NewApiError(400, "creator property must be equal to the authenticated user's id", map[string]validation.Error{
			"creator": validation.NewError(
				"creator_mismatch",
				"creator property must be equal to the authenticated user's id",
			),
		})
	}

	// set properties to nil if they are not allowed to be set based on the
	// time_type
	requiredFields, cleanErr := cleanTimeAmendment(app, record)
	if cleanErr != nil {
		return apis.NewBadRequestError("Error cleaning time_amendment record", cleanErr)
	}

	// write the week_ending property to the record. This is derived exclusively
	// from the date property.
	weekEnding, wkEndErr := utilities.GenerateWeekEnding(record.GetString("date"))
	if wkEndErr != nil {
		return apis.NewBadRequestError("Error generating week_ending", wkEndErr)
	}
	record.Set("week_ending", weekEnding)

	// write the tsid (Time Sheet ID) property to the record. We query the
	// time_sheets collection for the time_sheet record that matches the
	// weekEnding and uid properties then assign the id property of that record to
	// the tsid property of the time_amendment record.
	timeSheetRecords, timeSheetErr := app.Dao().FindRecordsByFilter("time_sheets", "uid={:userId} && week_ending={:weekEnding}", "", 0, 0, dbx.Params{
		"userId":     record.GetString("uid"),
		"weekEnding": weekEnding,
	})
	if timeSheetErr != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Error finding time_sheet record", map[string]validation.Error{
			"global": validation.NewError(
				"error_finding_time_sheet",
				"Error finding time_sheet record",
			),
		})
	}

	if !record.GetBool("skip_tsid_check") {
		// throw an error if no time_sheet record is found
		if len(timeSheetRecords) == 0 {
			return apis.NewApiError(http.StatusBadRequest, "No time_sheets record found, create one instead", map[string]validation.Error{
				"global": validation.NewError(
					"no_time_sheet",
					"No time_sheets record found, create one instead",
				),
			})

		}
		// throw an error if more than one time_sheet record is found
		if len(timeSheetRecords) > 1 {
			return apis.NewApiError(http.StatusInternalServerError, "More than one time_sheets record found", map[string]validation.Error{
				"global": validation.NewError(
					"multiple_time_sheets",
					"More than one time_sheets record found",
				),
			})
		}

		// throw if the found time_sheet record hasn't yet been committed, alerting
		// the user to instead create a time_entry record.
		if timeSheetRecords[0].GetDateTime("committed").IsZero() {
			return apis.NewApiError(http.StatusBadRequest, "Found time_sheets record has not been committed. Create a time_entries record instead.", map[string]validation.Error{
				"global": validation.NewError(
					"time_sheet_not_committed",
					"Found time_sheets record has not been committed. Create a time_entries record instead.",
				),
			})
		}

		// set the tsid property to the id of the found time_sheet record
		record.Set("tsid", timeSheetRecords[0].Id)
	}

	// perform the validation for the time_amendment record. In this step we also
	// write the uid property to the record so that we can validate it against the
	if validationErr := validateTimeAmendment(record, requiredFields); validationErr != nil {
		return apis.NewBadRequestError("Validation error", validationErr)
	}

	return nil
}
