package routes

import (
	"encoding/json"
	"testing"
)

func TestTimeAmendmentExportMarshalJSON_WithJobUsesJobHoursAndLegacyCommitFields(t *testing.T) {
	payload, err := json.Marshal(timeAmendmentExport{
		Id:         "amendment-1",
		Job:        "25-001",
		Hours:      2.5,
		CommitTime: "2024-09-27 12:00:00.000Z",
		CommitUid:  "legacy_committer",
		CommitName: "Fakesy Manjor",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got, ok := decoded["jobHours"].(float64); !ok || got != 2.5 {
		t.Fatalf("jobHours = %#v, want 2.5", decoded["jobHours"])
	}
	if _, ok := decoded["hours"]; ok {
		t.Fatalf("hours unexpectedly present: %#v", decoded["hours"])
	}
	if got, ok := decoded["committed"].(bool); !ok || !got {
		t.Fatalf("committed = %#v, want true", decoded["committed"])
	}
	if got := decoded["commitTime"]; got != "2024-09-27 12:00:00.000Z" {
		t.Fatalf("commitTime = %#v, want %q", got, "2024-09-27 12:00:00.000Z")
	}
	if got := decoded["commitUid"]; got != "legacy_committer" {
		t.Fatalf("commitUid = %#v, want %q", got, "legacy_committer")
	}
	if got := decoded["commitName"]; got != "Fakesy Manjor" {
		t.Fatalf("commitName = %#v, want %q", got, "Fakesy Manjor")
	}
	if _, ok := decoded["committer"]; ok {
		t.Fatalf("committer unexpectedly present: %#v", decoded["committer"])
	}
	if _, ok := decoded["committerName"]; ok {
		t.Fatalf("committerName unexpectedly present: %#v", decoded["committerName"])
	}
}

func TestTimeAmendmentExportMarshalJSON_WithoutJobUsesHours(t *testing.T) {
	payload, err := json.Marshal(timeAmendmentExport{
		Id:    "amendment-2",
		Hours: 1.5,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got, ok := decoded["hours"].(float64); !ok || got != 1.5 {
		t.Fatalf("hours = %#v, want 1.5", decoded["hours"])
	}
	if _, ok := decoded["jobHours"]; ok {
		t.Fatalf("jobHours unexpectedly present: %#v", decoded["jobHours"])
	}
	if got, ok := decoded["committed"].(bool); !ok || !got {
		t.Fatalf("committed = %#v, want true", decoded["committed"])
	}
}
