package hooks

import (
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

func extractArbitraryKey(err error) string {
	errMap := err.(validation.Errors)
	var arbitraryKey string
	for k := range errMap {
		arbitraryKey = k
		break
	}
	return arbitraryKey
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
		"valid no job":                  {timeTypeCode: "R", valid: true, record: buildRecordFromMap(map[string]any{"division": "DE", "meals_hours": 1.0, "hours": 1.5, "job_hours": 0.0, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"valid with job":                {timeTypeCode: "R", valid: true, record: buildRecordFromMap(map[string]any{"division": "DE", "meals_hours": 0.5, "hours": 0.0, "job_hours": 2.5, "job": "JOBID1234567890", "description": "This is more than 5 chars", "work_record": "F23-137", "payout_request_amount": 0.0})},
		"job hours not multiple of 0.5": {timeTypeCode: "R", valid: false, field: "job_hours", record: buildRecordFromMap(map[string]any{"division": "DE", "meals_hours": 0.5, "hours": 0.0, "job_hours": 2.4, "job": "JOBID1234567890", "description": "This is more than 5 chars", "work_record": "F23-137", "payout_request_amount": 0.0})},
		"hours not multiple of 0.5":     {timeTypeCode: "R", valid: false, field: "hours", record: buildRecordFromMap(map[string]any{"division": "DE", "meals_hours": 1.0, "hours": 1.4, "job_hours": 0.0, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"extraneous payout amount":      {timeTypeCode: "R", valid: false, field: "payout_request_amount", record: buildRecordFromMap(map[string]any{"division": "DE", "meals_hours": 1.0, "hours": 1.5, "job_hours": 0.0, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 500.0})},
		"work record without job":       {timeTypeCode: "R", valid: false, field: "work_record", record: buildRecordFromMap(map[string]any{"division": "DE", "meals_hours": 1.0, "hours": 1.5, "job_hours": 0.0, "job": "", "description": "This is more than 5 chars", "work_record": "F23-137", "payout_request_amount": 0.0})},
		"missing division":              {timeTypeCode: "R", valid: false, field: "division", record: buildRecordFromMap(map[string]any{"division": "", "meals_hours": 1.0, "hours": 1.5, "job_hours": 0.0, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"too many hours":                {timeTypeCode: "R", valid: false, field: "global", record: buildRecordFromMap(map[string]any{"division": "DE", "meals_hours": 1.0, "hours": 18.0, "job_hours": 0.0, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"no hours":                      {timeTypeCode: "R", valid: false, field: "hours", record: buildRecordFromMap(map[string]any{"division": "DE", "meals_hours": 0.0, "hours": 0.0, "job_hours": 0.0, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"both types of hours no job":    {timeTypeCode: "R", valid: false, field: "job_hours", record: buildRecordFromMap(map[string]any{"division": "DE", "meals_hours": 1.0, "hours": 3.0, "job_hours": 5.0, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"both types of hours with job":  {timeTypeCode: "R", valid: false, field: "hours", record: buildRecordFromMap(map[string]any{"division": "DE", "meals_hours": 1.0, "hours": 3.0, "job_hours": 5.0, "job": "JOBID1234567890", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"bad work_record value":         {timeTypeCode: "R", valid: false, field: "work_record", record: buildRecordFromMap(map[string]any{"division": "DE", "meals_hours": 0.5, "hours": 0.0, "job_hours": 2.5, "job": "JOBID1234567890", "description": "This is more than 5 chars", "work_record": "F23-137-", "payout_request_amount": 0.0})},
		"no job, missing hours":         {timeTypeCode: "R", valid: false, field: "hours", record: buildRecordFromMap(map[string]any{"division": "DE", "meals_hours": 0.0, "hours": 0.0, "job_hours": 0.0, "job": "", "description": "This is more than 5 chars", "work_record": "", "payout_request_amount": 0.0})},
		"description too short":         {timeTypeCode: "R", valid: false, field: "description", record: buildRecordFromMap(map[string]any{"division": "DE", "meals_hours": 0.5, "hours": 0, "job_hours": 2.5, "job": "JOBID1234567890", "description": "tiny", "work_record": "", "payout_request_amount": 0})},
	}

	// Run tests
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := validateTimeEntry(tt.record, tt.timeTypeCode)
			if got != nil {
				// we extract an arbitrary key from the error map. Key order doesn't
				// exist in Go maps but we expect only one error at a time so arbitrary
				// works here.
				field := extractArbitraryKey(got)
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
