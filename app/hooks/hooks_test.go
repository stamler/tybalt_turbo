package hooks

import (
	"fmt"
	"strings"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/models"
)

// We need to instantiate a Collection object to be part of the Record object
// so everything works as expected
var collection = &models.Collection{
	Name:   "time_entries",
	Type:   "base",
	System: false,
}

func buildRecordFromMap(m map[string]any) *models.Record {
	record := models.NewRecord(collection)
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
	// Test cases
	tests := map[string]struct {
		timeTypeCode string
		valid        bool
		field        string
		record       *models.Record
	}{
		// Regular Time (R)
		"valid no job":              {timeTypeCode: "R", valid: true, record: buildRecordFromMap(map[string]any{"date": "2024-01-22", "division": "DE", "meals_hours": 1.0, "hours": 1.5, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"valid with job":            {timeTypeCode: "R", valid: true, record: buildRecordFromMap(map[string]any{"date": "2024-01-22", "division": "DE", "meals_hours": 0.5, "hours": 6.5, "job": "JOBID1234567890", "description": "This is more than 5 chars", "work_record": "F23-137", "payout_request_amount": 0.0})},
		"valid leap year":           {timeTypeCode: "R", valid: true, record: buildRecordFromMap(map[string]any{"date": "2024-02-29", "division": "DE", "meals_hours": 1.0, "hours": 1.5, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"invalid date":              {timeTypeCode: "R", valid: false, field: "date", record: buildRecordFromMap(map[string]any{"date": "2024-01-32", "division": "DE", "meals_hours": 1.0, "hours": 1.5, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"invalid date string":       {timeTypeCode: "R", valid: false, field: "date", record: buildRecordFromMap(map[string]any{"date": "20240122", "division": "DE", "meals_hours": 1.0, "hours": 1.5, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"invalid leap year":         {timeTypeCode: "R", valid: false, field: "date", record: buildRecordFromMap(map[string]any{"date": "2023-02-29", "division": "DE", "meals_hours": 1.0, "hours": 1.5, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"hours not multiple of 0.5": {timeTypeCode: "R", valid: false, field: "hours", record: buildRecordFromMap(map[string]any{"date": "2024-01-22", "division": "DE", "meals_hours": 1.0, "hours": 1.4, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"extraneous payout amount":  {timeTypeCode: "R", valid: false, field: "payout_request_amount", record: buildRecordFromMap(map[string]any{"date": "2024-01-22", "division": "DE", "meals_hours": 1.0, "hours": 1.5, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 500.0})},
		"work record without job":   {timeTypeCode: "R", valid: false, field: "work_record", record: buildRecordFromMap(map[string]any{"date": "2024-01-22", "division": "DE", "meals_hours": 1.0, "hours": 1.5, "job": "", "description": "This is more than 5 chars", "work_record": "F23-137", "payout_request_amount": 0.0})},
		"missing division":          {timeTypeCode: "R", valid: false, field: "division", record: buildRecordFromMap(map[string]any{"date": "2024-01-22", "division": "", "meals_hours": 1.0, "hours": 1.5, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"too many hours":            {timeTypeCode: "R", valid: false, field: "global", record: buildRecordFromMap(map[string]any{"date": "2024-01-22", "division": "DE", "meals_hours": 1.0, "hours": 18.0, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"no hours":                  {timeTypeCode: "R", valid: false, field: "hours", record: buildRecordFromMap(map[string]any{"date": "2024-01-22", "division": "DE", "meals_hours": 0.0, "hours": 0.0, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"bad work_record value":     {timeTypeCode: "R", valid: false, field: "work_record", record: buildRecordFromMap(map[string]any{"date": "2024-01-22", "division": "DE", "meals_hours": 0.5, "hours": 5.0, "job": "JOBID1234567890", "description": "This is more than 5 chars", "work_record": "F23-137-", "payout_request_amount": 0.0})},
		"no job, missing hours":     {timeTypeCode: "R", valid: false, field: "hours", record: buildRecordFromMap(map[string]any{"date": "2024-01-22", "division": "DE", "meals_hours": 0.0, "hours": 0.0, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"description too short":     {timeTypeCode: "R", valid: false, field: "description", record: buildRecordFromMap(map[string]any{"date": "2024-01-22", "division": "DE", "meals_hours": 0.5, "hours": 5.0, "job": "JOBID1234567890", "description": "tiny", "work_record": "", "payout_request_amount": 0})},
	}

	// Run tests
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := validateTimeEntry(tt.record, tt.timeTypeCode)
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
