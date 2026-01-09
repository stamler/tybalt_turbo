package hooks

import (
	"testing"

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
