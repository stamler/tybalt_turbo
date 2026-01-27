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

	missingRoles, err := validateRateSheetComplete(app, rateSheetId)
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
	missingRoles, err := validateRateSheetComplete(app, rateSheetId)
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
	missingRoles, err := validateRateSheetComplete(app, rateSheetId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := len(roles) - 5
	if len(missingRoles) != expected {
		t.Errorf("expected %d missing roles, got %d", expected, len(missingRoles))
	}
}
