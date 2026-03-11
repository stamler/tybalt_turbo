package main

import (
	"database/sql"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	_ "modernc.org/sqlite"
)

func TestImportPoApproverProps_ImportsCommittedParquetIntoSeededImportBaselineDB(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "pb_data")
	buildImportBaselineDataDir(t, dataDir)

	repoRoot := repoRootForTest(t)
	chdirForTest(t, tempDir)
	linkParquetFixture(t, filepath.Join(repoRoot, "import_data", "parquet", "PoApproverProps.parquet"), filepath.Join(tempDir, "parquet", "PoApproverProps.parquet"))
	linkParquetFixture(t, filepath.Join(repoRoot, "import_data", "parquet", "Profiles.parquet"), filepath.Join(tempDir, "parquet", "Profiles.parquet"))

	dbPath := filepath.Join(dataDir, "data.db")
	initialPropsCount := countRows(t, dbPath, "SELECT COUNT(*) FROM po_approver_props")
	initialPoApproverClaimCount := countRows(t, dbPath, "SELECT COUNT(*) FROM user_claims WHERE cid = (SELECT id FROM claims WHERE name = 'po_approver')")

	rowsByUID, parquetExists, err := loadTurboPoApproverProps("./parquet/PoApproverProps.parquet")
	if err != nil {
		t.Fatalf("loadTurboPoApproverProps returned error: %v", err)
	}
	if !parquetExists {
		t.Fatal("expected committed PoApproverProps.parquet to exist")
	}

	stats, err := importPoApproverProps(dbPath)
	if err != nil {
		t.Fatalf("importPoApproverProps returned error: %v", err)
	}

	wantInserted := len(rowsByUID)
	if stats.inserted != wantInserted {
		t.Fatalf("inserted count = %d, want %d", stats.inserted, wantInserted)
	}

	if got := countRows(t, dbPath, "SELECT COUNT(*) FROM po_approver_props"); got != initialPropsCount+wantInserted {
		t.Fatalf("po_approver_props count = %d, want %d", got, initialPropsCount+wantInserted)
	}

	if got := countRows(t, dbPath, "SELECT COUNT(*) FROM user_claims WHERE cid = (SELECT id FROM claims WHERE name = 'po_approver')"); got != initialPoApproverClaimCount+wantInserted {
		t.Fatalf("po_approver user_claim count = %d, want %d", got, initialPoApproverClaimCount+wantInserted)
	}

	importedRow := firstImportedPoApproverPropsRow(t, dbPath)
	wantRow, ok := rowsByUID[importedRow.uid]
	if !ok {
		t.Fatalf("imported uid %s not found in parquet fixture set", importedRow.uid)
	}

	if importedRow.id != wantRow.id {
		t.Fatalf("imported id = %s, want %s", importedRow.id, wantRow.id)
	}
	assertFloatEqual(t, "max_amount", importedRow.maxAmount, wantRow.maxAmount)
	assertFloatEqual(t, "project_max", importedRow.projectMax, wantRow.projectMax)
	assertFloatEqual(t, "sponsorship_max", importedRow.sponsorshipMax, wantRow.sponsorshipMax)
	assertFloatEqual(t, "staff_and_social_max", importedRow.staffAndSocialMax, wantRow.staffAndSocialMax)
	assertFloatEqual(t, "media_and_event_max", importedRow.mediaAndEventMax, wantRow.mediaAndEventMax)
	assertFloatEqual(t, "computer_max", importedRow.computerMax, wantRow.computerMax)
	if importedRow.divisionsJSON != wantRow.divisionsJSON {
		t.Fatalf("imported divisions = %s, want %s", importedRow.divisionsJSON, wantRow.divisionsJSON)
	}
	if importedRow.created != wantRow.created {
		t.Fatalf("imported created = %s, want %s", importedRow.created, wantRow.created)
	}
	if importedRow.updated != wantRow.updated {
		t.Fatalf("imported updated = %s, want %s", importedRow.updated, wantRow.updated)
	}
}

func TestImportPoApproverProps_ErrorsWithoutParquetAndLeavesSeededDBUntouched(t *testing.T) {
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "pb_data")
	buildImportBaselineDataDir(t, dataDir)
	chdirForTest(t, tempDir)

	dbPath := filepath.Join(dataDir, "data.db")
	initialPropsCount := countRows(t, dbPath, "SELECT COUNT(*) FROM po_approver_props")
	initialPoApproverClaimCount := countRows(t, dbPath, "SELECT COUNT(*) FROM user_claims WHERE cid = (SELECT id FROM claims WHERE name = 'po_approver')")

	stats, err := importPoApproverProps(dbPath)
	if err == nil {
		t.Fatal("expected importPoApproverProps to fail when PoApproverProps.parquet is missing")
	}
	if stats.inserted != 0 {
		t.Fatalf("inserted count = %d, want 0", stats.inserted)
	}

	if got := countRows(t, dbPath, "SELECT COUNT(*) FROM po_approver_props"); got != initialPropsCount {
		t.Fatalf("po_approver_props count = %d, want %d", got, initialPropsCount)
	}

	if got := countRows(t, dbPath, "SELECT COUNT(*) FROM user_claims WHERE cid = (SELECT id FROM claims WHERE name = 'po_approver')"); got != initialPoApproverClaimCount {
		t.Fatalf("po_approver user_claim count = %d, want %d", got, initialPoApproverClaimCount)
	}
}

func buildImportBaselineDataDir(t *testing.T, dataDir string) {
	t.Helper()

	repoRoot := repoRootForTest(t)
	cmd := exec.Command("go", "run", "./cmd/testseed", "load", "--profile", "import-baseline", "--out", dataDir)
	cmd.Dir = filepath.Join(repoRoot, "app")
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build import-baseline data dir: %v\n%s", err, output)
	}
}

func repoRootForTest(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), ".."))
}

type importedPoApproverPropsRow struct {
	id                string
	uid               string
	maxAmount         float64
	projectMax        float64
	sponsorshipMax    float64
	staffAndSocialMax float64
	mediaAndEventMax  float64
	computerMax       float64
	divisionsJSON     string
	created           string
	updated           string
}

func linkParquetFixture(t *testing.T, sourcePath string, targetPath string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(targetPath), err)
	}
	if err := os.Symlink(sourcePath, targetPath); err != nil {
		t.Fatalf("symlink %s -> %s: %v", targetPath, sourcePath, err)
	}
}

func chdirForTest(t *testing.T, dir string) {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd to %s: %v", cwd, err)
		}
	})
}

func countRows(t *testing.T, dbPath, query string) int {
	t.Helper()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow(query).Scan(&count); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return count
}

func firstImportedPoApproverPropsRow(t *testing.T, dbPath string) importedPoApproverPropsRow {
	t.Helper()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer db.Close()

	var row importedPoApproverPropsRow
	query := `
		SELECT
			pap.id,
			uc.uid,
			pap.max_amount,
			pap.project_max,
			pap.sponsorship_max,
			pap.staff_and_social_max,
			pap.media_and_event_max,
			pap.computer_max,
			pap.divisions,
			pap.created,
			pap.updated
		FROM po_approver_props pap
		JOIN user_claims uc ON uc.id = pap.user_claim
		ORDER BY pap.id
		LIMIT 1
	`
	if err := db.QueryRow(query).Scan(
		&row.id,
		&row.uid,
		&row.maxAmount,
		&row.projectMax,
		&row.sponsorshipMax,
		&row.staffAndSocialMax,
		&row.mediaAndEventMax,
		&row.computerMax,
		&row.divisionsJSON,
		&row.created,
		&row.updated,
	); err != nil {
		t.Fatalf("query first imported po_approver_props row: %v", err)
	}

	return row
}

func assertFloatEqual(t *testing.T, field string, got float64, want float64) {
	t.Helper()

	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("%s = %s, want %s", field, fmt.Sprintf("%.12f", got), fmt.Sprintf("%.12f", want))
	}
}
