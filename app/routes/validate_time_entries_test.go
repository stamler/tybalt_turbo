package routes

import (
	"testing"
	"time"
	"tybalt/internal/testseed"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
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
	app := testseed.NewSeededTestApp(t)
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
	app := testseed.NewSeededTestApp(t)
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
	app := testseed.NewSeededTestApp(t)
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
	app := testseed.NewSeededTestApp(t)
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

func TestValidateTimeEntries_UntrackedTimeOffSkipsMinimumHoursCheck(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	uid := "u_no_claims"
	weekEnding := "2024-01-13"
	rId := getTimeTypeId(t, app, "R")

	timeEntriesCollection, _ := app.FindCollectionByNameOrId("time_entries")
	entry := core.NewRecord(timeEntriesCollection)
	entry.Set("uid", uid)
	entry.Set("time_type", rId)
	entry.Set("hours", 32.0)
	entry.Set("date", "2024-01-08")
	entry.Set("week_ending", weekEnding)

	payrollYearEndDate, _ := time.Parse("2006-01-02", "2024-01-06")

	adminProfile, err := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": uid})
	if err != nil {
		t.Fatalf("failed to fetch admin_profile: %v", err)
	}
	adminProfile.Set("salary", true)
	adminProfile.Set("skip_min_time_check", "no")
	adminProfile.Set("untracked_time_off", true)
	adminProfile.Set("work_week_hours", 40)

	if err := validateTimeEntries(app, adminProfile, payrollYearEndDate, []*core.Record{entry}); err != nil {
		t.Fatalf("validation should have passed for salaried staff with untracked time off, got error: %v", err)
	}
}

func TestValidateTimeEntries_UntrackedTimeOffRejectsRestrictedLeaveTypes(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	uid := "u_no_claims"
	weekEnding := "2024-01-13"
	payrollYearEndDate, _ := time.Parse("2006-01-02", "2024-01-06")

	adminProfile, err := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": uid})
	if err != nil {
		t.Fatalf("failed to fetch admin_profile: %v", err)
	}
	adminProfile.Set("salary", true)
	adminProfile.Set("skip_min_time_check", "no")
	adminProfile.Set("untracked_time_off", true)

	timeEntriesCollection, _ := app.FindCollectionByNameOrId("time_entries")
	for _, code := range []string{"OB", "OH", "OP", "OV"} {
		entry := core.NewRecord(timeEntriesCollection)
		entry.Set("uid", uid)
		entry.Set("time_type", getTimeTypeId(t, app, code))
		entry.Set("hours", 8.0)
		entry.Set("date", "2024-01-08")
		entry.Set("week_ending", weekEnding)

		err := validateTimeEntries(app, adminProfile, payrollYearEndDate, []*core.Record{entry})
		if err == nil {
			t.Fatalf("expected validation to fail for %s when untracked time off is enabled", code)
		}

		codeErr, ok := err.(*CodeError)
		if !ok {
			t.Fatalf("expected CodeError for %s, got %v", code, err)
		}
		if codeErr.Code != "untracked_time_off_restricted" {
			t.Fatalf("expected error code untracked_time_off_restricted for %s, got %s", code, codeErr.Code)
		}
	}
}
