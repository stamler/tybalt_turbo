package migrations_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	_ "tybalt/migrations"
)

const requestWorkflowMigration = "1791560769_project_request_workflow.go"

func newSplitMigrationApp(t *testing.T) (core.App, *core.Migration, *core.Migration) {
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
	var setup, workflow *core.Migration
	for _, migration := range core.AppMigrations.Items() {
		switch migration.File {
		case clientSetupMigration:
			setup = migration
		case requestWorkflowMigration:
			workflow = migration
		default:
			if migration.File < clientSetupMigration {
				previous.Add(migration)
			}
		}
	}
	if setup == nil || workflow == nil {
		t.Fatal("both migration stages must be registered")
	}
	if _, err := core.NewMigrationsRunner(app, previous).Up(); err != nil {
		t.Fatal(err)
	}
	return app, setup, workflow
}

func assertProfilePreserved(t *testing.T, app core.App, before *core.Record) {
	t.Helper()
	after, err := app.FindRecordById("client_invoicing_information", before.Id)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"client", "contact", "name", "fax", "invoicing_instructions", "creator", "created"} {
		if before.GetString(field) != after.GetString(field) {
			t.Errorf("workflow stage changed profile %s: %q -> %q", field, before.GetString(field), after.GetString(field))
		}
	}
}

func TestProjectRequestMigrationPreservesBusinessData(t *testing.T) {
	for _, scenario := range []string{"empty", "populated"} {
		t.Run(scenario+" evidence", func(t *testing.T) {
			app, setup, workflow := newSplitMigrationApp(t)
			jobs, err := app.FindCollectionByNameOrId("jobs")
			if err != nil {
				t.Fatal(err)
			}
			jobs.Fields.Add(&core.TextField{Name: "future_business_field"})
			jobs.AddIndex("future_business_index", false, "description", "")
			if err = app.Save(jobs); err != nil {
				t.Fatal(err)
			}
			// Migration fixtures use retired fields, so they stay outside datapackage.json.
			// Their job IDs select test cases; the canonical job ID stays unchanged.
			job := core.NewRecord(jobs)
			for k, v := range fixtureRow(t, "data/jobs.csv", "cjf0kt0defhq480") {
				if jobs.Fields.GetByName(k) != nil {
					job.Set(k, v)
				}
			}
			for k, v := range fixtureRow(t, "migrations/project_request_jobs.csv", scenario) {
				if k != "id" {
					job.Set(k, v)
				}
			}
			if err = app.SaveNoValidate(job); err != nil {
				t.Fatal(err)
			}
			branches, err := app.FindCollectionByNameOrId("branches")
			if err != nil {
				t.Fatal(err)
			}
			branch := core.NewRecord(branches)
			for k, v := range fixtureRow(t, "data/branches.csv", "80875lm27v8wgi4") {
				branch.Set(k, v)
			}
			if err = app.SaveNoValidate(branch); err != nil {
				t.Fatal(err)
			}

			// The old configuration fixture includes both retired and unrelated flags.
			configCollection, err := app.FindCollectionByNameOrId("app_config")
			if err != nil {
				t.Fatal(err)
			}
			// Cover both creation of a missing settings row and replacement of an
			// existing legacy description without changing unrelated settings.
			if scenario == "populated" {
				legacyConfig := core.NewRecord(configCollection)
				for k, v := range fixtureRow(t, "migrations/project_request_config.csv", "58lzohds2s9ho7o") {
					legacyConfig.Set(k, v)
				}
				if err = app.Save(legacyConfig); err != nil {
					t.Fatal(err)
				}
			}
			notificationConfig := core.NewRecord(configCollection)
			for k, v := range fixtureRow(t, "data/app_config.csv", "030887mb4spir3z") {
				notificationConfig.Set(k, v)
			}
			notificationConfig.Set("value", fixtureRow(t, "migrations/project_request_config.csv", notificationConfig.Id)["value"])
			if err = app.SaveNoValidate(notificationConfig); err != nil {
				t.Fatal(err)
			}
			applyMigration(t, app, setup)
			assertClientSetupStage(t, app)
			// A profile entered during the first release must survive the second.
			profile := loadMigrationRecord(t, app, "client_invoicing_information", "migrations/client_setup_profiles.csv", "migprofile00001")
			job, err = app.FindRecordById("jobs", job.Id)
			if err != nil {
				t.Fatal(err)
			}
			job.Set("invoicing_information", profile.Id)
			if err = app.SaveNoValidate(job); err != nil {
				t.Fatal(err)
			}
			applyMigration(t, app, workflow)
			assertProfilePreserved(t, app, profile)
			contacts := migrationCollection(t, app, "client_contacts")
			if contacts.Fields.GetByName("email").(*core.EmailField).Required {
				t.Fatal("workflow stage must keep contact email optional")
			}
			if _, ok := contacts.Fields.GetByName("no_email").(*core.BoolField); !ok {
				t.Fatal("workflow stage must add the contact no-email mark")
			}
			assertNoPendingMigrations(t, app, setup, workflow)
			allocations, err := app.FindCollectionByNameOrId("job_time_allocations")
			if err != nil {
				t.Fatal(err)
			}
			if allocations.CreateRule != nil || allocations.UpdateRule != nil || allocations.DeleteRule != nil {
				t.Fatal("generic allocation writes remain enabled")
			}
			if allocations.ListRule == nil || allocations.ViewRule == nil {
				t.Fatal("allocation read access was removed")
			}
			jobs, err = app.FindCollectionByNameOrId("jobs")
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{"project_authorization_doc", "project_authorization_doc_hash", "pa_reviewer", "pa_reviewed", "pa_uploader", "pa_uploaded", "pa_rejector", "pa_rejected", "pa_rejection_reason"} {
				if jobs.Fields.GetByName(field) != nil {
					t.Errorf("retired field remains: %s", field)
				}
			}
			if jobs.Fields.GetByName("future_business_field") == nil {
				t.Fatal("unrelated field removed")
			}
			if !strings.Contains(strings.Join(jobs.Indexes, " "), "future_business_index") {
				t.Fatal("unrelated index removed")
			}
			job, err = app.FindRecordById("jobs", job.Id)
			if err != nil {
				t.Fatal(err)
			}
			if job.GetInt("project_value") != 100000 || job.GetString("future_business_field") != "preserve this field" || job.GetString("invoicing_information") != profile.Id {
				t.Fatal("business data changed")
			}
			gotBranch, err := app.FindRecordById("branches", branch.Id)
			if err != nil {
				t.Fatal(err)
			}
			if gotBranch.GetString("manager") != branch.GetString("manager") {
				t.Fatal("branch manager changed")
			}
			count, err := app.CountRecords("job_attachments")
			if err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatal("old evidence was copied")
			}
			requests, err := app.FindCollectionByNameOrId("job_requests")
			if err != nil {
				t.Fatal(err)
			}
			// The workflow stage must install the complete request schema.
			// It must allow empty drafts and must not auto-issue a token on save.
			submission, ok := requests.Fields.GetByName("submission_token").(*core.TextField)
			if !ok || submission.Required || submission.Max != 32 || submission.Pattern != "^[a-z0-9]{32}$" || submission.AutogeneratePattern != "" {
				t.Fatalf("invalid submission_token schema: %#v", submission)
			}
			config, err := app.FindFirstRecordByData("app_config", "key", "jobs")
			if err != nil {
				t.Fatal(err)
			}
			var value map[string]any
			if err = json.Unmarshal([]byte(config.GetString("value")), &value); err != nil {
				t.Fatal(err)
			}
			if _, ok := value["enforce_project_authorization"]; ok {
				t.Fatal("old flag remains")
			}
			if value["require_job_request_approval"] != false {
				t.Fatal("request flag must default off")
			}
			if config.GetString("description") != fixtureRow(t, "data/app_config.csv", "58lzohds2s9ho7o")["description"] {
				t.Fatal("jobs configuration description does not match the current settings")
			}
			if scenario == "populated" && (value["create_edit_absorb"] != false || value["custom_setting"] != "keep this") {
				t.Fatal("updating the jobs description changed unrelated settings")
			}
			notificationConfig, err = app.FindFirstRecordByData("app_config", "key", "notifications")
			if err != nil {
				t.Fatal(err)
			}
			var notificationFlags map[string]any
			if err = json.Unmarshal([]byte(notificationConfig.GetString("value")), &notificationFlags); err != nil {
				t.Fatal(err)
			}
			if _, exists := notificationFlags["project_authorization_rejected"]; exists {
				t.Fatal("retired notification toggle remains")
			}
			if notificationFlags["po_active"] != false {
				t.Fatal("unrelated notification toggle changed")
			}
			for _, key := range []string{"job_request_workflow", "job_value_workflow"} {
				if notificationFlags[key] != true {
					t.Fatalf("required notice is disabled: %s", key)
				}
			}

		})
	}
}

// All migrations, in release order, must produce the canonical seed schema.
// Later main migrations run between the two project stages.
func TestProjectMigrationsMatchCanonicalSchema(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.ResetBootstrapState() })
	if err := app.RunAllMigrations(); err != nil {
		t.Fatal(err)
	}
	if clientSetupMigration > requestWorkflowMigration {
		t.Fatal("the request workflow migration must run after client setup")
	}
	assertCanonicalCollections(t, app, "clients", "client_contacts", "jobs", "client_invoicing_information",
		"job_requests", "job_request_allocations", "job_attachments", "project_authorization_props", "job_time_allocations")
	assertCanonicalUserViewGrants(t, app)
}
