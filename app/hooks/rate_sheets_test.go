package hooks

import (
	"testing"
	"tybalt/internal/testseed"
)

// TestValidateRateSheetComplete_NoEntries tests that a rate sheet without any entries
// returns all roles as missing.
//
// Test data in test_pb_data/data.db:
//   - rate_sheets: "test_empty_sheet" (Empty Test Rate Sheet, active=false)
//   - rate_roles: 33 roles
//   - rate_sheet_entries: none for test_empty_sheet
func TestValidateRateSheetComplete_NoEntries(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	rateSheetId := "test_empty_sheet"

	missingRoles, err := ValidateRateSheetComplete(app, rateSheetId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 33 missing roles (all of them)
	if len(missingRoles) != 33 {
		t.Errorf("expected 33 missing roles, got %d", len(missingRoles))
	}
}

// TestValidateRateSheetComplete_AllEntries tests that a rate sheet with all entries
// returns no missing roles.
func TestValidateRateSheetComplete_AllEntries(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	rateSheetId := "rs_complete_all_001"

	// Now validate - should have no missing roles
	missingRoles, err := ValidateRateSheetComplete(app, rateSheetId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(missingRoles) != 0 {
		t.Errorf("expected 0 missing roles, got %d: %v", len(missingRoles), missingRoles)
	}
}

// TestValidateRateSheetComplete_PartialEntries tests that a rate sheet with some entries
// returns only the missing roles.
func TestValidateRateSheetComplete_PartialEntries(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	rateSheetId := "rs_partial_001"

	// Get all roles
	roles, err := app.FindRecordsByFilter("rate_roles", "1=1", "", 100, 0, nil)
	if err != nil {
		t.Fatalf("failed to get roles: %v", err)
	}

	if len(roles) < 5 {
		t.Fatalf("expected at least 5 roles, got %d", len(roles))
	}

	// Validate - should have 25 missing roles (30 - 5)
	missingRoles, err := ValidateRateSheetComplete(app, rateSheetId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := len(roles) - 5
	if len(missingRoles) != expected {
		t.Errorf("expected %d missing roles, got %d", expected, len(missingRoles))
	}
}

// TestCheckRevisionEffectiveDate_Revision0 tests that revision 0 is always allowed
// regardless of effective date.
func TestCheckRevisionEffectiveDate_Revision0(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	// Revision 0 should always pass validation, even with any date
	prevDate, err := CheckRevisionEffectiveDate(app, "Any Name", 0, "2020-01-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prevDate != "" {
		t.Errorf("expected empty string (valid), got %q", prevDate)
	}
}

// TestCheckRevisionEffectiveDate_ValidRevision tests that a revision with
// effective_date >= previous revision's date is allowed.
func TestCheckRevisionEffectiveDate_ValidRevision(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	sheetName := "Test Effective Date Validation"

	// Test revision 1 with same date - should be valid
	prevDate, err := CheckRevisionEffectiveDate(app, sheetName, 1, "2025-01-15")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prevDate != "" {
		t.Errorf("same date should be valid, got previous date %q", prevDate)
	}

	// Test revision 1 with later date - should be valid
	prevDate, err = CheckRevisionEffectiveDate(app, sheetName, 1, "2025-06-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prevDate != "" {
		t.Errorf("later date should be valid, got previous date %q", prevDate)
	}
}

// TestCheckRevisionEffectiveDate_InvalidRevision tests that a revision with
// effective_date < previous revision's date is rejected.
func TestCheckRevisionEffectiveDate_InvalidRevision(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	sheetName := "Test Invalid Effective Date"

	// Test revision 1 with earlier date - should be invalid
	prevDate, err := CheckRevisionEffectiveDate(app, sheetName, 1, "2025-01-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prevDate == "" {
		t.Error("earlier date should be invalid, but validation passed")
	}
	if prevDate != "2025-06-15" {
		t.Errorf("expected previous date to be 2025-06-15, got %q", prevDate)
	}
}

// TestCheckNewerRevisionExists_NoNewer tests that CheckNewerRevisionExists returns
// false when no newer revision exists.
func TestCheckNewerRevisionExists_NoNewer(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	sheetName := "Test Newer Revision Check"

	// Check for newer revision - should return false
	exists, err := CheckNewerRevisionExists(app, sheetName, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected no newer revision to exist")
	}
}

// TestCheckNewerRevisionExists_NewerExists tests that CheckNewerRevisionExists
// returns true when a newer revision exists.
func TestCheckNewerRevisionExists_NewerExists(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	sheetName := "Test Newer Revision Exists"

	// Check for newer revision from revision 0 - should return true
	exists, err := CheckNewerRevisionExists(app, sheetName, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected newer revision to exist")
	}

	// Check for newer revision from revision 1 - should return false
	exists, err = CheckNewerRevisionExists(app, sheetName, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected no newer revision from revision 1")
	}
}

// TestDeactivateOtherRevisions tests that DeactivateOtherRevisions deactivates
// all other revisions with the same name.
func TestDeactivateOtherRevisions(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	sheetName := "Test Deactivate Others"
	// Deactivate other revisions (keeping rev1)
	if err := DeactivateOtherRevisions(app, sheetName, "rs_deactivate_r1"); err != nil {
		t.Fatalf("failed to deactivate other revisions: %v", err)
	}

	// Refresh rev0 from database
	rev0, err := app.FindRecordById("rate_sheets", "rs_deactivate_r0")
	if err != nil {
		t.Fatalf("failed to fetch rev0: %v", err)
	}

	// rev0 should now be inactive
	if rev0.GetBool("active") {
		t.Error("expected revision 0 to be deactivated")
	}

	// rev1 should still be inactive (we didn't activate it, just deactivated others)
	rev1, err := app.FindRecordById("rate_sheets", "rs_deactivate_r1")
	if err != nil {
		t.Fatalf("failed to fetch rev1: %v", err)
	}
	if rev1.GetBool("active") {
		t.Error("expected revision 1 to still be inactive")
	}
}

// TestDeactivateOtherRevisions_DifferentName tests that DeactivateOtherRevisions
// does not affect rate sheets with different names.
func TestDeactivateOtherRevisions_DifferentName(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	// Deactivate other revisions of Sheet A (keeping sheetA1)
	if err := DeactivateOtherRevisions(app, "Sheet A", "rs_sheeta_r1"); err != nil {
		t.Fatalf("failed to deactivate other revisions: %v", err)
	}

	// Sheet A rev 0 should be deactivated
	sheetA, err := app.FindRecordById("rate_sheets", "rs_sheeta_r0")
	if err != nil {
		t.Fatalf("failed to fetch sheet A: %v", err)
	}
	if sheetA.GetBool("active") {
		t.Error("expected Sheet A rev 0 to be deactivated")
	}

	// Sheet B should still be active (different name)
	sheetB, err := app.FindRecordById("rate_sheets", "rs_sheetb_r0")
	if err != nil {
		t.Fatalf("failed to fetch sheet B: %v", err)
	}
	if !sheetB.GetBool("active") {
		t.Error("expected Sheet B to still be active")
	}
}
