package hooks

import (
	"fmt"
	"strings"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// We need to instantiate a Collection object to be part of the Record object
// so everything works as expected
var timeEntriesCollection = core.NewBaseCollection("time_entries")

func buildRecordFromMap(collection *core.Collection, m map[string]any) *core.Record {
	record := core.NewRecord(collection)
	record.Load(m)
	return record
}

func extractArbitraryKey(err error) (string, error) {
	errMap := err.(validation.Errors)
	keys := make([]string, len(errMap))
	i := 0
	for k := range errMap {
		keys[i] = k
		i++
	}

	// if the keys array length is not 1, return an error
	if len(keys) != 1 {
		return "", fmt.Errorf("multiple error keys: %v", strings.Join(keys, ", "))
	}

	// return the first key
	return keys[0], nil
}

func TestValidateTimeEntry(t *testing.T) {
	requiredFieldsByTimeTypeCode := map[string][]string{
		"OB":  {"date", "time_type", "hours"},
		"OH":  {"date", "time_type", "hours"},
		"OP":  {"date", "time_type", "hours"},
		"OR":  {"date", "time_type"},
		"OS":  {"date", "time_type", "hours"},
		"OTO": {"date", "time_type", "payout_request_amount"},
		"OV":  {"date", "time_type", "hours"},
		"OW":  {"date", "time_type"},
		"R":   {"date", "time_type", "division", "hours", "description"},
		"RB":  {"date", "time_type", "hours"},
		"RT":  {"date", "time_type", "division", "hours", "description"},
	}

	// Test cases
	cases := map[string]struct {
		timeTypeCode string
		valid        bool
		field        string
		record       *core.Record
	}{
		// Regular Time (R)
		// NB: the time_type value is arbitrary and is required but otherwise not
		// validated in this test. It is an id of a relation and is validated by the
		// pocketbase relation itself.
		"valid no job":              {timeTypeCode: "R", valid: true, record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "division": "DE", "meals_hours": 1.0, "hours": 1.5, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"valid with job":            {timeTypeCode: "R", valid: true, record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "division": "DE", "meals_hours": 0.5, "hours": 6.5, "job": "mg0sp9iyjzo4zw9", "description": "This is more than 5 chars", "work_record": "F23-137", "payout_request_amount": 0.0})},
		"valid leap year":           {timeTypeCode: "R", valid: true, record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-02-29", "division": "DE", "meals_hours": 1.0, "hours": 1.5, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"invalid date":              {timeTypeCode: "R", valid: false, field: "date", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-32", "division": "DE", "meals_hours": 1.0, "hours": 1.5, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"missing date":              {timeTypeCode: "R", valid: false, field: "date", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "division": "DE", "meals_hours": 1.0, "hours": 1.5, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"invalid date string":       {timeTypeCode: "R", valid: false, field: "date", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "20240122", "division": "DE", "meals_hours": 1.0, "hours": 1.5, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"invalid leap year":         {timeTypeCode: "R", valid: false, field: "date", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2023-02-29", "division": "DE", "meals_hours": 1.0, "hours": 1.5, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"hours not multiple of 0.5": {timeTypeCode: "R", valid: false, field: "hours", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "division": "DE", "meals_hours": 1.0, "hours": 1.4, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"work record without job":   {timeTypeCode: "R", valid: false, field: "work_record", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "division": "DE", "meals_hours": 1.0, "hours": 1.5, "job": "", "description": "This is more than 5 chars", "work_record": "F23-137", "payout_request_amount": 0.0})},
		"missing division":          {timeTypeCode: "R", valid: false, field: "division", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "division": "", "meals_hours": 1.0, "hours": 1.5, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"too many hours":            {timeTypeCode: "R", valid: false, field: "global", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "division": "DE", "meals_hours": 1.0, "hours": 18.0, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"no hours":                  {timeTypeCode: "R", valid: false, field: "hours", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "division": "DE", "meals_hours": 0.0, "hours": 0.0, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"bad work_record value":     {timeTypeCode: "R", valid: false, field: "work_record", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "division": "DE", "meals_hours": 0.5, "hours": 5.0, "job": "mg0sp9iyjzo4zw9", "description": "This is more than 5 chars", "work_record": "F23-137-", "payout_request_amount": 0.0})},
		"no job, missing hours":     {timeTypeCode: "R", valid: false, field: "hours", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "division": "DE", "meals_hours": 0.0, "hours": 0.0, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"description too short":     {timeTypeCode: "R", valid: false, field: "description", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "division": "DE", "meals_hours": 0.5, "hours": 5.0, "job": "mg0sp9iyjzo4zw9", "description": "tiny", "work_record": "", "payout_request_amount": 0})},
		"closed job":                {timeTypeCode: "R", valid: false, field: "job", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "division": "DE", "hours": 1.0, "job": "zke3cs3yipplwtu", "description": "This is more than 5 chars"})},

		// Vacation, Sick, Personal, Holiday, Bereavement
		"valid vacation":                {timeTypeCode: "OV", valid: true, record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "hours": 8, "description": "This is more than 5 chars"})},
		"valid sick":                    {timeTypeCode: "OS", valid: true, record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "hours": 8, "description": "This is more than 5 chars"})},
		"valid vacation no description": {timeTypeCode: "OV", valid: true, record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "hours": 8})},
		"vacation missing hours":        {timeTypeCode: "OV", valid: false, field: "hours", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "description": "This is more than 5 chars"})},
		"vac description too short":     {timeTypeCode: "OV", valid: false, field: "description", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "hours": 8, "description": "tiny"})},
		"ppto description too short":    {timeTypeCode: "OP", valid: false, field: "description", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "hours": 8, "description": "tiny"})},

		// Bank Time, Overtime Off Request
		"valid bank time":            {timeTypeCode: "RB", valid: true, record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "hours": 8})},
		"valid overtime off request": {timeTypeCode: "OTO", valid: true, record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "payout_request_amount": 100.0})},
		"bank time missing date":     {timeTypeCode: "RB", valid: false, field: "date", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "hours": 8})},
		"OTO no amount":              {timeTypeCode: "OTO", valid: false, field: "payout_request_amount", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22"})},
		"negative OTO amount":        {timeTypeCode: "OTO", valid: false, field: "payout_request_amount", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "payout_request_amount": -100.0})},
		"fractional OTO amount":      {timeTypeCode: "OTO", valid: false, field: "payout_request_amount", record: buildRecordFromMap(timeEntriesCollection, map[string]any{"time_type": "dummy", "date": "2024-01-22", "payout_request_amount": 132.001})},
	}

	// Initialize a PocketBase TestApp so validateTimeEntry can perform lookups
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatalf("failed to init test app: %v", err)
	}
	defer app.Cleanup()

	// Run tests
	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			got := validateTimeEntry(app, tt.record, requiredFieldsByTimeTypeCode[tt.timeTypeCode])
			if got != nil {
				// we extract an arbitrary key from the error map. Key order doesn't
				// exist in Go maps but we expect only one error at a time so arbitrary
				// works here.
				field, keysError := extractArbitraryKey(got)
				if keysError != nil {
					t.Errorf("error extracting key from error map: %v", keysError)
				}
				if tt.valid {
					t.Errorf("failed validation (%v) but expected valid", got)
				}
				if !tt.valid {
					// If the field doesn't match expected field, the test has failed
					if field != string(tt.field) {
						t.Errorf("expected field: %s, got: %s", string(tt.field), field)
					}
				}
			}
			if got == nil && !tt.valid {
				t.Errorf("passed validation but expected invalid")
			}
		})
	}
}
