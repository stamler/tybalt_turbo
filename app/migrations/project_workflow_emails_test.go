package migrations_test

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	_ "tybalt/migrations"
)

const workflowEmailsMigration = "1791913591_project_workflow_emails.go"

var workflowEmailCodes = []string{
	"job_value_submitted", "job_value_approved", "job_value_rejected",
	"job_request_submitted", "job_request_approved", "job_request_rejected", "job_request_created",
}
var replacedWorkflowEmailCodes = []string{"job_request_workflow", "job_value_workflow"}

func newMigratedApp(t *testing.T) core.App {
	t.Helper()
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.ResetBootstrapState() })
	if err := app.RunAllMigrations(); err != nil {
		t.Fatal(err)
	}
	return app
}

func notificationFlags(t *testing.T, app core.App) map[string]any {
	t.Helper()
	config, err := app.FindFirstRecordByData("app_config", "key", "notifications")
	if err != nil {
		t.Fatal(err)
	}
	flags := map[string]any{}
	if err = json.Unmarshal([]byte(config.GetString("value")), &flags); err != nil {
		t.Fatal(err)
	}
	return flags
}

// assertWorkflowEmails checks which workflow templates exist and are switched
// on. A template listed in kept exists but is switched off.
func assertWorkflowEmails(t *testing.T, app core.App, on, off, kept []string) {
	t.Helper()
	flags := notificationFlags(t, app)
	for _, code := range on {
		if _, err := app.FindFirstRecordByData("notification_templates", "code", code); err != nil {
			t.Errorf("template %s is missing", code)
		}
		if flags[code] != true {
			t.Errorf("template %s is not switched on", code)
		}
	}
	for _, code := range off {
		_, err := app.FindFirstRecordByData("notification_templates", "code", code)
		isKept := false
		for _, k := range kept {
			isKept = isKept || k == code
		}
		if (err == nil) != isKept {
			t.Errorf("template %s exists = %v; want %v", code, err == nil, isKept)
		}
		if _, exists := flags[code]; exists {
			t.Errorf("replaced template %s still has a flag", code)
		}
	}
}

func workflowEmailsMigrationItem(t *testing.T) *core.Migration {
	t.Helper()
	for _, migration := range core.AppMigrations.Items() {
		if migration.File == workflowEmailsMigration {
			return migration
		}
	}
	t.Fatal("workflow emails migration is not registered")
	return nil
}

func TestWorkflowEmailsReplaceTheRequestWorkflowNotices(t *testing.T) {
	app := newMigratedApp(t)
	assertWorkflowEmails(t, app, workflowEmailCodes, replacedWorkflowEmailCodes, nil)
}

// The seeded test data must send the same emails as a migrated database.
func TestWorkflowEmailFixturesMatchTheMigration(t *testing.T) {
	app := newMigratedApp(t)
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
	fixtures := map[string][]string{}
	for _, row := range rows[1:] {
		fixtures[row[columns["code"]]] = row
	}
	for _, code := range replacedWorkflowEmailCodes {
		if _, ok := fixtures[code]; ok {
			t.Errorf("fixtures still contain replaced template %s", code)
		}
	}
	for _, code := range workflowEmailCodes {
		template, err := app.FindFirstRecordByData("notification_templates", "code", code)
		if err != nil {
			t.Fatal(err)
		}
		row, ok := fixtures[code]
		if !ok {
			t.Errorf("fixtures are missing template %s", code)
			continue
		}
		for _, field := range []string{"id", "description", "subject", "text_email", "html_email"} {
			if row[columns[field]] != template.GetString(field) {
				t.Errorf("%s fixture %s = %q; migration has %q", code, field, row[columns[field]], template.GetString(field))
			}
		}
	}
}

// Undoing the migration restores the two earlier templates. A template that a
// sent notification uses stays, switched off, and is reused when redone.
func TestWorkflowEmailsMigrationRoundTrip(t *testing.T) {
	app := newMigratedApp(t)
	migration := workflowEmailsMigrationItem(t)
	if _, err := app.DB().NewQuery("INSERT INTO notifications (id, recipient, template, status) VALUES ('sentworkflow001', 'anyrecipient001', 'pavalsubmitted0', 'sent')").Execute(); err != nil {
		t.Fatal(err)
	}
	if err := migration.Down(app); err != nil {
		t.Fatal(err)
	}
	assertWorkflowEmails(t, app, replacedWorkflowEmailCodes, workflowEmailCodes, []string{"job_value_submitted"})
	unrelated := notificationFlags(t, app)
	for _, code := range replacedWorkflowEmailCodes {
		delete(unrelated, code)
	}
	if err := migration.Up(app); err != nil {
		t.Fatal(err)
	}
	assertWorkflowEmails(t, app, workflowEmailCodes, replacedWorkflowEmailCodes, nil)
	flags := notificationFlags(t, app)
	for code, value := range unrelated {
		if flags[code] != value {
			t.Errorf("unrelated notification flag %s changed: %v -> %v", code, value, flags[code])
		}
	}
	template, err := app.FindFirstRecordByData("notification_templates", "code", "job_value_submitted")
	if err != nil || template.Id != "pavalsubmitted0" {
		t.Fatalf("the kept template was not reused: %v", err)
	}
}
