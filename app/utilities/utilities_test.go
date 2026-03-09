package utilities

import (
	"testing"
	"tybalt/internal/testseed"
)

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
