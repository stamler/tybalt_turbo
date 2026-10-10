package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"tybalt/routes"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func approveMergeFixtureRequest(t *testing.T, app *tests.TestApp) {
	t.Helper()
	const id = "parequest000001"
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", "time@test.com", requestActionData(t, app, id), http.StatusOK)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/approve", workflowApprover, requestActionData(t, app, id), http.StatusOK)
}

func TestProjectMergeUndoAfterRequestLifecycle(t *testing.T) {
	for _, collection := range []string{"clients", "client_contacts"} {
		for _, outcome := range []string{"deleted", "created"} {
			t.Run(collection+"/"+outcome, func(t *testing.T) {
				app := newWorkflowTestApp(t, true)
				const requestID = "parequest000001"
				source, target, field := "pqpd90fqd5ohjcs", "lb0fnenkeyitsny", "client"
				if collection == "client_contacts" {
					source, target, field = "184qbpajciwrrp2", "pacontact000002", "contact"
				} else {
					// Set the optional owner on the existing fixture to cover both client links.
					if _, err := app.DB().NewQuery("UPDATE job_requests SET job_owner=client WHERE id='parequest000001'").Execute(); err != nil {
						t.Fatal(err)
					}
				}
				// Notes become a client note at creation, which undo must also restore.
				clientSetupJSON(t, app, http.MethodPatch, "/api/job_requests/"+requestID, "time@test.com", map[string]any{"version": 1, "request": map[string]any{"notes": "Legal description to keep with the job"}}, http.StatusOK)
				if err := routes.AbsorbRecords(app, collection, target, []string{source}); err != nil {
					t.Fatal(err)
				}
				jobID := ""
				var evidence map[string]any
				if outcome == "deleted" {
					// Age only the merged fixture to exercise the supported admin cleanup API.
					if _, err := app.DB().NewQuery("UPDATE job_requests SET last_edited='2020-01-01 00:00:00.000Z' WHERE id='parequest000001'").Execute(); err != nil {
						t.Fatal(err)
					}
					clientSetupJSON(t, app, http.MethodDelete, "/api/job_requests/"+requestID, workflowAuthor, nil, http.StatusNoContent)
				} else {
					approveMergeFixtureRequest(t, app)
					result := clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+requestID+"/create", workflowAuthor, requestActionData(t, app, requestID), http.StatusOK)
					jobID = result["job_id"].(string)
					evidence = workflowRecord(t, app, "job_attachments", "paattachment001").PublicExport()
				}
				clientSetupJSON(t, app, http.MethodPost, "/api/"+collection+"/undo_absorb", "book@keeper.com", nil, http.StatusOK)
				if _, err := app.FindRecordById("job_requests", requestID); !errors.Is(err, sql.ErrNoRows) {
					t.Fatalf("undo recreated the removed request: %v", err)
				}
				if _, err := app.FindRecordById(collection, source); err != nil {
					t.Fatalf("undo did not restore the source: %v", err)
				}
				profile := workflowRecord(t, app, "client_invoicing_information", "painvoice000001")
				if profile.GetString(field) != source {
					t.Fatal("undo did not restore the billing relation")
				}
				if jobID != "" {
					job := workflowRecord(t, app, "jobs", jobID)
					if job.GetString(field) != source || (collection == "clients" && job.GetString("job_owner") != source) {
						t.Fatal("undo did not restore the transferred job relations")
					}
					if err := utilities.ValidateClientReferences(app, job); err != nil {
						t.Fatalf("undo left invalid job relations: %v", err)
					}
					note, err := app.FindFirstRecordByData("client_notes", "job", jobID)
					if err != nil {
						t.Fatal(err)
					}
					if note.GetString("client") != job.GetString("client") {
						t.Fatalf("undo left the job note under client %s, job client %s", note.GetString("client"), job.GetString("client"))
					}
					if !reflect.DeepEqual(evidence, workflowRecord(t, app, "job_attachments", "paattachment001").PublicExport()) {
						t.Fatal("undo changed approval evidence")
					}
				}
			})
		}
	}
}

func TestProjectMergeUndoStillRejectsInvalidRemainingRecords(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	if err := routes.AbsorbRecords(app, "clients", "lb0fnenkeyitsny", []string{"pqpd90fqd5ohjcs"}); err != nil {
		t.Fatal(err)
	}
	// Corrupt only an existing fixture relation to check that undo still fails closed.
	if _, err := app.DB().NewQuery("UPDATE client_invoicing_information SET contact='235g6k01xx3sdjk' WHERE id='painvoice000001'").Execute(); err != nil {
		t.Fatal(err)
	}
	request := workflowRecord(t, app, "job_requests", "parequest000001").PublicExport()
	clientSetupJSON(t, app, http.MethodPost, "/api/clients/undo_absorb", "book@keeper.com", nil, http.StatusInternalServerError)
	if _, err := app.FindRecordById("clients", "pqpd90fqd5ohjcs"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("failed undo restored the source: %v", err)
	}
	if !reflect.DeepEqual(request, workflowRecord(t, app, "job_requests", "parequest000001").PublicExport()) {
		t.Fatal("failed undo changed request state")
	}
	if _, err := app.FindFirstRecordByData("absorb_actions", "collection_name", "clients"); err != nil {
		t.Fatal("failed undo removed its action")
	}
}

func TestProjectMergeReferenceTransferRollsBackWithCreation(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	if err := routes.AbsorbRecords(app, "clients", "lb0fnenkeyitsny", []string{"pqpd90fqd5ohjcs"}); err != nil {
		t.Fatal(err)
	}
	approveMergeFixtureRequest(t, app)
	action, err := app.FindFirstRecordByData("absorb_actions", "collection_name", "clients")
	if err != nil {
		t.Fatal(err)
	}
	before := action.GetString("updated_references")
	evidence := workflowRecord(t, app, "job_attachments", "paattachment001").PublicExport()
	jobsBefore, err := app.CountRecords("jobs")
	if err != nil {
		t.Fatal(err)
	}
	// Fail the last creation step, after the merge references have been transferred.
	app.OnRecordDelete("job_requests").BindFunc(func(e *core.RecordEvent) error {
		return fmt.Errorf("test failure after merge reference transfer")
	})
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/parequest000001/create", workflowAuthor, requestActionData(t, app, "parequest000001"), http.StatusBadRequest)
	if workflowRecord(t, app, "absorb_actions", action.Id).GetString("updated_references") != before {
		t.Fatal("failed creation changed merge references")
	}
	jobsAfter, err := app.CountRecords("jobs")
	if err != nil || jobsAfter != jobsBefore {
		t.Fatalf("failed creation retained a job: %v", err)
	}
	if workflowRecord(t, app, "job_requests", "parequest000001").GetString("status") != "Awaiting creation" || !reflect.DeepEqual(evidence, workflowRecord(t, app, "job_attachments", "paattachment001").PublicExport()) {
		t.Fatal("failed creation changed request or approval evidence")
	}
}
