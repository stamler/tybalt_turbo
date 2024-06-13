package hooks

import (
	"testing"

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

func TestValidateTimeEntry(t *testing.T) {
	// Test cases
	tests := []struct {
		name         string
		timeTypeCode string
		record       *models.Record
		valid        bool
	}{
		{
			name:         "Valid time entry without job",
			timeTypeCode: "R",
			record: buildRecordFromMap(map[string]any{
				"division":              "DE",
				"meals_hours":           0,
				"hours":                 1.5,
				"job_hours":             0,
				"job":                   "",
				"description":           "This is more than 5 chars",
				"work_record":           "",
				"payout_request_amount": 0,
			}),
			valid: true,
		},
		{
			name:         "invalid time entry without job and no hours",
			timeTypeCode: "R",
			record: buildRecordFromMap(map[string]any{
				"division":              "DE",
				"meals_hours":           0,
				"hours":                 0,
				"job_hours":             0,
				"job":                   "",
				"description":           "This is more than 5 chars",
				"work_record":           "",
				"payout_request_amount": 0,
			}),
			valid: false,
		},
		{
			name:         "Valid time entry with job",
			timeTypeCode: "R",
			record: buildRecordFromMap(map[string]any{
				"division":              "DE",
				"meals_hours":           0,
				"hours":                 0,
				"job_hours":             2.5,
				"job":                   "JOBID1234567890",
				"description":           "This is more than 5 chars",
				"work_record":           "",
				"payout_request_amount": 0,
			}),
			valid: true,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateTimeEntry(tt.record, tt.timeTypeCode)
			if got != nil && tt.valid {
				t.Errorf("failed validation (%v) but expected valid", got)
			}
			if got == nil && !tt.valid {
				t.Errorf("passed validation but expected invalid")
			}
		})
	}
}
