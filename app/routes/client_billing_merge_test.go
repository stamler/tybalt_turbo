package routes

import (
	"net/http/httptest"
	"testing"
	"tybalt/hooks"
	"tybalt/internal/testseed"

	"github.com/pocketbase/pocketbase/core"
)

func TestClientBillingMergeAndUndo(t *testing.T) {
	for _, collection := range []string{"clients", "client_contacts"} {
		t.Run(collection, func(t *testing.T) {
			app := testseed.NewSeededTestApp(t)
			defer app.Cleanup()
			hooks.AddClientSetupHooks(app)
			source, target, field := "pqpd90fqd5ohjcs", "lb0fnenkeyitsny", "client"
			if collection == "client_contacts" {
				source, target, field = "184qbpajciwrrp2", "pacontact000002", "contact"
			}
			before, err := app.FindRecordById("client_invoicing_information", "painvoice000001")
			if err != nil {
				t.Fatal(err)
			}
			if err = AbsorbRecords(app, collection, target, []string{source}); err != nil {
				t.Fatal(err)
			}
			profile, err := app.FindRecordById("client_invoicing_information", before.Id)
			if err != nil {
				t.Fatal(err)
			}
			if profile.GetString(field) != target {
				t.Fatal("profile reference did not move")
			}
			actor, err := app.FindRecordById("users", "tqqf7q0f3378rvp")
			if err != nil {
				t.Fatal(err)
			}
			event := &core.RequestEvent{App: app, Auth: actor}
			event.Request = httptest.NewRequest("POST", "/undo", nil)
			event.Response = httptest.NewRecorder()
			if err = CreateUndoAbsorbHandler(app, collection)(event); err != nil {
				t.Fatal(err)
			}
			profile, err = app.FindRecordById("client_invoicing_information", before.Id)
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"client", "contact", "name", "creator", "created", "fax", "invoicing_instructions"} {
				if profile.GetString(name) != before.GetString(name) {
					t.Fatalf("undo changed %s", name)
				}
			}
		})
	}
}

func TestClientBillingMergeInvalidStateRollsBack(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	// Corrupt one CSV-loaded relation to check rollback when a merge discovers old bad data.
	if _, err := app.DB().NewQuery("UPDATE client_invoicing_information SET contact='235g6k01xx3sdjk' WHERE id='painvoice000001'").Execute(); err != nil {
		t.Fatal(err)
	}
	if err := AbsorbRecords(app, "clients", "lb0fnenkeyitsny", []string{"pqpd90fqd5ohjcs"}); err == nil {
		t.Fatal("merge accepted cross-client billing contact")
	}
	profile, err := app.FindRecordById("client_invoicing_information", "painvoice000001")
	if err != nil {
		t.Fatal(err)
	}
	if profile.GetString("client") != "pqpd90fqd5ohjcs" {
		t.Fatal("failed merge changed profile")
	}
	if _, err = app.FindRecordById("clients", "pqpd90fqd5ohjcs"); err != nil {
		t.Fatal("failed merge removed source client")
	}
}

func TestClientBillingUndoBlocksNewReferences(t *testing.T) {
	for _, field := range []string{"contact", "invoicing_information"} {
		t.Run(field, func(t *testing.T) {
			app := testseed.NewSeededTestApp(t)
			defer app.Cleanup()
			hooks.AddClientSetupHooks(app)
			if err := AbsorbRecords(app, "clients", "lb0fnenkeyitsny", []string{"pqpd90fqd5ohjcs"}); err != nil {
				t.Fatal(err)
			}
			// Change one existing CSV job's link to represent work saved after the merge.
			// This job already belongs to the target client, so the new link is valid now.
			job, err := app.FindRecordById("jobs", "cjf0kt0defhq480")
			if err != nil {
				t.Fatal(err)
			}
			value := "184qbpajciwrrp2"
			if field == "invoicing_information" {
				value = "painvoice000001"
			}
			job.Set(field, value)
			if err = app.SaveNoValidate(job); err != nil {
				t.Fatal(err)
			}
			actor, err := app.FindRecordById("users", "tqqf7q0f3378rvp")
			if err != nil {
				t.Fatal(err)
			}
			event := &core.RequestEvent{App: app, Auth: actor}
			event.Request = httptest.NewRequest("POST", "/undo", nil)
			event.Response = httptest.NewRecorder()
			if err = CreateUndoAbsorbHandler(app, "clients")(event); err == nil {
				t.Fatal("undo left a newer job with a cross-client reference")
			}
			profile, err := app.FindRecordById("client_invoicing_information", "painvoice000001")
			if err != nil {
				t.Fatal(err)
			}
			if profile.GetString("client") != "lb0fnenkeyitsny" {
				t.Fatal("failed undo changed the profile")
			}
			if _, err = app.FindRecordById("clients", "pqpd90fqd5ohjcs"); err == nil {
				t.Fatal("failed undo restored the source client")
			}
			if _, err = app.FindFirstRecordByData("absorb_actions", "collection_name", "clients"); err != nil {
				t.Fatal("failed undo discarded the merge record")
			}
		})
	}
}
