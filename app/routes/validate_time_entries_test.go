package routes

import (
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// helper to fetch a time_type id by code
func getTimeTypeId(t *testing.T, app core.App, code string) string {
	record, err := app.FindFirstRecordByFilter("time_types", "code={:code}", dbx.Params{"code": code})
	if err != nil {
		t.Fatalf("failed to load time_type %s: %v", code, err)
	}
	return record.Id
}

// ──────────────────────────────────────────────────────────────────────────────
// Scenario 1: User has negative Vacation balance but is NOT making a new OV
//
//	claim on this bundle → validation should succeed.
//
// ──────────────────────────────────────────────────────────────────────────────
func TestValidateTimeEntries_NegativeBalanceNoOVClaims_Passes(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatalf("failed to init test app: %v", err)
	}
	defer app.Cleanup()

	uid := "u_no_claims"
	weekEnding := "2024-01-13" // Saturday

	// create current R entry (40h) without OV/OP claims
	rId := getTimeTypeId(t, app, "R")

	timeEntriesCollection, _ := app.FindCollectionByNameOrId("time_entries")
	entry := core.NewRecord(timeEntriesCollection)
	entry.Set("uid", uid)
	entry.Set("time_type", rId)
	entry.Set("hours", 40.0)
	entry.Set("date", "2024-01-08")
	entry.Set("week_ending", weekEnding)

	// payroll year end date (must be before weekEnding)
	payrollYearEndDate, _ := time.Parse("2006-01-02", "2024-01-06")

	// fetch admin_profile record
	adminProfile, err := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": uid})
	if err != nil {
		t.Fatalf("failed to fetch admin_profile: %v", err)
	}

	if err := validateTimeEntries(app, adminProfile, payrollYearEndDate, []*core.Record{entry}); err != nil {
		t.Fatalf("validation should have passed, got error: %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Scenario 2: User has negative Vacation balance AND is claiming additional OV
//
//	→ validator must reject with ov_claim_exceeds_balance.
//
// ──────────────────────────────────────────────────────────────────────────────
func TestValidateTimeEntries_ClaimOVWithNegativeBalance_Fails(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatalf("failed to init test app: %v", err)
	}
	defer app.Cleanup()

	uid := "u_with_claim"
	weekEnding := "2024-01-13" // Saturday

	// create current OV entry (1h) which should now trigger error
	ovId := getTimeTypeId(t, app, "OV")
	timeEntriesCollection, _ := app.FindCollectionByNameOrId("time_entries")
	entry := core.NewRecord(timeEntriesCollection)
	entry.Set("uid", uid)
	entry.Set("time_type", ovId)
	entry.Set("hours", 1.0)
	entry.Set("date", "2024-01-08")
	entry.Set("week_ending", weekEnding)

	payrollYearEndDate, _ := time.Parse("2006-01-02", "2024-01-06")

	adminProfile, err := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": uid})
	if err != nil {
		t.Fatalf("failed to fetch admin_profile: %v", err)
	}

	if err := validateTimeEntries(app, adminProfile, payrollYearEndDate, []*core.Record{entry}); err == nil {
		t.Fatalf("expected validation to fail but it passed")
	} else {
		if codeErr, ok := err.(*CodeError); ok {
			if codeErr.Code != "ov_claim_exceeds_balance" {
				t.Fatalf("expected error code ov_claim_exceeds_balance, got %s", codeErr.Code)
			}
		} else {
			t.Fatalf("expected CodeError, got %v", err)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Scenario 3: User has negative PPTO balance but is NOT making a new OP claim
//
//	on this bundle → validation should succeed.
//
// ──────────────────────────────────────────────────────────────────────────────
func TestValidateTimeEntries_NegativeBalanceNoOPClaims_Passes(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatalf("failed to init test app: %v", err)
	}
	defer app.Cleanup()

	uid := "u_no_ppto_claim"
	weekEnding := "2024-01-13"

	rId := getTimeTypeId(t, app, "R")

	timeEntriesCollection, _ := app.FindCollectionByNameOrId("time_entries")
	entry := core.NewRecord(timeEntriesCollection)
	entry.Set("uid", uid)
	entry.Set("time_type", rId)
	entry.Set("hours", 40.0)
	entry.Set("date", "2024-01-08")
	entry.Set("week_ending", weekEnding)

	payrollYearEndDate, _ := time.Parse("2006-01-02", "2024-01-06")

	adminProfile, err := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": uid})
	if err != nil {
		t.Fatalf("failed to fetch admin_profile: %v", err)
	}

	if err := validateTimeEntries(app, adminProfile, payrollYearEndDate, []*core.Record{entry}); err != nil {
		t.Fatalf("validation should have passed, got error: %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Scenario 4: User has negative PPTO balance AND is claiming additional OP
//
//	→ validator must reject with ppto_claim_exceeds_balance.
//
// ──────────────────────────────────────────────────────────────────────────────
func TestValidateTimeEntries_ClaimOPWithNegativeBalance_Fails(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatalf("failed to init test app: %v", err)
	}
	defer app.Cleanup()

	uid := "u_with_ppto_claim"
	weekEnding := "2024-01-13"

	opId := getTimeTypeId(t, app, "OP")

	timeEntriesCollection, _ := app.FindCollectionByNameOrId("time_entries")
	entry := core.NewRecord(timeEntriesCollection)
	entry.Set("uid", uid)
	entry.Set("time_type", opId)
	entry.Set("hours", 1.0)
	entry.Set("date", "2024-01-08")
	entry.Set("week_ending", weekEnding)

	payrollYearEndDate, _ := time.Parse("2006-01-02", "2024-01-06")

	adminProfile, err := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": uid})
	if err != nil {
		t.Fatalf("failed to fetch admin_profile: %v", err)
	}

	if err := validateTimeEntries(app, adminProfile, payrollYearEndDate, []*core.Record{entry}); err == nil {
		t.Fatalf("expected validation to fail but it passed")
	} else {
		if codeErr, ok := err.(*CodeError); ok {
			if codeErr.Code != "ppto_claim_exceeds_balance" {
				t.Fatalf("expected error code ppto_claim_exceeds_balance, got %s", codeErr.Code)
			}
		} else {
			t.Fatalf("expected CodeError, got %v", err)
		}
	}
}
