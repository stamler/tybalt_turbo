package hooks

import (
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const (
	testJobID     = "cjf0kt0defhq480"
	testClientID  = "lb0fnenkeyitsny"
	testContactID = "nh5u9z3cyknjclv"
)

func setJobImported(t *testing.T, app *tests.TestApp, value bool) {
	t.Helper()
	if _, err := app.DB().NewQuery(`
		UPDATE jobs
		SET _imported = {:value}
		WHERE id = {:id}
	`).Bind(dbx.Params{"value": value, "id": testJobID}).Execute(); err != nil {
		t.Fatalf("failed to set job import flag: %v", err)
	}
}

func assertJobNotImported(t *testing.T, app *tests.TestApp) {
	t.Helper()
	job, err := app.FindRecordById("jobs", testJobID)
	if err != nil {
		t.Fatalf("failed to load job: %v", err)
	}
	if job.GetBool("_imported") {
		t.Fatal("expected job to be marked as not imported")
	}
}

func TestCategoriesCreateMarksJobNotImported(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatalf("failed to init test app: %v", err)
	}
	defer app.Cleanup()
	AddHooks(app)

	setJobImported(t, app, true)

	categoriesCollection, err := app.FindCollectionByNameOrId("categories")
	if err != nil {
		t.Fatalf("failed to load categories collection: %v", err)
	}
	category := core.NewRecord(categoriesCollection)
	category.Set("job", testJobID)
	category.Set("name", "hook-create-test-category")

	if err := app.Save(category); err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	assertJobNotImported(t, app)
}

func TestCategoriesDeleteMarksJobNotImported(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatalf("failed to init test app: %v", err)
	}
	defer app.Cleanup()
	AddHooks(app)

	categoriesCollection, err := app.FindCollectionByNameOrId("categories")
	if err != nil {
		t.Fatalf("failed to load categories collection: %v", err)
	}
	category := core.NewRecord(categoriesCollection)
	category.Set("job", testJobID)
	category.Set("name", "hook-delete-test-category")
	if err := app.Save(category); err != nil {
		t.Fatalf("failed to create category fixture: %v", err)
	}

	setJobImported(t, app, true)

	if err := app.Delete(category); err != nil {
		t.Fatalf("failed to delete category: %v", err)
	}

	assertJobNotImported(t, app)
}

func TestClientUpdateMarksReferencingJobsNotImported(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatalf("failed to init test app: %v", err)
	}
	defer app.Cleanup()
	AddHooks(app)

	setJobImported(t, app, true)

	client, err := app.FindRecordById("clients", testClientID)
	if err != nil {
		t.Fatalf("failed to load client: %v", err)
	}
	client.Set("name", client.GetString("name")+" [hook-test]")

	if err := app.Save(client); err != nil {
		t.Fatalf("failed to update client: %v", err)
	}

	assertJobNotImported(t, app)
}

func TestContactUpdateMarksReferencingJobsNotImported(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatalf("failed to init test app: %v", err)
	}
	defer app.Cleanup()
	AddHooks(app)

	setJobImported(t, app, true)

	contact, err := app.FindRecordById("client_contacts", testContactID)
	if err != nil {
		t.Fatalf("failed to load client contact: %v", err)
	}
	contact.Set("surname", contact.GetString("surname")+"-hook")

	if err := app.Save(contact); err != nil {
		t.Fatalf("failed to update client contact: %v", err)
	}

	assertJobNotImported(t, app)
}
