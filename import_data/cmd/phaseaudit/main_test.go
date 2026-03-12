package main

import (
	"bytes"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRunUnchangedSelectedPhases(t *testing.T) {
	before := filepath.Join(t.TempDir(), "before.db")
	after := filepath.Join(t.TempDir(), "after.db")

	writeFixtureDB(t, before, baseFixtureStatements())
	writeFixtureDB(t, after, baseFixtureStatements())

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--before", before, "--after", after, "--jobs", "--time"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d, stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "phase jobs") || !strings.Contains(output, "phase time") {
		t.Fatalf("expected selected phases in output, got: %s", output)
	}
	if strings.Contains(output, "phase expenses") || strings.Contains(output, "phase users") {
		t.Fatalf("expected unselected phases to be ignored, got: %s", output)
	}
	if count := strings.Count(output, "phase unchanged"); count != 2 {
		t.Fatalf("expected 2 unchanged phase markers, got %d in output: %s", count, output)
	}
}

func TestRunDetectsChangedSelectedPhase(t *testing.T) {
	before := filepath.Join(t.TempDir(), "before.db")
	after := filepath.Join(t.TempDir(), "after.db")

	writeFixtureDB(t, before, baseFixtureStatements())

	statements := baseFixtureStatements()
	statements = append(statements,
		`UPDATE expenses SET amount = 999 WHERE id = 'exp-1'`,
		`UPDATE jobs SET value = 'job-mutated-but-unselected' WHERE id = 'job-1'`,
	)
	writeFixtureDB(t, after, statements)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--before", before, "--after", after, "--expenses"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d, want 1, stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}

	output := stdout.String()
	if !strings.Contains(output, `changed rows: id="exp-1"`) {
		t.Fatalf("expected changed expense row in output, got: %s", output)
	}
	if strings.Contains(output, "jobs") {
		t.Fatalf("expected unselected jobs phase to be omitted, got: %s", output)
	}
}

func TestRunUsersPhaseReportsFallbackForNoKeyTable(t *testing.T) {
	before := filepath.Join(t.TempDir(), "before.db")
	after := filepath.Join(t.TempDir(), "after.db")

	writeFixtureDB(t, before, baseFixtureStatements())

	statements := baseFixtureStatements()
	statements = append(statements, `UPDATE "_externalAuths" SET providerId = 'azure-2' WHERE recordRef = 'user-1'`)
	writeFixtureDB(t, after, statements)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--before", before, "--after", after, "--users"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d, want 1, stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "CHANGED _externalAuths") {
		t.Fatalf("expected _externalAuths change in output, got: %s", output)
	}
	if !strings.Contains(output, "no primary key or id column") {
		t.Fatalf("expected fallback note in output, got: %s", output)
	}
}

func writeFixtureDB(t *testing.T, path string, statements []string) {
	t.Helper()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec %q: %v", stmt, err)
		}
	}
}

func baseFixtureStatements() []string {
	return []string{
		`CREATE TABLE clients (id TEXT PRIMARY KEY, value TEXT)`,
		`INSERT INTO clients (id, value) VALUES ('client-1', 'alpha')`,
		`CREATE TABLE client_contacts (id TEXT PRIMARY KEY, value TEXT)`,
		`INSERT INTO client_contacts (id, value) VALUES ('contact-1', 'alpha')`,
		`CREATE TABLE client_notes (id TEXT PRIMARY KEY, value TEXT)`,
		`INSERT INTO client_notes (id, value) VALUES ('note-1', 'alpha')`,
		`CREATE TABLE jobs (id TEXT PRIMARY KEY, value TEXT)`,
		`INSERT INTO jobs (id, value) VALUES ('job-1', 'alpha')`,
		`CREATE TABLE categories (id TEXT PRIMARY KEY, value TEXT)`,
		`INSERT INTO categories (id, value) VALUES ('category-1', 'alpha')`,
		`CREATE TABLE job_time_allocations (id TEXT PRIMARY KEY, value TEXT)`,
		`INSERT INTO job_time_allocations (id, value) VALUES ('alloc-1', 'alpha')`,
		`CREATE TABLE vendors (id TEXT PRIMARY KEY, value TEXT)`,
		`INSERT INTO vendors (id, value) VALUES ('vendor-1', 'alpha')`,
		`CREATE TABLE purchase_orders (id TEXT PRIMARY KEY, value TEXT)`,
		`INSERT INTO purchase_orders (id, value) VALUES ('po-1', 'alpha')`,
		`CREATE TABLE expenses (id TEXT PRIMARY KEY, amount INTEGER)`,
		`INSERT INTO expenses (id, amount) VALUES ('exp-1', 100)`,
		`CREATE TABLE time_sheets (id TEXT PRIMARY KEY, value TEXT)`,
		`INSERT INTO time_sheets (id, value) VALUES ('ts-1', 'alpha')`,
		`CREATE TABLE time_entries (id TEXT PRIMARY KEY, value TEXT)`,
		`INSERT INTO time_entries (id, value) VALUES ('te-1', 'alpha')`,
		`CREATE TABLE time_amendments (id TEXT PRIMARY KEY, value TEXT)`,
		`INSERT INTO time_amendments (id, value) VALUES ('ta-1', 'alpha')`,
		`CREATE TABLE users (id TEXT PRIMARY KEY, value TEXT)`,
		`INSERT INTO users (id, value) VALUES ('user-1', 'alpha')`,
		`CREATE TABLE profiles (id TEXT PRIMARY KEY, value TEXT)`,
		`INSERT INTO profiles (id, value) VALUES ('profile-1', 'alpha')`,
		`CREATE TABLE admin_profiles (id TEXT PRIMARY KEY, value TEXT)`,
		`INSERT INTO admin_profiles (id, value) VALUES ('admin-1', 'alpha')`,
		`CREATE TABLE "_externalAuths" (collectionRef TEXT, provider TEXT, providerId TEXT, recordRef TEXT)`,
		`INSERT INTO "_externalAuths" (collectionRef, provider, providerId, recordRef) VALUES ('_pb_users_auth_', 'microsoft', 'azure-1', 'user-1')`,
		`CREATE TABLE user_claims (id TEXT PRIMARY KEY, value TEXT)`,
		`INSERT INTO user_claims (id, value) VALUES ('claim-1', 'alpha')`,
		`CREATE TABLE po_approver_props (id TEXT PRIMARY KEY, value TEXT)`,
		`INSERT INTO po_approver_props (id, value) VALUES ('prop-1', 'alpha')`,
		`CREATE TABLE mileage_reset_dates (id TEXT PRIMARY KEY, value TEXT)`,
		`INSERT INTO mileage_reset_dates (id, value) VALUES ('mileage-1', 'alpha')`,
	}
}
