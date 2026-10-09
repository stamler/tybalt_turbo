package migrations_test

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	_ "tybalt/migrations"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const clientSetupMigration = "1790969510_client_setup.go"

func newClientSetupMigrationApp(t *testing.T) (core.App, *core.Migration) {
	t.Helper()
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.ResetBootstrapState() })
	if _, err := core.NewMigrationsRunner(app, core.SystemMigrations).Up(); err != nil {
		t.Fatal(err)
	}
	previous := core.MigrationsList{}
	var setup *core.Migration
	for _, migration := range core.AppMigrations.Items() {
		switch migration.File {
		case clientSetupMigration:
			setup = migration
		default:
			if migration.File < clientSetupMigration {
				previous.Add(migration)
			}
		}
	}
	if setup == nil {
		t.Fatal("client setup migration must be registered")
	}
	if _, err := core.NewMigrationsRunner(app, previous).Up(); err != nil {
		t.Fatal(err)
	}
	return app, setup
}

func applyMigration(t *testing.T, app core.App, migration *core.Migration) {
	t.Helper()
	list := core.MigrationsList{}
	list.Add(migration)
	applied, err := core.NewMigrationsRunner(app, list).Up()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(applied, []string{migration.File}) {
		t.Fatalf("expected to apply %s once; got %v", migration.File, applied)
	}
}

func assertNoPendingMigrations(t *testing.T, app core.App, migrations ...*core.Migration) {
	t.Helper()
	list := core.MigrationsList{}
	for _, migration := range migrations {
		list.Add(migration)
	}
	applied, err := core.NewMigrationsRunner(app, list).Up()
	if err != nil || len(applied) != 0 {
		t.Fatalf("a second runner call must be a no-op; applied=%v, err=%v", applied, err)
	}
}

func migrationCollection(t *testing.T, app core.App, name string) *core.Collection {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		t.Fatal(err)
	}
	return collection
}

func loadMigrationRecord(t *testing.T, app core.App, collection, path, id string) *core.Record {
	t.Helper()
	record := core.NewRecord(migrationCollection(t, app, collection))
	for key, value := range fixtureRow(t, path, id) {
		if record.Collection().Fields.GetByName(key) != nil {
			record.Set(key, value)
		}
	}
	// These migration fixtures can reference records outside this small database.
	// Bypass relation validation, as the existing migration preservation test does.
	if err := app.SaveNoValidate(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func assertClientSetupStage(t *testing.T, app core.App) {
	t.Helper()
	for _, name := range []string{"job_requests", "job_attachments", "job_request_allocations", "project_authorization_props"} {
		if _, err := app.FindCollectionByNameOrId(name); err == nil {
			t.Errorf("client setup must not install %s", name)
		}
	}
	profiles := migrationCollection(t, app, "client_invoicing_information")
	for _, rule := range []*string{profiles.CreateRule, profiles.UpdateRule, profiles.DeleteRule} {
		if rule == nil || !strings.Contains(*rule, "?= 'job'") || strings.Contains(*rule, "?= 'time'") || strings.Contains(*rule, "job_requests") {
			t.Fatalf("client setup profile writes must require job, without request dependencies: %v", rule)
		}
	}
	for _, rule := range []*string{profiles.ListRule, profiles.ViewRule} {
		if rule == nil || !strings.Contains(*rule, "active = true") {
			t.Fatal("profile reads must require an active user")
		}
	}
	jobs := migrationCollection(t, app, "jobs")
	invoice, ok := jobs.Fields.GetByName("invoicing_information").(*core.RelationField)
	if !ok || invoice.Required || invoice.CascadeDelete || invoice.CollectionId != profiles.Id || invoice.MaxSelect != 1 {
		t.Fatalf("job profile must be an optional, non-cascading relation: %#v", invoice)
	}
	for _, field := range []string{"project_authorization_doc", "project_authorization_doc_hash", "pa_reviewer", "pa_reviewed", "pa_uploader", "pa_uploaded", "pa_rejector", "pa_rejected", "pa_rejection_reason", "request_creator", "created_by", "request_created", "request_submitted", "creation_completed"} {
		if jobs.Fields.GetByName(field) != nil {
			t.Errorf("client setup retains or introduces a workflow field: %s", field)
		}
	}
	if jobs.UpdateRule == nil || !strings.Contains(*jobs.UpdateRule, "'job'") || !strings.Contains(*jobs.UpdateRule, "number") || strings.Contains(*jobs.UpdateRule, "project_value") || strings.Contains(*jobs.UpdateRule, "client:changed") || strings.Contains(*jobs.UpdateRule, "pa_") || strings.Contains(*jobs.UpdateRule, "project_authorization") {
		t.Fatalf("client setup must retain main job edit access without retired guards: %v", jobs.UpdateRule)
	}
	for _, name := range []string{"clients", "client_contacts"} {
		collection := migrationCollection(t, app, name)
		for _, field := range []string{"address", "city", "province_state", "postal_code", "country", "phone"} {
			text, ok := collection.Fields.GetByName(field).(*core.TextField)
			if !ok || text.Required {
				t.Errorf("%s.%s must be optional text", name, field)
			}
		}
		if collection.DeleteRule == nil || strings.Contains(*collection.DeleteRule, "job_requests") || !strings.Contains(*collection.DeleteRule, "client_invoicing_information") {
			t.Errorf("%s deletion must protect profiles without request dependencies", name)
		}
		for _, rule := range []*string{collection.CreateRule, collection.UpdateRule, collection.DeleteRule} {
			if rule == nil || !strings.Contains(*rule, "?= 'job'") || !strings.Contains(*rule, "admin_profiles_via_uid.active = true") {
				t.Errorf("%s writes must require an active user with the job claim: %v", name, rule)
			}
		}
	}
	email := migrationCollection(t, app, "client_contacts").Fields.GetByName("email").(*core.EmailField)
	if email.Required {
		t.Fatal("client setup must keep legacy contact email optional")
	}
	// The migration only cleans rows that exist; it never creates settings rows.
	retired := map[string][]string{
		"jobs":          {"enforce_project_authorization", "require_job_request_approval"},
		"notifications": {"project_authorization_rejected", "job_request_workflow", "job_value_workflow"},
	}
	for key, settings := range retired {
		config, err := app.FindFirstRecordByData("app_config", "key", key)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		var values map[string]any
		if err := json.Unmarshal([]byte(config.GetString("value")), &values); err != nil {
			t.Fatal(err)
		}
		for _, setting := range settings {
			if _, exists := values[setting]; exists {
				t.Errorf("client setup must not keep or introduce %s.%s", key, setting)
			}
		}
		if key == "jobs" && config.GetString("description") != "create_edit_absorb (default true) enables job and proposal creation/editing, manual renumbering, and client/contact merges." {
			t.Fatal("client setup configuration describes settings that are not installed")
		}
	}
	for _, code := range []string{"project_authorization_rejected", "job_request_workflow", "job_value_workflow"} {
		if record, err := app.FindFirstRecordByData("notification_templates", "code", code); err == nil {
			t.Errorf("client setup must not keep or introduce template %s: %s", code, record.Id)
		}
	}
	if _, err := app.FindFirstRecordByData("claims", "name", "project_authorization"); err == nil {
		t.Fatal("client setup must not install the request approval claim")
	}
}

// Compare stored schema with the canonical seed so API tests use the schema
// that a deployment will install. Later migrations also change these
// collections, so all migrations run first. These four collections are base
// collections; generated view field IDs do not need normalization here.
func TestClientSetupMigrationMatchesCanonicalSchema(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.ResetBootstrapState() })
	if err := app.RunAllMigrations(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"clients", "client_contacts", "jobs", "client_invoicing_information"} {
		t.Run(name, func(t *testing.T) {
			collection := migrationCollection(t, app, name)
			want := fixtureRow(t, "data/_collections.csv", collection.Id)
			var row dbx.NullStringMap
			if err := app.DB().NewQuery("SELECT * FROM _collections WHERE id = {:id}").Bind(dbx.Params{"id": collection.Id}).One(&row); err != nil {
				t.Fatal(err)
			}
			for key, expected := range want {
				if key == "created" || key == "updated" {
					continue
				}
				actual := row[key].String
				if key == "fields" || key == "indexes" || key == "options" {
					var wantValue, gotValue any
					if err := json.Unmarshal([]byte(expected), &wantValue); err != nil {
						t.Fatalf("fixture %s: %v", key, err)
					}
					if err := json.Unmarshal([]byte(actual), &gotValue); err != nil {
						t.Fatalf("migration %s: %v", key, err)
					}
					if !reflect.DeepEqual(wantValue, gotValue) {
						t.Errorf("%s differs: fixture=%s migration=%s", key, expected, actual)
					}
				} else if expected != actual {
					t.Errorf("%s differs: fixture=%q migration=%q", key, expected, actual)
				}
			}
		})
	}
}

func TestClientSetupMigrationStandsAlone(t *testing.T) {
	for _, configState := range []string{"missing", "existing", "legacy only"} {
		t.Run(configState+" config", func(t *testing.T) {
			app, setup := newClientSetupMigrationApp(t)
			allocations := migrationCollection(t, app, "job_time_allocations")
			beforeRules := []*string{allocations.ListRule, allocations.ViewRule, allocations.CreateRule, allocations.UpdateRule, allocations.DeleteRule}
			jobs := migrationCollection(t, app, "jobs")
			jobCreate, jobDelete := jobs.CreateRule, jobs.DeleteRule
			if configState == "existing" {
				loadMigrationRecord(t, app, "app_config", "migrations/client_setup_config.csv", "58lzohds2s9ho7o")
			} else if configState == "legacy only" {
				for _, id := range []string{"migjoblegacy001", "mignotlegacy001"} {
					loadMigrationRecord(t, app, "app_config", "migrations/client_setup_config.csv", id)
				}
			}
			applyMigration(t, app, setup)
			assertClientSetupStage(t, app)
			if configState == "missing" {
				if total, err := app.CountRecords("app_config"); err != nil || total != 0 {
					t.Fatalf("client setup must not create settings rows: count=%d, err=%v", total, err)
				}
			}
			if configState == "existing" {
				config, err := app.FindFirstRecordByData("app_config", "key", "jobs")
				if err != nil {
					t.Fatal(err)
				}
				var values map[string]any
				if err := json.Unmarshal([]byte(config.GetString("value")), &values); err != nil {
					t.Fatal(err)
				}
				if values["create_edit_absorb"] != false || values["custom_setting"] != "keep this" {
					t.Fatal("client setup changed unrelated job settings")
				}
			}
			allocations = migrationCollection(t, app, "job_time_allocations")
			afterRules := []*string{allocations.ListRule, allocations.ViewRule, allocations.CreateRule, allocations.UpdateRule, allocations.DeleteRule}
			if !reflect.DeepEqual(beforeRules, afterRules) {
				t.Fatal("client setup changed main allocation permissions")
			}
			jobs = migrationCollection(t, app, "jobs")
			if !reflect.DeepEqual(jobCreate, jobs.CreateRule) || !reflect.DeepEqual(jobDelete, jobs.DeleteRule) {
				t.Fatal("client setup changed main job creation or deletion permissions")
			}
			assertNoPendingMigrations(t, app, setup)
		})
	}
}

// Snapshot schema and fixture-backed business rows to check transaction rollback.
func migrationSnapshot(t *testing.T, app core.App) string {
	t.Helper()
	collections, err := app.FindAllCollections()
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(collections, func(i, j int) bool { return collections[i].Name < collections[j].Name })
	rows := map[string][]*core.Record{}
	for _, name := range []string{"app_config", "jobs", "client_invoicing_information", "notification_templates"} {
		if _, err := app.FindCollectionByNameOrId(name); err != nil {
			continue
		}
		records, err := app.FindRecordsByFilter(name, "", "id", 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		rows[name] = records
	}
	result, err := json.Marshal(struct {
		Collections []*core.Collection
		Rows        map[string][]*core.Record
	}{collections, rows})
	if err != nil {
		t.Fatal(err)
	}
	return string(result)
}

func TestClientSetupMigrationFailureRollsBackAndCanRetry(t *testing.T) {
	app, setup := newClientSetupMigrationApp(t)
	loadMigrationRecord(t, app, "app_config", "migrations/client_setup_config.csv", "58lzohds2s9ho7o")
	before := migrationSnapshot(t, app)
	failure := errors.New("force rollback after schema and configuration writes")
	failing := *setup
	failing.Up = func(txApp core.App) error {
		if err := setup.Up(txApp); err != nil {
			return err
		}
		return failure
	}
	list := core.MigrationsList{}
	list.Add(&failing)
	if _, err := core.NewMigrationsRunner(app, list).Up(); !errors.Is(err, failure) {
		t.Fatalf("expected injected failure: %v", err)
	}
	if after := migrationSnapshot(t, app); before != after {
		t.Fatal("failed migration changed schema or data")
	}
	applyMigration(t, app, setup)
	assertNoPendingMigrations(t, app, setup)
}

func fixtureRow(t *testing.T, path, id string) map[string]string {
	t.Helper()
	f, err := os.Open("../test_seed_data/" + path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, values := range rows[1:] {
		row := map[string]string{}
		for i, key := range rows[0] {
			row[key] = values[i]
		}
		if row["id"] == id {
			return row
		}
	}
	t.Fatalf("fixture %s missing %s", path, id)
	return nil
}
