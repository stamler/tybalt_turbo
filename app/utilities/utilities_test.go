package utilities

import (
	"testing"
	"tybalt/internal/testseed"
)

func TestGenerateCommittedPayPeriodEnding(t *testing.T) {
	tests := []struct {
		name                string
		expenseDate         string
		committedWeekEnding string
		want                string
	}{
		{
			name:                "week2 commit stays in same payroll",
			expenseDate:         "2026-03-19",
			committedWeekEnding: "2026-03-28",
			want:                "2026-03-28",
		},
		{
			name:                "week1 commit with old dated expense goes to previous payroll",
			expenseDate:         "2026-03-14",
			committedWeekEnding: "2026-03-21",
			want:                "2026-03-14",
		},
		{
			name:                "week1 commit with current dated expense goes to next payroll",
			expenseDate:         "2026-03-15",
			committedWeekEnding: "2026-03-21",
			want:                "2026-03-28",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateCommittedPayPeriodEnding(tt.expenseDate, tt.committedWeekEnding)
			if err != nil {
				t.Fatalf("GenerateCommittedPayPeriodEnding returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("GenerateCommittedPayPeriodEnding(%q, %q) = %q, want %q", tt.expenseDate, tt.committedWeekEnding, got, tt.want)
			}
		})
	}
}

func TestValidateTimeOffOpeningDate(t *testing.T) {
	tests := []struct {
		name        string
		openingDate string
		wantErr     bool
	}{
		{
			name:        "blank opening date allowed",
			openingDate: "",
			wantErr:     false,
		},
		{
			name:        "valid sunday after pay period ending",
			openingDate: "2026-01-04",
			wantErr:     false,
		},
		{
			name:        "weekday rejected",
			openingDate: "2026-01-01",
			wantErr:     true,
		},
		{
			name:        "sunday not after pay period ending rejected",
			openingDate: "2026-01-11",
			wantErr:     true,
		},
		{
			name:        "invalid calendar date rejected",
			openingDate: "2026-02-30",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimeOffOpeningDate(tt.openingDate)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateTimeOffOpeningDate(%q) error = %v, wantErr %v", tt.openingDate, err, tt.wantErr)
			}
		})
	}
}

func TestRecordHasMeaningfulChanges(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	client, err := app.FindRecordById("clients", "lb0fnenkeyitsny")
	if err != nil {
		t.Fatalf("failed to load client: %v", err)
	}

	if RecordHasMeaningfulChanges(client) {
		t.Fatal("expected unchanged record to have no meaningful changes")
	}

	originalName := client.GetString("name")
	client.Set("name", originalName+" (updated)")

	if !RecordHasMeaningfulChanges(client) {
		t.Fatal("expected changed record to report meaningful changes")
	}
}

func TestRecordHasMeaningfulChanges_OptionalImportedSkip(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	job, err := app.FindRecordById("jobs", "cjf0kt0defhq480")
	if err != nil {
		t.Fatalf("failed to load job: %v", err)
	}

	originalImported := job.GetBool("_imported")
	job.Set("_imported", !originalImported)

	if !RecordHasMeaningfulChanges(job) {
		t.Fatal("expected _imported-only change to be meaningful by default")
	}
	if RecordHasMeaningfulChanges(job, "_imported") {
		t.Fatal("expected _imported-only change to be ignored when _imported is skipped")
	}
}

func TestMarkReferencingJobsNotImported_InvalidColumn(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	if err := MarkReferencingJobsNotImported(app, "branch", "80875lm27v8wgi4"); err == nil {
		t.Fatal("expected invalid column to return an error")
	}
}

func TestMarkReferencingJobsNotImported_UpdatesJobs(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	const jobID = "job_marknotimp_001"
	const clientID = "lb0fnenkeyitsny"

	if err := MarkReferencingJobsNotImported(app, "client", clientID); err != nil {
		t.Fatalf("MarkReferencingJobsNotImported returned error: %v", err)
	}

	job, err := app.FindRecordById("jobs", jobID)
	if err != nil {
		t.Fatalf("failed to reload job: %v", err)
	}
	if job.GetBool("_imported") {
		t.Fatal("expected referenced job to be marked not imported")
	}
}
