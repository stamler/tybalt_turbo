package routes

import (
	"github.com/pocketbase/pocketbase/core"
	"net/http/httptest"
	"testing"
	"tybalt/hooks"
	"tybalt/internal/testseed"
)

func TestProjectMergeAndUndo(t *testing.T) {
	for _, collection := range []string{"clients", "client_contacts"} {
		t.Run(collection, func(t *testing.T) {
			app := testseed.NewSeededTestApp(t)
			defer app.Cleanup()
			hooks.AddProjectWorkflowHooks(app)
			source, target, field := "pqpd90fqd5ohjcs", "lb0fnenkeyitsny", "client"
			if collection == "client_contacts" {
				source, target, field = "184qbpajciwrrp2", "pacontact000002", "contact"
			}
			before, err := app.FindRecordById("job_requests", "parequest000001")
			if err != nil {
				t.Fatal(err)
			}
			attachment, err := app.FindRecordById("job_attachments", "paattachment001")
			if err != nil {
				t.Fatal(err)
			}
			hash := attachment.GetString("document_hash")
			if err = AbsorbRecords(app, collection, target, []string{source}); err != nil {
				t.Fatal(err)
			}
			request, err := app.FindRecordById("job_requests", before.Id)
			if err != nil {
				t.Fatal(err)
			}
			if request.GetString(field) != target || request.GetInt("version") <= before.GetInt("version") {
				t.Fatal("request relation/version not changed")
			}
			profile, err := app.FindRecordById("client_invoicing_information", "painvoice000001")
			if err != nil {
				t.Fatal(err)
			}
			if profile.GetString(field) != target {
				t.Fatal("billing profile not merged")
			}
			auth, err := app.FindRecordById("users", "tqqf7q0f3378rvp")
			if err != nil {
				t.Fatal(err)
			}
			event := &core.RequestEvent{App: app, Auth: auth}
			event.Request = httptest.NewRequest("POST", "/undo", nil)
			event.Response = httptest.NewRecorder()
			if err = CreateUndoAbsorbHandler(app, collection)(event); err != nil {
				t.Fatal(err)
			}
			restored, err := app.FindRecordById("job_requests", before.Id)
			if err != nil {
				t.Fatal(err)
			}
			if restored.GetString(field) != source || restored.GetInt("version") <= request.GetInt("version") {
				t.Fatal("undo did not restore relation and invalidate review")
			}
			profile, err = app.FindRecordById("client_invoicing_information", profile.Id)
			if err != nil {
				t.Fatal(err)
			}
			if profile.GetString(field) != source {
				t.Fatal("billing profile not restored")
			}
			attachment, err = app.FindRecordById("job_attachments", attachment.Id)
			if err != nil {
				t.Fatal(err)
			}
			if attachment.GetString("document_hash") != hash {
				t.Fatal("merge changed evidence")
			}
		})
	}
}

func TestProjectMergeRollsBackInvalidFinalRelations(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	// Change only a fixture relation to test rollback of an invalid final state.
	if _, err := app.DB().NewQuery("UPDATE client_invoicing_information SET contact='235g6k01xx3sdjk' WHERE id='painvoice000001'").Execute(); err != nil {
		t.Fatal(err)
	}
	if err := AbsorbRecords(app, "clients", "lb0fnenkeyitsny", []string{"pqpd90fqd5ohjcs"}); err == nil {
		t.Fatal("merge accepted an invalid billing client")
	}
	request, err := app.FindRecordById("job_requests", "parequest000001")
	if err != nil {
		t.Fatal(err)
	}
	if request.GetString("client") != "pqpd90fqd5ohjcs" || request.GetInt("version") != 1 {
		t.Fatal("failed merge changed request")
	}
	if _, err = app.FindRecordById("clients", "pqpd90fqd5ohjcs"); err != nil {
		t.Fatal("failed merge deleted source client")
	}
}
