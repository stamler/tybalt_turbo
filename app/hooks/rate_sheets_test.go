package hooks

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// TestValidateRateSheetComplete_NoEntries tests that a rate sheet without any entries
// returns all roles as missing.
//
// Test data in test_pb_data/data.db:
//   - rate_sheets: "test_empty_sheet" (Empty Test Rate Sheet, active=false)
//   - rate_roles: 33 roles
//   - rate_sheet_entries: none for test_empty_sheet
func TestValidateRateSheetComplete_NoEntries(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
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
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	rateSheetId := "test_empty_sheet"

	// Get all roles
	roles, err := app.FindRecordsByFilter("rate_roles", "1=1", "", 100, 0, nil)
	if err != nil {
		t.Fatalf("failed to get roles: %v", err)
	}

	// Create entries for all roles
	rateSheetEntriesCollection, err := app.FindCollectionByNameOrId("rate_sheet_entries")
	if err != nil {
		t.Fatalf("failed to get collection: %v", err)
	}

	for _, role := range roles {
		record := core.NewRecord(rateSheetEntriesCollection)
		record.Set("role", role.Id)
		record.Set("rate_sheet", rateSheetId)
		record.Set("rate", 100)
		record.Set("overtime_rate", 150)
		if err := app.Save(record); err != nil {
			t.Fatalf("failed to create entry: %v", err)
		}
	}

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
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	rateSheetId := "test_empty_sheet"

	// Get all roles
	roles, err := app.FindRecordsByFilter("rate_roles", "1=1", "", 100, 0, nil)
	if err != nil {
		t.Fatalf("failed to get roles: %v", err)
	}

	if len(roles) < 5 {
		t.Fatalf("expected at least 5 roles, got %d", len(roles))
	}

	// Create entries for first 5 roles only
	rateSheetEntriesCollection, err := app.FindCollectionByNameOrId("rate_sheet_entries")
	if err != nil {
		t.Fatalf("failed to get collection: %v", err)
	}

	for i := 0; i < 5; i++ {
		record := core.NewRecord(rateSheetEntriesCollection)
		record.Set("role", roles[i].Id)
		record.Set("rate_sheet", rateSheetId)
		record.Set("rate", 100)
		record.Set("overtime_rate", 150)
		if err := app.Save(record); err != nil {
			t.Fatalf("failed to create entry: %v", err)
		}
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
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
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
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	collection, err := app.FindCollectionByNameOrId("rate_sheets")
	if err != nil {
		t.Fatalf("failed to get collection: %v", err)
	}

	sheetName := "Test Effective Date Validation"

	// Create revision 0
	rev0 := core.NewRecord(collection)
	rev0.Set("name", sheetName)
	rev0.Set("effective_date", "2025-01-15")
	rev0.Set("revision", 0)
	rev0.Set("active", false)
	if err := app.Save(rev0); err != nil {
		t.Fatalf("failed to create revision 0: %v", err)
	}

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
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	collection, err := app.FindCollectionByNameOrId("rate_sheets")
	if err != nil {
		t.Fatalf("failed to get collection: %v", err)
	}

	sheetName := "Test Invalid Effective Date"

	// Create revision 0 with effective date 2025-06-15
	rev0 := core.NewRecord(collection)
	rev0.Set("name", sheetName)
	rev0.Set("effective_date", "2025-06-15")
	rev0.Set("revision", 0)
	rev0.Set("active", false)
	if err := app.Save(rev0); err != nil {
		t.Fatalf("failed to create revision 0: %v", err)
	}

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
