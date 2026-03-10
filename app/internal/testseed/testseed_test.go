package testseed

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

var importBaselineEmptyTables = []string{
	"users",
	"absorb_actions",
	"admin_profiles",
	"categories",
	"client_contacts",
	"client_notes",
	"clients",
	"expenses",
	"job_time_allocations",
	"jobs",
	"machine_secrets",
	"notifications",
	"po_approver_props",
	"profiles",
	"purchase_orders",
	"time_amendments",
	"time_entries",
	"time_sheet_reviewers",
	"time_sheets",
	"user_claims",
	"vendors",
}

var importBaselineRequiredTables = []string{
	"app_config",
	"branches",
	"claims",
	"divisions",
	"expenditure_kinds",
	"expense_rates",
	"notification_templates",
	"payroll_year_end_dates",
	"rate_roles",
	"rate_sheets",
	"time_types",
}

func TestBuildSeededDataDirForProfile_ImportBaseline(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := BuildSeededDataDirForProfile(dir, DefaultSeedDir(), ImportBaselineProfile); err != nil {
		t.Fatalf("build import baseline: %v", err)
	}

	db := openTestSQLite(t, filepath.Join(dir, "data.db"))

	for _, table := range importBaselineEmptyTables {
		if got := rowCount(t, db, table); got != 0 {
			t.Fatalf("%s should be empty in %s, got %d rows", table, ImportBaselineProfile, got)
		}
	}

	for _, table := range importBaselineRequiredTables {
		if got := rowCount(t, db, table); got == 0 {
			t.Fatalf("%s should be populated in %s", table, ImportBaselineProfile)
		}
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM rate_sheets WHERE id = 'test_empty_sheet'").Scan(&count); err != nil {
		t.Fatalf("query test_empty_sheet: %v", err)
	}
	if count != 0 {
		t.Fatalf("test_empty_sheet should be excluded from %s", ImportBaselineProfile)
	}
}

func TestBuildSeededDataDirForProfile_RejectsUnknownProfile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := BuildSeededDataDirForProfile(dir, DefaultSeedDir(), "does-not-exist"); err == nil {
		t.Fatal("expected unknown profile error")
	}
}

func TestImportBaselineEmptyTablesMatchProductionMap(t *testing.T) {
	t.Parallel()

	if len(importBaselineEmptyTables) != len(testFullOnlyTables) {
		t.Fatalf("import-baseline empty table count mismatch: tests=%d prod=%d", len(importBaselineEmptyTables), len(testFullOnlyTables))
	}

	for _, table := range importBaselineEmptyTables {
		if _, ok := testFullOnlyTables[table]; !ok {
			t.Fatalf("table %s is expected empty in tests but missing from production split", table)
		}
	}

	for table := range testFullOnlyTables {
		found := false
		for _, expected := range importBaselineEmptyTables {
			if expected == table {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("production split table %s missing from explicit import-baseline expectations", table)
		}
	}
}

func TestImportBaselineMatchesCleanupOracle(t *testing.T) {
	t.Parallel()

	fullDir := t.TempDir()
	if err := BuildSeededDataDirForProfile(fullDir, DefaultSeedDir(), TestFullProfile); err != nil {
		t.Fatalf("build %s: %v", TestFullProfile, err)
	}

	baselineDir := t.TempDir()
	if err := BuildSeededDataDirForProfile(baselineDir, DefaultSeedDir(), ImportBaselineProfile); err != nil {
		t.Fatalf("build %s: %v", ImportBaselineProfile, err)
	}

	fullDBPath := filepath.Join(fullDir, "data.db")
	if err := applyImportBaselineCleanupOracle(fullDBPath); err != nil {
		t.Fatalf("cleanup oracle: %v", err)
	}

	pkg, err := readPackage(DefaultSeedDir())
	if err != nil {
		t.Fatalf("read package: %v", err)
	}

	resources := filterResources(pkg.Resources, ImportBaselineProfile)
	oracleRows, err := collectVerifyRows(fullDBPath, resources)
	if err != nil {
		t.Fatalf("collect oracle rows: %v", err)
	}
	baselineRows, err := collectVerifyRows(filepath.Join(baselineDir, "data.db"), resources)
	if err != nil {
		t.Fatalf("collect baseline rows: %v", err)
	}

	if len(oracleRows) != len(baselineRows) {
		t.Fatalf("row set length mismatch: oracle=%d baseline=%d", len(oracleRows), len(baselineRows))
	}

	for i := range oracleRows {
		if oracleRows[i] != baselineRows[i] {
			t.Fatalf("mismatch for %s: oracle=%+v baseline=%+v", oracleRows[i].Table, oracleRows[i], baselineRows[i])
		}
	}
}

func openTestSQLite(t *testing.T, dbPath string) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func rowCount(t *testing.T, db *sql.DB, table string) int {
	t.Helper()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + quoteIdent(table)).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func applyImportBaselineCleanupOracle(dbPath string) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if _, err := db.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		return err
	}
	defer db.Exec("PRAGMA foreign_keys = ON")

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, table := range importBaselineEmptyTables {
		if _, err := tx.Exec("DELETE FROM " + quoteIdent(table)); err != nil {
			return err
		}
	}

	if _, err := tx.Exec("DELETE FROM rate_sheets WHERE id = 'test_empty_sheet'"); err != nil {
		return err
	}

	return tx.Commit()
}
