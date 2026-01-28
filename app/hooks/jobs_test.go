package hooks

import (
	"testing"

	"tybalt/errs"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// TestGenerateTopLevelJobNumber_MixedFormats verifies that the job number generator
// correctly handles a mix of legacy 3-digit and current 4-digit job number formats.
//
// Test data in test_pb_data/data.db for year 24:
//   - 24-291, 24-321, 24-326, 24-334 (3-digit, seq 291-334)
//   - 24-0350 (4-digit, seq 350)
//   - 24-334-01 (sub-job with parent, should be excluded)
//   - 24-350-01 (orphaned sub-job, excluded by length filter)
//
// Expected: next number should be 24-0351 (max of 334 and 350 is 350, so next is 351)
func TestGenerateTopLevelJobNumber_MixedFormats(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	// Test with year 24 to use our test data
	number, err := generateTopLevelJobNumberForYear(app, jobTypeProject, 24)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "24-0351"
	if number != expected {
		t.Errorf("expected %q, got %q", expected, number)
	}
}

// TestGenerateTopLevelJobNumber_ExcludesSubjobs verifies that sub-jobs
// are excluded from the sequence calculation by the LIKE pattern.
//
// Test data includes 24-334-01 (9 chars) which won't match
// either "24-___" (6 chars) or "24-____" (7 chars).
func TestGenerateTopLevelJobNumber_ExcludesSubjobs(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	// Sub-jobs like 24-334-01 are excluded because the LIKE patterns
	// only match exactly 6 or 7 character job numbers.
	// The expected result is 24-0351 (based on max of base jobs: 350)
	number, err := generateTopLevelJobNumberForYear(app, jobTypeProject, 24)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "24-0351"
	if number != expected {
		t.Errorf("expected %q, got %q (sub-job may not be excluded correctly)", expected, number)
	}
}

// TestGenerateTopLevelJobNumber_Proposals verifies that proposal number generation
// works correctly with the P prefix.
//
// Test data in test_pb_data/data.db:
//   - P24-487 (seq 487)
//   - P24-999 (seq 999)
//
// Expected: next proposal number should be P24-1000
func TestGenerateTopLevelJobNumber_Proposals(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	number, err := generateTopLevelJobNumberForYear(app, jobTypeProposal, 24)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "P24-1000"
	if number != expected {
		t.Errorf("expected %q, got %q", expected, number)
	}
}

// TestGenerateTopLevelJobNumber_EmptyYear verifies behavior when no jobs exist
// for the given year.
//
// Using year 99 which has no test data.
// Expected: first job number should be XX-0001
func TestGenerateTopLevelJobNumber_EmptyYear(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	// Test with year 99 which has no existing jobs
	number, err := generateTopLevelJobNumberForYear(app, jobTypeProject, 99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "99-0001"
	if number != expected {
		t.Errorf("expected %q, got %q", expected, number)
	}
}

// TestGenerateTopLevelJobNumber_EmptyYearProposal verifies behavior when no proposals
// exist for the given year.
func TestGenerateTopLevelJobNumber_EmptyYearProposal(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	// Test with year 99 which has no existing proposals
	number, err := generateTopLevelJobNumberForYear(app, jobTypeProposal, 99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "P99-0001"
	if number != expected {
		t.Errorf("expected %q, got %q", expected, number)
	}
}

// TestValidateRateSheetIsActive_InactiveRateSheet verifies that setting a job's
// rate_sheet to an inactive rate sheet returns an error.
//
// Test data in test_pb_data/data.db:
//   - rate_sheets: "c41ofep525bcacj" (2025 Standard Rates)
func TestValidateRateSheetIsActive_InactiveRateSheet(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	// Ensure the test rate sheet is inactive
	rateSheet, err := app.FindRecordById("rate_sheets", "c41ofep525bcacj")
	if err != nil {
		t.Fatalf("failed to find rate sheet: %v", err)
	}
	rateSheet.Set("active", false)
	if err := app.Save(rateSheet); err != nil {
		t.Fatalf("failed to deactivate rate sheet: %v", err)
	}

	jobsCollection, err := app.FindCollectionByNameOrId("jobs")
	if err != nil {
		t.Fatalf("failed to get jobs collection: %v", err)
	}

	// Create a new job record and set rate_sheet to the inactive rate sheet
	record := core.NewRecord(jobsCollection)
	record.Set("rate_sheet", "c41ofep525bcacj")

	err = validateRateSheetIsActive(app, record)
	if err == nil {
		t.Error("expected error when setting inactive rate sheet, got nil")
	}
}

// TestValidateRateSheetIsActive_RequiredForNewProject verifies that creating
// a new project without a rate_sheet returns an error.
func TestValidateRateSheetIsActive_RequiredForNewProject(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	jobsCollection, err := app.FindCollectionByNameOrId("jobs")
	if err != nil {
		t.Fatalf("failed to get jobs collection: %v", err)
	}

	// Create a new project record (has project_award_date, no proposal dates)
	// without a rate_sheet - should fail
	record := core.NewRecord(jobsCollection)
	record.Set("project_award_date", "2025-01-15")
	// rate_sheet intentionally left empty

	err = validateRateSheetIsActive(app, record)
	if err == nil {
		t.Error("expected error when creating project without rate_sheet, got nil")
	}

	// Verify it's the right error
	hookErr, ok := err.(*errs.HookError)
	if !ok {
		t.Fatalf("expected HookError, got %T", err)
	}
	if _, exists := hookErr.Data["rate_sheet"]; !exists {
		t.Error("expected error on rate_sheet field")
	}
}

// TestValidateRateSheetIsActive_NotRequiredForExistingProject verifies that
// updating an existing project without changing rate_sheet is allowed,
// even if rate_sheet is empty.
//
// Test data in test_pb_data/data.db:
//   - jobs: "u09fwwcg07y03m7" (24-291, project with no rate_sheet)
func TestValidateRateSheetIsActive_NotRequiredForExistingProject(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	// Use an existing project job from the test database
	existingRecord, err := app.FindRecordById("jobs", "u09fwwcg07y03m7")
	if err != nil {
		t.Fatalf("failed to fetch existing project: %v", err)
	}

	// Verify it's a project without a rate_sheet
	if existingRecord.GetString("rate_sheet") != "" {
		t.Fatalf("test expects project without rate_sheet, but got: %q", existingRecord.GetString("rate_sheet"))
	}

	// Update some other field, leave rate_sheet empty
	existingRecord.Set("description", "Updated description for test")

	// Should not fail because rate_sheet isn't being changed
	err = validateRateSheetIsActive(app, existingRecord)
	if err != nil {
		t.Errorf("expected no error when updating existing project without rate_sheet, got: %v", err)
	}
}

// TestValidateRateSheetIsActive_SkippedForProposal verifies that
// rate_sheet validation is skipped for proposals.
func TestValidateRateSheetIsActive_SkippedForProposal(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	jobsCollection, err := app.FindCollectionByNameOrId("jobs")
	if err != nil {
		t.Fatalf("failed to get jobs collection: %v", err)
	}

	// Create a new proposal record (has proposal dates, no project_award_date)
	// without a rate_sheet - should NOT fail
	record := core.NewRecord(jobsCollection)
	record.Set("proposal_opening_date", "2025-01-15")
	record.Set("proposal_submission_due_date", "2025-01-20")
	// rate_sheet intentionally left empty

	err = validateRateSheetIsActive(app, record)
	if err != nil {
		t.Errorf("expected no error for proposal without rate_sheet, got: %v", err)
	}
}

// TestCleanJob_ClearsRateSheetForProposal verifies that cleanJob
// clears the rate_sheet field for proposals.
func TestCleanJob_ClearsRateSheetForProposal(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	jobsCollection, err := app.FindCollectionByNameOrId("jobs")
	if err != nil {
		t.Fatalf("failed to get jobs collection: %v", err)
	}

	// Create a proposal record with a rate_sheet set
	record := core.NewRecord(jobsCollection)
	record.Set("proposal_opening_date", "2025-01-15")
	record.Set("proposal_submission_due_date", "2025-01-20")
	record.Set("rate_sheet", "c41ofep525bcacj") // Set a rate_sheet

	// Run cleanJob
	err = cleanJob(app, record)
	if err != nil {
		t.Fatalf("cleanJob returned error: %v", err)
	}

	// Verify rate_sheet was cleared
	if record.GetString("rate_sheet") != "" {
		t.Errorf("expected rate_sheet to be cleared for proposal, got: %q", record.GetString("rate_sheet"))
	}
}

// TestValidateRateSheetIsActive_ChangeToInactiveRejected verifies that
// changing an existing project's rate_sheet to an inactive one fails.
//
// Test data in test_pb_data/data.db:
//   - jobs: "u09fwwcg07y03m7" (24-291, project)
//   - rate_sheets: "c41ofep525bcacj" (active), "test_empty_sheet" (inactive)
func TestValidateRateSheetIsActive_ChangeToInactiveRejected(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	// Set up: give an existing project an active rate_sheet
	project, err := app.FindRecordById("jobs", "u09fwwcg07y03m7")
	if err != nil {
		t.Fatalf("failed to fetch project: %v", err)
	}
	project.Set("rate_sheet", "c41ofep525bcacj") // active rate sheet
	if err := app.Save(project); err != nil {
		t.Fatalf("failed to save project with rate_sheet: %v", err)
	}

	// Fetch again so Original() has the active rate_sheet
	project, err = app.FindRecordById("jobs", "u09fwwcg07y03m7")
	if err != nil {
		t.Fatalf("failed to re-fetch project: %v", err)
	}

	// Try to change to inactive rate_sheet
	project.Set("rate_sheet", "test_empty_sheet")

	err = validateRateSheetIsActive(app, project)
	if err == nil {
		t.Error("expected error when changing to inactive rate_sheet, got nil")
	}

	// Verify it's the right error
	hookErr, ok := err.(*errs.HookError)
	if !ok {
		t.Fatalf("expected HookError, got %T", err)
	}
	if _, exists := hookErr.Data["rate_sheet"]; !exists {
		t.Error("expected error on rate_sheet field")
	}
}
