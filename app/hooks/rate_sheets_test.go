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
//   - rate_sheets: "c41ofep525bcacj" (2025 Standard Rates, active=false)
//   - rate_roles: 30 roles
//   - rate_sheet_entries: none
//
// FIXTURE DEPENDENCY: This test assumes no rate_sheet_entries exist for the test
// rate sheet. If entries are added to the fixture, either create a new rate_sheet
// for this test or delete this test entirely (the other tests cover the core logic).
func TestValidateRateSheetComplete_NoEntries(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	rateSheetId := "c41ofep525bcacj"

	missingRoles, err := validateRateSheetComplete(app, rateSheetId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 30 missing roles (all of them)
	if len(missingRoles) != 30 {
		t.Errorf("expected 30 missing roles, got %d", len(missingRoles))
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

	rateSheetId := "c41ofep525bcacj"

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
//
// FIXTURE DEPENDENCY: This test assumes no rate_sheet_entries exist for the test
// rate sheet. If entries are added to the fixture, create a new rate_sheet for this
// test to ensure a known starting state.
func TestValidateRateSheetComplete_PartialEntries(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	rateSheetId := "c41ofep525bcacj"

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
