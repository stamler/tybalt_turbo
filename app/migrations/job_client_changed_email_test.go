package migrations_test

import (
	"encoding/csv"
	"os"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

const jobClientChangedEmailMigration = "1792003502_job_client_changed_email.go"

// The seeded test data must send the same client change email as a migrated
// database.
func TestJobClientChangedEmailFixtureMatchesTheMigration(t *testing.T) {
	app := newMigratedApp(t)
	assertWorkflowEmails(t, app, []string{"job_client_changed"}, nil, nil)
	template, err := app.FindFirstRecordByData("notification_templates", "code", "job_client_changed")
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open("../test_seed_data/data/notification_templates.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	columns := map[string]int{}
	for i, name := range rows[0] {
		columns[name] = i
	}
	for _, row := range rows[1:] {
		if row[columns["code"]] != "job_client_changed" {
			continue
		}
		for _, field := range []string{"id", "description", "subject", "text_email", "html_email"} {
			if row[columns[field]] != template.GetString(field) {
				t.Errorf("fixture %s = %q; migration has %q", field, row[columns[field]], template.GetString(field))
			}
		}
		return
	}
	t.Fatal("fixtures are missing the job_client_changed template")
}

// Undoing the migration removes the email and its flag, and leaves the other
// flags as they were.
func TestJobClientChangedEmailMigrationRoundTrip(t *testing.T) {
	app := newMigratedApp(t)
	var migration *core.Migration
	for _, item := range core.AppMigrations.Items() {
		if item.File == jobClientChangedEmailMigration {
			migration = item
		}
	}
	if migration == nil {
		t.Fatal("client change email migration is not registered")
	}
	before := notificationFlags(t, app)
	if err := migration.Down(app); err != nil {
		t.Fatal(err)
	}
	assertWorkflowEmails(t, app, nil, []string{"job_client_changed"}, nil)
	if err := migration.Up(app); err != nil {
		t.Fatal(err)
	}
	assertWorkflowEmails(t, app, []string{"job_client_changed"}, nil, nil)
	after := notificationFlags(t, app)
	for code, value := range before {
		if after[code] != value {
			t.Errorf("notification flag %s changed: %v -> %v", code, value, after[code])
		}
	}
}
