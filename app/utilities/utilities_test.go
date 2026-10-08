package utilities

import (
	"testing"
	"time"
	"tybalt/internal/testseed"
)

func TestIsMultipleOfPointFive(t *testing.T) {
	tests := []struct {
		name    string
		value   float64
		wantErr bool
	}{
		{name: "negative half hour", value: -0.5},
		{name: "negative boundary", value: -18},
		{name: "zero", value: 0},
		{name: "positive half hour", value: 0.5},
		{name: "positive boundary", value: 18},
		{name: "negative quarter hour", value: -0.25, wantErr: true},
		{name: "positive quarter hour", value: 0.25, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsMultipleOfPointFive()(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("IsMultipleOfPointFive() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsPositiveMultipleOfPointFiveRejectsNegativeValues(t *testing.T) {
	if err := IsPositiveMultipleOfPointFive()(-0.5); err == nil {
		t.Fatal("expected IsPositiveMultipleOfPointFive to reject a negative value")
	}
}

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

func TestRecurringOccurrences(t *testing.T) {
	tests := []struct {
		name      string
		frequency string
		start     string
		end       string
		want      int
	}{
		// End date is exclusive: a one-year term has twelve monthly payments.
		{name: "monthly one-year term", frequency: "Monthly", start: "2026-02-01", end: "2027-02-01", want: 12},
		{name: "monthly payment on end date excluded", frequency: "Monthly", start: "2026-05-04", end: "2026-09-04", want: 4},
		{name: "monthly payment day before end date", frequency: "Monthly", start: "2026-05-04", end: "2026-09-05", want: 5},
		{name: "monthly month end to longer month", frequency: "Monthly", start: "2026-04-30", end: "2026-05-31", want: 2},
		{name: "monthly through calendar year", frequency: "Monthly", start: "2025-01-01", end: "2025-12-31", want: 12},
		{name: "monthly single payment", frequency: "Monthly", start: "2026-04-30", end: "2026-05-30", want: 1},
		// The 31st falls on the last day of shorter months.
		{name: "monthly 31st clamps to February", frequency: "Monthly", start: "2026-01-31", end: "2026-03-01", want: 2},
		{name: "monthly 31st February payment on end date", frequency: "Monthly", start: "2026-01-31", end: "2026-02-28", want: 1},
		{name: "monthly 31st leap year February", frequency: "Monthly", start: "2028-01-31", end: "2028-03-01", want: 2},
		{name: "monthly 31st leap year February payment on end date", frequency: "Monthly", start: "2028-01-31", end: "2028-02-29", want: 1},
		// Payments stay on the 31st after a short month rather than drifting.
		{name: "monthly 31st does not drift after February", frequency: "Monthly", start: "2026-01-31", end: "2026-03-31", want: 2},
		{name: "monthly 31st resumes on 31st", frequency: "Monthly", start: "2026-01-31", end: "2026-04-01", want: 3},
		{name: "monthly across year end", frequency: "Monthly", start: "2026-11-15", end: "2027-02-16", want: 4},
		{name: "weekly payment on end date excluded", frequency: "Weekly", start: "2025-02-17", end: "2025-03-03", want: 2},
		{name: "weekly day after boundary", frequency: "Weekly", start: "2025-02-17", end: "2025-03-04", want: 3},
		{name: "weekly one-year term", frequency: "Weekly", start: "2026-06-03", end: "2027-06-03", want: 53},
		{name: "weekly across leap day", frequency: "Weekly", start: "2028-02-22", end: "2028-03-08", want: 3},
		{name: "biweekly payment on end date excluded", frequency: "Biweekly", start: "2025-02-17", end: "2025-03-17", want: 2},
		{name: "biweekly", frequency: "Biweekly", start: "2025-02-17", end: "2025-03-31", want: 3},
		{name: "end date equals start date", frequency: "Monthly", start: "2026-04-30", end: "2026-04-30", want: 0},
		{name: "end date before start date", frequency: "Weekly", start: "2026-04-30", end: "2026-04-01", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, _ := time.Parse(time.DateOnly, tt.start)
			end, _ := time.Parse(time.DateOnly, tt.end)
			got, ok := RecurringOccurrences(start, end, tt.frequency)
			if !ok || got != tt.want {
				t.Fatalf("RecurringOccurrences(%s, %s, %s) = %d, %v; want %d", tt.start, tt.end, tt.frequency, got, ok, tt.want)
			}
		})
	}

	if _, ok := RecurringOccurrences(time.Now(), time.Now().AddDate(1, 0, 0), "Daily"); ok {
		t.Fatal("expected unknown frequency to be rejected")
	}
}
