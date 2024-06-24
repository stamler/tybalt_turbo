package hooks

import (
	"encoding/json"
	"log"
	"regexp"
	"time"

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

func isPositiveMultipleOfPointFive() validation.RuleFunc {
	return func(value interface{}) error {
		s, _ := value.(float64)
		if s == 0 {
			return nil
		}
		if s < 0 {
			return validation.NewError("validation_negative_number", "must be a positive number")
		}
		// return error is s is not a multiple of 0.5
		if s/0.5 != float64(int(s/0.5)) {
			return validation.NewError("validation_not_multiple_of_point_five", "must be a multiple of 0.5")
		}
		return nil
	}
}

func isPositiveMultipleOfPointZeroOne() validation.RuleFunc {
	return func(value interface{}) error {
		s, _ := value.(float64)
		if s == 0 {
			return nil
		}
		if s < 0 {
			return validation.NewError("validation_negative_number", "must be a positive number")
		}
		// return error is s is not a multiple of 0.1
		if s/0.01 != float64(int(s/0.01)) {
			return validation.NewError("validation_not_multiple_of_point_zero_one", "must be a multiple of 0.01")
		}
		return nil
	}
}

func isValidDate(value interface{}) error {
	s, _ := value.(string)
	if _, err := time.Parse(time.DateOnly, s); err != nil {
		return validation.NewError("validation_invalid_date", s+" is not a valid date")
	}
	return nil
}

// generate the week ending date from the date property. The week ending date is
// the Saturday immediately following the date property. If the argument is
// already a Saturday, it is returned unchanged. The date property is a string
// in the format "YYYY-MM-DD".
func generateWeekEnding(date string) (string, error) {
	// parse the date string
	t, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return "", err
	}

	// add days to the date until it is a Saturday
	for t.Weekday() != time.Saturday {
		t = t.AddDate(0, 0, 1)
	}

	// return the date as a string
	return t.Format(time.DateOnly), nil
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
		"hours":                 validation.Validate(timeEntryRecord.Get("hours"), validation.By(isPositiveMultipleOfPointFive())),
		"date":                  validation.Validate(timeEntryRecord.Get("date"), validation.By(isValidDate)),
		"global":                validation.Validate(totalHours, validation.Max(18.0).Error("Total hours must not exceed 18")),
		"meals_hours":           validation.Validate(timeEntryRecord.Get("meals_hours"), validation.Max(3.0).Error("Meals Hours must not exceed 3")),
		"description":           validation.Validate(timeEntryRecord.Get("description"), validation.Length(5, 0).Error("must be at least 5 characters")),
		"work_record":           validation.Validate(timeEntryRecord.Get("work_record"), validation.When(jobIsPresent, validation.Match(regexp.MustCompile("^[FKQ][0-9]{2}-[0-9]{3,4}(-[0-9]+)?$")).Error("must be in the correct format")).Else(validation.In("").Error("Work Record must be empty when job is not provided"))),
		"payout_request_amount": validation.Validate(timeEntryRecord.Get("payout_request_amount"), validation.Min(0.0).Exclusive().Error("Amount must be greater than 0"), validation.By(isPositiveMultipleOfPointZeroOne())),
	}.Filter()

	return otherValidationsErrors

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
	requiredFields, cleanErr := cleanTimeEntry(app, record)
	if cleanErr != nil {
		return apis.NewBadRequestError("Error cleaning time_entry record", cleanErr)
	}

	// write the week_ending property to the record. This is derived exclusively
	// from the date property.
	weekEnding, wkEndErr := generateWeekEnding(record.GetString("date"))
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
