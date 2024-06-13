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

// TODO: use NewRecordFromNullStringMap() or Load() instead of this
func buildRecordFromMap(m map[string]any) *models.Record {
	record := models.NewRecord(collection)
	record.Load(m)
	return record
}

func TestValidateTimeEntry(t *testing.T) {
	// Test cases
	tests := []struct {
		name   string
		record *models.Record
		want   error
	}{
		{
			name: "Valid time entry without job",
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
			want: nil,
		},
		{
			name: "Valid time entry with job",
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
			want: nil,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateTimeEntry(tt.record, "R"); got != tt.want {
				t.Errorf("validateTimeEntry() = %v, want %v", got, tt.want)
			}
		})
	}
}
