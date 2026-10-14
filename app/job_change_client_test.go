package main

import (
	"net/http"
	"strings"
	"testing"

	"tybalt/hooks"

	"github.com/pocketbase/pocketbase/tests"
)

// =============================================================================
// Change Client Endpoint Tests
// =============================================================================
//
// POST /api/jobs/{id}/change-client moves a project and its subjobs to another
// client. The tests move the legacy workflow project 26-9876 from Sepulchi & Co
// (pqpd90fqd5ohjcs) to Mobilia SA (lb0fnenkeyitsny), whose contact
// nh5u9z3cyknjclv is billed through profile painvoice000002. Horace Silver
// (author@soup.com) holds job and manages the project.

const (
	clientChangeJob       = legacyWorkflowJob
	clientChangeSubjob    = "ccsubjob0000001"
	clientChangeOldClient = "pqpd90fqd5ohjcs"
	clientChangeNewClient = "lb0fnenkeyitsny"
	clientChangeContact   = "nh5u9z3cyknjclv"
	clientChangeProfile   = "painvoice000002"
	clientChangeReason    = "The project was entered under the architect instead of the owner."
)

func clientChangeBody(overrides map[string]any) map[string]any {
	body := map[string]any{
		"client":                clientChangeNewClient,
		"contact":               clientChangeContact,
		"invoicing_information": clientChangeProfile,
		"reason":                clientChangeReason,
		"subjobs":               []string{clientChangeSubjob},
	}
	for k, v := range overrides {
		body[k] = v
	}
	return body
}

// postClientChange sends a change and returns the response body.
func postClientChange(t *testing.T, app *tests.TestApp, job, email string, body map[string]any, status int) string {
	t.Helper()
	response := performJSONRequest(t, app, http.MethodPost, "/api/jobs/"+job+"/change-client", workflowToken(t, app, email), body)
	mustStatus(t, response, status)
	return response.Body.String()
}

func execSQL(t *testing.T, app *tests.TestApp, queries ...string) {
	t.Helper()
	for _, query := range queries {
		if _, err := app.DB().NewQuery(query).Execute(); err != nil {
			t.Fatal(err)
		}
	}
}

// newClientChangeApp adds a Closed subjob managed by Tester Time and a client
// note about the project.
func newClientChangeApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app := newWorkflowTestApp(t, false)
	execSQL(t, app,
		"INSERT INTO jobs (id, number, parent, client, contact, invoicing_information, manager, status, description) VALUES ('"+clientChangeSubjob+"', '26-9876-01', '"+clientChangeJob+"', '"+clientChangeOldClient+"', '184qbpajciwrrp2', 'painvoice000001', 'rzr98oadsp9qc11', 'Closed', 'Subjob of the legacy fixture')",
		"INSERT INTO client_notes (id, client, job, uid, note) VALUES ('ccnote000000001', '"+clientChangeOldClient+"', '"+clientChangeJob+"', 'f2j5a8vk006baub', 'Called the client about the site survey.')",
	)
	return app
}

func TestChangeJobClientMovesProjectSubjobsAndNotes(t *testing.T) {
	app := newClientChangeApp(t)
	// Tier Two approves a value for the project while it belongs to Sepulchi & Co.
	value := createValueChange(t, app, clientChangeJob, 10000, "Value before the move")
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+value+"/approve", workflowApprover, attachmentActionData(t, app, value), http.StatusOK)
	before := len(app.TestMailer.Messages())

	postClientChange(t, app, clientChangeJob, horaceEmail, clientChangeBody(nil), http.StatusOK)

	for _, id := range []string{clientChangeJob, clientChangeSubjob} {
		job := workflowRecord(t, app, "jobs", id)
		if job.GetString("client") != clientChangeNewClient || job.GetString("contact") != clientChangeContact || job.GetString("invoicing_information") != clientChangeProfile {
			t.Errorf("job %s has client %q, contact %q, profile %q", id, job.GetString("client"), job.GetString("contact"), job.GetString("invoicing_information"))
		}
	}
	if note := workflowRecord(t, app, "client_notes", "ccnote000000001"); note.GetString("client") != clientChangeNewClient {
		t.Errorf("the project's note stayed with client %q", note.GetString("client"))
	}
	if a := workflowRecord(t, app, "job_attachments", value); a.GetString("status") != "Approved" || a.GetString("job") != clientChangeJob {
		t.Error("the approved value change was altered")
	}

	moved := " Subjobs moved with it: 26-9876-01. Reason: " + clientChangeReason
	if _, err := app.FindFirstRecordByData("client_notes", "note", "Client changed from Sepulchi & Co to Mobilia SA."+moved); err != nil {
		t.Error("no note under the new client records the change")
	} else if note, _ := app.FindFirstRecordByData("client_notes", "note", "Client changed from Sepulchi & Co to Mobilia SA."+moved); note.GetString("client") != clientChangeNewClient || note.GetString("job") != clientChangeJob {
		t.Error("the change note is not on the project under the new client")
	}
	departure, err := app.FindFirstRecordByData("client_notes", "note", "Project 26-9876 moved to Mobilia SA."+moved)
	if err != nil {
		t.Fatal("no note under the old client records where the project went")
	}
	if departure.GetString("client") != clientChangeOldClient || departure.GetString("job") != "" || !departure.GetBool("job_not_applicable") {
		t.Error("the departure note is not a general note under the old client")
	}

	text := func(name, role string) string {
		return `Hello ` + name + `,

Horace Silver changed the client of project 26-9876 from Sepulchi & Co to Mobilia SA. Its subjobs moved with it: 26-9876-01.

Reason:
` + clientChangeReason + `

Project / scope of work:
Legacy zero value workflow fixture

` + role + ` The project now uses the contact and invoicing profile chosen for Mobilia SA. View the project here:

%s`
	}
	path := "/jobs/" + clientChangeJob + "/details"
	// Horace manages the project but made the change, so he is not emailed.
	expectWorkflowEmails(t, app, before,
		workflowEmail{tierTwoEmail, "Client changed: project 26-9876", text("Tier Two", "You approved a value for this project when its client was Sepulchi & Co."), path},
		workflowEmail{testerTimeEmail, "Client changed: project 26-9876", text("Tester Time", "You manage this project or one of its subjobs."), path},
	)
}

func TestChangeJobClientRefusals(t *testing.T) {
	app := newClientChangeApp(t)
	cases := []struct {
		name, job, email string
		body             map[string]any
		status           int
		want             string
	}{
		{"user without job", clientChangeJob, "noclaims@example.com", clientChangeBody(nil), http.StatusForbidden, "job claim"},
		{"proposal", "fcpropimpinprog", horaceEmail, clientChangeBody(nil), http.StatusBadRequest, "proposal"},
		{"subjob", clientChangeSubjob, horaceEmail, clientChangeBody(nil), http.StatusBadRequest, "parent project"},
		{"same client", clientChangeJob, horaceEmail, clientChangeBody(map[string]any{"client": clientChangeOldClient, "contact": "184qbpajciwrrp2", "invoicing_information": "painvoice000001"}), http.StatusBadRequest, `"unchanged"`},
		{"short reason", clientChangeJob, horaceEmail, clientChangeBody(map[string]any{"reason": "  wrong client     "}), http.StatusBadRequest, `"reason":{"code":"too_short"`},
		{"long reason", clientChangeJob, horaceEmail, clientChangeBody(map[string]any{"reason": strings.Repeat("x", 501)}), http.StatusBadRequest, `"reason":{"code":"too_long"`},
		{"missing selections", clientChangeJob, horaceEmail, clientChangeBody(map[string]any{"contact": "", "invoicing_information": ""}), http.StatusBadRequest, `"invoicing_information":{"code":"required"`},
		{"contact of the old client", clientChangeJob, horaceEmail, clientChangeBody(map[string]any{"contact": "184qbpajciwrrp2"}), http.StatusBadRequest, "contact"},
		{"profile of the old client", clientChangeJob, horaceEmail, clientChangeBody(map[string]any{"invoicing_information": "painvoice000001"}), http.StatusBadRequest, "invoicing_information"},
		{"unknown client", clientChangeJob, horaceEmail, clientChangeBody(map[string]any{"client": "nosuchclient000"}), http.StatusBadRequest, "not found"},
		{"subjobs not confirmed", clientChangeJob, horaceEmail, clientChangeBody(map[string]any{"subjobs": []string{}}), http.StatusConflict, `"subjobs":{"code":"unconfirmed"`},
		{"confirmation missing", clientChangeJob, horaceEmail, clientChangeBody(map[string]any{"subjobs": nil}), http.StatusConflict, `"subjobs":{"code":"unconfirmed"`},
		{"other subjobs confirmed", clientChangeJob, horaceEmail, clientChangeBody(map[string]any{"subjobs": []string{"testsubjob01id"}}), http.StatusConflict, `"subjobs":{"code":"unconfirmed"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if body := postClientChange(t, app, c.job, c.email, c.body, c.status); !strings.Contains(body, c.want) {
				t.Errorf("response %s does not mention %s", body, c.want)
			}
		})
	}
	for _, id := range []string{clientChangeJob, clientChangeSubjob} {
		if workflowRecord(t, app, "jobs", id).GetString("client") != clientChangeOldClient {
			t.Errorf("a refused change moved job %s", id)
		}
	}
	if workflowRecord(t, app, "client_notes", "ccnote000000001").GetString("client") != clientChangeOldClient {
		t.Error("a refused change moved the project's note")
	}
}

// Work for the old client that is still in progress blocks the change.
func TestChangeJobClientWaitsForOpenWork(t *testing.T) {
	t.Run("value change awaiting approval", func(t *testing.T) {
		app := newClientChangeApp(t)
		createValueChange(t, app, clientChangeJob, 10000, "Pending value")
		// Only an admin can delete a value change, so the message says so.
		if body := postClientChange(t, app, clientChangeJob, horaceEmail, clientChangeBody(nil), http.StatusConflict); !strings.Contains(body, "ask an admin") {
			t.Errorf("response %s does not say who can delete the value change", body)
		}
	})
	t.Run("open subjob request", func(t *testing.T) {
		app := newClientChangeApp(t)
		execSQL(t, app, "INSERT INTO job_requests (id, parent, client, creator, status) VALUES ('ccrequest000001', '"+clientChangeJob+"', '"+clientChangeOldClient+"', 'f2j5a8vk006baub', 'Draft')")
		postClientChange(t, app, clientChangeJob, horaceEmail, clientChangeBody(nil), http.StatusConflict)
	})
	t.Run("merge that can still be undone", func(t *testing.T) {
		app := newClientChangeApp(t)
		execSQL(t, app, `INSERT INTO absorb_actions (id, collection_name, target_id, absorbed_records, updated_references) VALUES ('ccabsorb0000001', 'clients', '`+clientChangeOldClient+`', '[]', '{"client_notes":{"client":{"ccnote000000001":"absorbedclient1"}}}')`)
		postClientChange(t, app, clientChangeJob, horaceEmail, clientChangeBody(nil), http.StatusConflict)
	})
	// Undo would move the chosen contact back to the absorbed client, leaving
	// the project pointing at another client's contact.
	t.Run("merge that moved the chosen contact", func(t *testing.T) {
		app := newClientChangeApp(t)
		execSQL(t, app, `INSERT INTO absorb_actions (id, collection_name, target_id, absorbed_records, updated_references) VALUES ('ccabsorb0000002', 'clients', '`+clientChangeNewClient+`', '[]', '{"client_contacts":{"client":{"`+clientChangeContact+`":"absorbedclient1"}}}')`)
		postClientChange(t, app, clientChangeJob, horaceEmail, clientChangeBody(nil), http.StatusConflict)
	})
}

// The notes quote the reason with both client names and every subjob, so a
// reason that fits on its own can still make them too long. The user is told
// how much to shorten it.
func TestChangeJobClientKeepsNotesWithinTheirLimit(t *testing.T) {
	app := newClientChangeApp(t)
	execSQL(t, app, "UPDATE clients SET alias = '"+strings.Repeat("a", 480)+"' WHERE id = '"+clientChangeNewClient+"'")
	body := postClientChange(t, app, clientChangeJob, horaceEmail, clientChangeBody(map[string]any{"reason": strings.Repeat("r", 480)}), http.StatusBadRequest)
	if !strings.Contains(body, `"reason":{"code":"too_long"`) || !strings.Contains(body, "Shorten the reason by") {
		t.Fatalf("response %s does not ask for a shorter reason", body)
	}
	if workflowRecord(t, app, "jobs", clientChangeJob).GetString("client") != clientChangeOldClient {
		t.Error("the refused change moved the project")
	}
}

// People who have left are not emailed.
func TestChangeJobClientSkipsInactiveRecipients(t *testing.T) {
	app := newClientChangeApp(t)
	execSQL(t, app, "UPDATE jobs SET manager = 'u_inactive' WHERE id = '"+clientChangeSubjob+"'")
	before := len(app.TestMailer.Messages())
	postClientChange(t, app, clientChangeJob, horaceEmail, clientChangeBody(nil), http.StatusOK)
	// Horace manages the project and made the change; the subjob's manager has left.
	expectWorkflowEmails(t, app, before)
}

// Outside the change client action and merges, a project's client stays fixed.
func TestProjectClientStaysFixedOutsideTheChangeAction(t *testing.T) {
	app := newClientChangeApp(t)
	job := workflowRecord(t, app, "jobs", clientChangeJob)
	job.Set("client", clientChangeNewClient)
	job.Set("contact", clientChangeContact)
	job.Set("invoicing_information", clientChangeProfile)
	if err := app.Save(job); err == nil {
		t.Fatal("a plain save changed the project's client")
	}
	ctx := hooks.WithProjectWorkflowMutation(t.Context(), hooks.ProjectWorkflowMutationValue)
	if err := app.SaveWithContext(ctx, job); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("another workflow action changed the project's client: %v", err)
	}
	// Full project validation accepts a client change only from a merge.
	actor, err := app.FindAuthRecordByEmail("users", horaceEmail)
	if err != nil {
		t.Fatal(err)
	}
	if err := hooks.ProcessJobWorkflow(app, job, actor, hooks.ProjectWorkflowMutationChangeClient); err == nil || !strings.Contains(err.Error(), "client merge") {
		t.Fatalf("project validation accepted a client change outside a merge: %v", err)
	}
}

// A subjob added after the user confirmed the list is not moved without
// being confirmed, and a project without subjobs needs no confirmation.
func TestChangeJobClientConfirmsTheSubjobsItMoves(t *testing.T) {
	app := newClientChangeApp(t)
	execSQL(t, app, "INSERT INTO jobs (id, number, parent, client, contact, invoicing_information, manager, status, description) VALUES ('ccsubjob0000002', '26-9876-02', '"+clientChangeJob+"', '"+clientChangeOldClient+"', '184qbpajciwrrp2', 'painvoice000001', 'f2j5a8vk006baub', 'Active', 'Added after the dialog opened')")
	postClientChange(t, app, clientChangeJob, horaceEmail, clientChangeBody(nil), http.StatusConflict)

	execSQL(t, app, "DELETE FROM jobs WHERE parent = '"+clientChangeJob+"'")
	postClientChange(t, app, clientChangeJob, horaceEmail, clientChangeBody(map[string]any{"subjobs": []string{}}), http.StatusOK)
	if workflowRecord(t, app, "jobs", clientChangeJob).GetString("client") != clientChangeNewClient {
		t.Error("a project without subjobs did not move")
	}
}
