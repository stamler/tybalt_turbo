package hooks

import (
	"encoding/json"
	"log"
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
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
// updated. It is called by ProcessTimeEntry to reduce the number of fields
// that need to be validated.
func cleanTimeEntry(app *pocketbase.PocketBase, timeEntryRecord *models.Record) (string, error) {
	timeTypeId := timeEntryRecord.GetString("time_type")

	// Load the allowed fields for the time_type from the time_types collection in
	// PocketBase. They are stored in the allowed_fields property as a JSON array
	// of strings.
	timeTypeRecord, findError := app.Dao().FindRecordById("time_types", timeTypeId)
	if findError != nil {
		return "", findError
	}

	// Get the allowed fields from the time_type record with type assertion
	var allowedFields []string
	allowedFieldsJson := timeTypeRecord.Get("allowed_fields").(types.JsonRaw)

	// use json.Unmarshal to convert the JSON array to a Go slice of strings
	if unmarshalError := json.Unmarshal(allowedFieldsJson, &allowedFields); unmarshalError != nil {
		log.Fatalf("Error unmarshalling JSON: %v", unmarshalError)
		return "", unmarshalError
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

	return timeTypeRecord.GetString("code"), nil
}

func isPositiveMultipleOfPointFive(fieldName string) validation.RuleFunc {
	return func(value interface{}) error {
		s, _ := value.(float64)
		if s < 0.5 {
			return validation.NewError("validation_smaller_than_point_five", fieldName+" must be at least 0.5")
		}
		// return error is s is not a multiple of 0.5
		if s/0.5 != float64(int(s/0.5)) {
			return validation.NewError("validation_not_multiple_of_point_five", fieldName+" must be a multiple of 0.5")
		}
		return nil
	}
}

// cross-field validation is performed in this function.
func validateTimeEntry(timeEntryRecord *models.Record, timeTypeCode string) error {
	isWorkTime := list.ExistInSlice(timeTypeCode, []string{"R", "RT"})
	jobIsPresent := timeEntryRecord.Get("job") != ""
	isOTO := timeTypeCode == "OTO"
	hoursRequired := list.ExistInSlice(timeTypeCode, []string{"OB", "OH", "OP", "OS", "OV", "RB"})
	hoursProhibited := list.ExistInSlice(timeTypeCode, []string{"OR", "OW", "OTO"})
	totalHours := timeEntryRecord.GetFloat("hours") + timeEntryRecord.GetFloat("job_hours") + timeEntryRecord.GetFloat("meals_hours")

	err := validation.Errors{
		"hours": validation.Validate(timeEntryRecord.Get("hours"),
			validation.When(isWorkTime && jobIsPresent, validation.In(0).Error("Hours must be 0 when a job is provided")),
			validation.When(hoursProhibited, validation.In(0).Error("Hours must be 0 when time_type is OR, OW or OTO")),
			validation.When((isWorkTime && !jobIsPresent) || hoursRequired, validation.By(isPositiveMultipleOfPointFive("Hours"))),
		),
		"global":                validation.Validate(totalHours, validation.Max(18.0).Error("Total hours must not exceed 18")),
		"division":              validation.Validate(timeEntryRecord.Get("division"), validation.When(isWorkTime, validation.Required.Error("A division is required when time_type is R or RT")).Else(validation.In("").Error("division must be empty when time_type is not R or RT"))),
		"meals_hours":           validation.Validate(timeEntryRecord.Get("meals_hours"), validation.When(isWorkTime, validation.Max(3.0).Error("Meals Hours must not exceed 3"))),
		"job_hours":             validation.Validate(timeEntryRecord.Get("job_hours"), validation.When(jobIsPresent, validation.By(isPositiveMultipleOfPointFive("Job Hours"))).Else(validation.In(0).Error("Job hours must be 0 when a job is not provided"))),
		"description":           validation.Validate(timeEntryRecord.Get("description"), validation.When(!hoursProhibited, validation.Required.Error("Description required"), validation.Length(5, 0).Error("Description must be at least 5 characters"))),
		"work_record":           validation.Validate(timeEntryRecord.Get("work_record"), validation.When(jobIsPresent, validation.Match(regexp.MustCompile("^[FKQ][0-9]{2}-[0-9]{3,}(-[0-9]+)?$")).Error("Work record must be in the correct format"))),
		"payout_request_amount": validation.Validate(timeEntryRecord.Get("payout_request_amount"), validation.When(isOTO, validation.Required.Error("Amount is required when time type is OTO")).Else(validation.Min(0).Exclusive().Error("Payout request amount must be greater than 0"))),
	}.Filter()

	return err

	// The following is the firestore rules function that we are trying to
	// replicate here. It is a simple example of cross-field validation.
	/*
		function validTimeEntry() {
			return

				// prevent salaried staff from submitting an RB entry
				(
					get(/databases/$(database)/documents/Profiles/$(request.auth.uid)).data.salary == false ||
					( get(/databases/$(database)/documents/Profiles/$(request.auth.uid)).data.salary == true && newDoc().timetype != "RB" )
				) &&

				// when job provided, job & division exist in db,
				// jobDescription and client are present, timetype is 'R' or 'RT'
				// hours field is missing
				(
					(
						// if a category property is specified, it must be a string and that
						// string must be in the categories list of the corresponding job
						// Document. if the category property is not specified, the job must not
						// have any categories
						(
							(isMissing("category") && !("categories" in get(/databases/$(database)/documents/Jobs/$(newDoc().job)).data)) ||
							(newDoc().category is string && newDoc().category in get(/databases/$(database)/documents/Jobs/$(newDoc().job)).data.categories)
						)
					) ||
					isMissing("job")
				) &&

				// at least one hours type is provided OR the timetype is "OR", "OW" or "OTO" or "RB"
				(
					(
						newDoc().timetype != "OW" &&
						newDoc().timetype != "OR" &&
						newDoc().timetype != "RB" &&
						newDoc().timetype != "OTO" &&
						!(newDoc().keys().hasAll(["payoutRequestAmount"])) && // payoutRequestAmount is not defined
						(
							(isPositiveMultipleOfPointFiveUnderEighteen("jobHours") && !(newDoc().timetype in ["OR", "OW"])) ||
							(isPositiveMultipleOfPointFiveUnderEighteen("hours") && !(newDoc().timetype in ["OR", "OW"])) ||
							(isPositiveMultipleOfPointFiveUnderEighteen("mealsHours") && !(newDoc().timetype in ["OR", "OW"]))
						)
					) ||
					(newDoc().timetype in ["OR", "OW"] && newDoc().keys().hasOnly(["date", "timetype", "timetypeName", "uid", "weekEnding"])) ||
					(newDoc().timetype == "OTO" && newDoc().keys().hasOnly(["date", "timetype", "timetypeName", "uid", "weekEnding", "payoutRequestAmount"]) && newDoc().payoutRequestAmount is number && newDoc().payoutRequestAmount > 0) ||
					(newDoc().timetype == "RB" && isPositiveMultipleOfPointFive("hours"))
				) &&

				// validate presence and length of description for Regular Hours and Training
				// also ensure workDescription does not contain jobNumbers of the format
				// XX-YYY where XX is a number between 15 and 40 and YYY is a 3-digit
				// number between 001 and 999
				(
					(newDoc().workDescription.size() > 4 && !newDoc().workDescription.matches('.*(1[5-9]|2[0-9]|3[0-9]|40)-([0-9]{3}).*') && newDoc().timetype in ["R", "RT"]) || !(newDoc().timetype in ["R", "RT"])
				) &&

				// ensure absence of description for Banking, Payout, Off and Off Rotation
				(
					( isMissing("workDescription") && newDoc().timetype in ["RB", "OTO", "OR", "OW"]) || !(newDoc().timetype in ["RB", "OTO", "OW", "OR"])
				) &&

				// if timetype is "RB", only hours is provided and is a positive number
				(
					(isPositiveMultipleOfPointFive("hours") && newDoc().timetype == "RB" && newDoc().keys().hasOnly(["date", "timetype", "timetypeName", "uid", "weekEnding", "hours"])) ||
					newDoc().timetype != "RB"
				) &&

				// when provided, jobHours require an existing job
				(
					( isPositiveMultipleOfPointFiveUnderEighteen("jobHours") && isInCollection("job", "Jobs") ) ||
					isMissing("jobHours")
				) &&

				// when provided, mealsHours requires either jobHours or hours to be present
				(
					(
						isPositiveMultipleOfPointFiveUnderEighteen("mealsHours") &&
						(isPositiveMultipleOfPointFiveUnderEighteen("hours") || isPositiveMultipleOfPointFiveUnderEighteen("jobHours"))
					) ||
					isMissing("mealsHours")
				) &&

				// The total of jobHours and mealsHours and hours doesn't exceed 18 unless timetype is RB
				(
					newDoc().timetype != "RB" &&
					valueOrZero("mealsHours") + valueOrZero("jobHours") + valueOrZero("hours") <= 18 ||
					newDoc().timetype == "RB"
				) &&
				// when provided workrecord requires an existing job
				(
					( hasValidWorkrecord() && isInCollection("job", "Jobs") ) ||
					isMissing("workrecord")
				);
		}
	*/

	// return nil
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
	timeTypeCode, cleanErr := cleanTimeEntry(app, record)
	if cleanErr != nil {
		return apis.NewBadRequestError("Error cleaning time_entry record", cleanErr)
	}

	// perform the validation for the time_entry record. In this step we also
	// write the uid property to the record so that we can validate it against the
	if validationErr := validateTimeEntry(record, timeTypeCode); validationErr != nil {
		return apis.NewBadRequestError("Validation error", validationErr)
	}

	return nil
}

func AddHooks(app *pocketbase.PocketBase) {
	// hooks for time_entries model
	app.OnRecordBeforeCreateRequest("time_entries").Add(func(e *core.RecordCreateEvent) error {
		if err := ProcessTimeEntry(app, e.Record, e.HttpContext); err != nil {
			// return the error to the client
			return err
		}
		return nil
	})
	app.OnRecordBeforeUpdateRequest("time_entries").Add(func(e *core.RecordUpdateEvent) error {
		if err := ProcessTimeEntry(app, e.Record, e.HttpContext); err != nil {
			// return the error to the client
			return err
		}
		return nil
	})
}
