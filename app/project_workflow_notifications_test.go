package main

import (
	"fmt"
	"net/http"
	"testing"

	"tybalt/notifications"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

const (
	tierTwoEmail     = "tier2@poapprover.com" // Tier Two, the selected approver
	testerTimeEmail  = "time@test.com"        // Tester Time, the request creator
	horaceEmail      = "author@soup.com"      // Horace Silver, a job holder
	workflowClientUI = "Sepulchi & Co"
)

// workflowEmail is one expected message: its recipient, subject, and full text
// with %s standing for the action link.
type workflowEmail struct {
	to, subject, text, path string
}

// expectWorkflowEmails checks that exactly the expected messages were sent
// after the first `before` messages, so no one else, including the actor, was
// emailed.
func expectWorkflowEmails(t *testing.T, app *tests.TestApp, before int, want ...workflowEmail) {
	t.Helper()
	sent := app.TestMailer.Messages()[before:]
	if len(sent) != len(want) {
		t.Fatalf("sent %d emails, want %d: %v", len(sent), len(want), describeEmails(sent))
	}
	for _, expected := range want {
		var message *mailer.Message
		for _, m := range sent {
			if len(m.To) == 1 && m.To[0].Address == expected.to {
				message = m
			}
		}
		if message == nil {
			t.Fatalf("no email to %s: %v", expected.to, describeEmails(sent))
		}
		if message.Subject != expected.subject {
			t.Errorf("subject to %s = %q; want %q", expected.to, message.Subject, expected.subject)
		}
		text := fmt.Sprintf(expected.text, notifications.BuildActionURL(app, expected.path))
		if message.Text != text {
			t.Errorf("text to %s =\n%s\n\nwant\n%s", expected.to, message.Text, text)
		}
	}
}

func describeEmails(messages []*mailer.Message) []string {
	result := []string{}
	for _, m := range messages {
		result = append(result, m.To[0].Address+": "+m.Subject)
	}
	return result
}

// grantTesterTimeApproval gives Tester Time an approval limit of $1,000,000.
func grantTesterTimeApproval(t *testing.T, app *tests.TestApp) {
	t.Helper()
	for _, query := range []string{
		"INSERT INTO user_claims (id, cid, uid, _imported, created, updated) VALUES ('paucselfappr001', 'projectauth0001', 'rzr98oadsp9qc11', 0, '2026-10-13 00:00:00.000Z', '2026-10-13 00:00:00.000Z')",
		"INSERT INTO project_authorization_props (id, user_claim, max_value, created, updated) VALUES ('papropsselfapr1', 'paucselfappr001', 1000000, '2026-10-13 00:00:00.000Z', '2026-10-13 00:00:00.000Z')",
	} {
		if _, err := app.DB().NewQuery(query).Execute(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestProjectRequestEmails(t *testing.T) {
	const id = "parequest000001"
	const scope = "Request workflow acceptance project"
	const subject = "project request for Sepulchi & Co — " + scope
	requestLink := "/job-requests/" + id
	submitted := func(revised string) workflowEmail {
		return workflowEmail{tierTwoEmail, "Approval needed: " + subject, `Hello Tier Two,

Tester Time has asked you to approve a project request for Sepulchi & Co with a project value of $100,000.` + revised + `

Project / scope of work:
` + scope + `

Review the request and its authorization PDF, then approve or reject it here:

%s`, requestLink}
	}
	rejected := func(verb, withdrawn string) workflowEmail {
		return workflowEmail{testerTimeEmail, "Rejected: " + subject, `Hello Tester Time,

Tier Two ` + verb + ` your project request for Sepulchi & Co for the following reason:

Revise the signed scope.

Project / scope of work:
` + scope + `

` + withdrawn + `You can correct the request and submit it again here:

%s`, requestLink}
	}
	reject := func(t *testing.T, app *tests.TestApp) {
		t.Helper()
		rejection := requestActionData(t, app, id)
		rejection["rejection_reason"] = "Revise the signed scope."
		clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/reject", workflowApprover, rejection, http.StatusOK)
	}
	created := func(t *testing.T, app *tests.TestApp, closed bool) (*core.Record, []workflowEmail) {
		t.Helper()
		result := clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/create", horaceEmail, requestActionData(t, app, id), http.StatusOK)
		job := workflowRecord(t, app, "jobs", result["job_id"].(string))
		number := job.GetString("number")
		usable := "Time, expenses, and purchase orders can now be recorded against " + number + "."
		if closed {
			usable = "The project is Closed, so time, expenses, and purchase orders cannot be recorded against it."
		}
		email := func(to, name, from string) workflowEmail {
			return workflowEmail{to, "Project " + number + " created for Sepulchi & Co", `Hello ` + name + `,

Horace Silver created project ` + number + ` for Sepulchi & Co from ` + from + `.

Project / scope of work:
` + scope + `

` + usable + ` View the project here:

%s`, "/jobs/" + job.Id + "/details"}
		}
		return job, []workflowEmail{
			email(testerTimeEmail, "Tester Time", "your project request"),
			email(tierTwoEmail, "Tier Two", "a project request by Tester Time. You are its project manager"),
		}
	}

	t.Run("approved and created", func(t *testing.T) {
		app := newWorkflowTestApp(t, true)
		// Make the approver the project manager so creation also emails a manager
		// who is not the requester. Horace Silver, the original manager, creates it.
		clientSetupJSON(t, app, http.MethodPatch, "/api/job_requests/"+id, testerTimeEmail, map[string]any{"version": 1, "request": map[string]any{"manager": "6bq4j0eb26631dy"}}, http.StatusOK)

		before := len(app.TestMailer.Messages())
		clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", testerTimeEmail, requestActionData(t, app, id), http.StatusOK)
		expectWorkflowEmails(t, app, before, submitted(""))

		before = len(app.TestMailer.Messages())
		clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/approve", workflowApprover, requestActionData(t, app, id), http.StatusOK)
		expectWorkflowEmails(t, app, before, workflowEmail{testerTimeEmail, "Approved: " + subject, `Hello Tester Time,

Tier Two approved your project request for Sepulchi & Co with a project value of $100,000.

Project / scope of work:
` + scope + `

The project will now be created and given a number. You will receive another email with the project number when that happens. View the request here:

%s`, requestLink})

		before = len(app.TestMailer.Messages())
		_, emails := created(t, app, false)
		expectWorkflowEmails(t, app, before, emails...)
	})
	t.Run("created closed", func(t *testing.T) {
		app := newWorkflowTestApp(t, true)
		clientSetupJSON(t, app, http.MethodPatch, "/api/job_requests/"+id, testerTimeEmail, map[string]any{"version": 1, "request": map[string]any{"manager": "6bq4j0eb26631dy", "project_status": "Closed"}}, http.StatusOK)
		clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", testerTimeEmail, requestActionData(t, app, id), http.StatusOK)
		clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/approve", workflowApprover, requestActionData(t, app, id), http.StatusOK)
		before := len(app.TestMailer.Messages())
		job, emails := created(t, app, true)
		if job.GetString("status") != "Closed" {
			t.Fatalf("created status %s; want Closed", job.GetString("status"))
		}
		expectWorkflowEmails(t, app, before, emails...)
	})
	t.Run("rejected and resubmitted", func(t *testing.T) {
		app := newWorkflowTestApp(t, true)
		clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", testerTimeEmail, requestActionData(t, app, id), http.StatusOK)
		before := len(app.TestMailer.Messages())
		reject(t, app)
		expectWorkflowEmails(t, app, before, rejected("rejected", ""))

		// The approver is told the corrected request replaces the one they were sent.
		version := workflowRecord(t, app, "job_requests", id).GetInt("version")
		clientSetupJSON(t, app, http.MethodPatch, "/api/job_requests/"+id, testerTimeEmail, map[string]any{"version": version, "request": map[string]any{"client_po": "PO-77"}}, http.StatusOK)
		before = len(app.TestMailer.Messages())
		clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", testerTimeEmail, requestActionData(t, app, id), http.StatusOK)
		expectWorkflowEmails(t, app, before, submitted(" This replaces a version of this request sent to you earlier."))
	})
	t.Run("approval withdrawn", func(t *testing.T) {
		app := newWorkflowTestApp(t, true)
		clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", testerTimeEmail, requestActionData(t, app, id), http.StatusOK)
		clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/approve", workflowApprover, requestActionData(t, app, id), http.StatusOK)
		before := len(app.TestMailer.Messages())
		reject(t, app)
		expectWorkflowEmails(t, app, before, rejected("withdrew approval of", "The project will not be created. "))
	})
	t.Run("own approval sends nothing", func(t *testing.T) {
		app := newWorkflowTestApp(t, true)
		// The requester approves their own request, so no one else is involved.
		grantTesterTimeApproval(t, app)
		if _, err := app.DB().NewQuery("UPDATE job_requests SET approver=creator WHERE id='" + id + "'").Execute(); err != nil {
			t.Fatal(err)
		}
		before := len(app.TestMailer.Messages())
		clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", testerTimeEmail, requestActionData(t, app, id), http.StatusOK)
		clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/approve", testerTimeEmail, requestActionData(t, app, id), http.StatusOK)
		if status := workflowRecord(t, app, "job_requests", id).GetString("status"); status != "Awaiting creation" {
			t.Fatalf("own approval did not complete: %s", status)
		}
		expectWorkflowEmails(t, app, before)
	})
}

func TestProjectValueEmails(t *testing.T) {
	path := "/jobs/" + legacyWorkflowJob + "/details"
	submitted := func(to, name, value, revised string) workflowEmail {
		return workflowEmail{to, "Approval needed: project 26-9876 value change", `Hello ` + name + `,

Horace Silver has asked you to approve a new project value.` + revised + `

Project: 26-9876
Client: Sepulchi & Co
Current value: $0
Proposed value: ` + value + `

Description:
Legacy zero value workflow fixture

Review the change and its authorization PDF, then approve or reject it here:

%s`, path}
	}
	revise := func(t *testing.T, app *tests.TestApp, id, value, approver string) {
		t.Helper()
		fields := map[string]string{"job": legacyWorkflowJob, "project_value": value, "assigned_approver": approver, "version": fmt.Sprint(workflowRecord(t, app, "job_attachments", id).GetInt("version"))}
		workflowUploadMethod(t, app, http.MethodPatch, "/api/job_attachments/"+id, horaceEmail, "%PDF-1.4\nValue email\n%%EOF", fields, http.StatusOK)
	}
	t.Run("revised and approved", func(t *testing.T) {
		app := newWorkflowTestApp(t, false)
		before := len(app.TestMailer.Messages())
		id := createValueChange(t, app, legacyWorkflowJob, 12000, "Value email")
		expectWorkflowEmails(t, app, before, submitted(tierTwoEmail, "Tier Two", "$12,000", ""))

		// The same approver is told the revision replaces the change they were sent.
		before = len(app.TestMailer.Messages())
		revise(t, app, id, "10000", "6bq4j0eb26631dy")
		expectWorkflowEmails(t, app, before, submitted(tierTwoEmail, "Tier Two", "$10,000", " This replaces a version of this change sent to you earlier."))

		before = len(app.TestMailer.Messages())
		clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/approve", workflowApprover, attachmentActionData(t, app, id), http.StatusOK)
		expectWorkflowEmails(t, app, before, workflowEmail{horaceEmail, "Approved: project 26-9876 value change", `Hello Horace Silver,

Tier Two approved the new value for project 26-9876 (Sepulchi & Co).

Previous value: $0
New value: $10,000

The new value is now in effect. View the project here:

%s`, path})
	})
	t.Run("new approver", func(t *testing.T) {
		app := newWorkflowTestApp(t, false)
		grantTesterTimeApproval(t, app)
		id := createValueChange(t, app, legacyWorkflowJob, 10000, "Value email")
		// A newly selected approver was not sent the earlier version.
		before := len(app.TestMailer.Messages())
		revise(t, app, id, "10000", "rzr98oadsp9qc11")
		expectWorkflowEmails(t, app, before, submitted(testerTimeEmail, "Tester Time", "$10,000", ""))
	})
	t.Run("rejected", func(t *testing.T) {
		app := newWorkflowTestApp(t, false)
		id := createValueChange(t, app, legacyWorkflowJob, 10000, "Value email rejection")
		before := len(app.TestMailer.Messages())
		clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/reject", workflowApprover, attachmentRejectData(t, app, id, "Attach the signed change order."), http.StatusOK)
		expectWorkflowEmails(t, app, before, workflowEmail{horaceEmail, "Rejected: project 26-9876 value change", `Hello Horace Silver,

Tier Two rejected the proposed value of $10,000 for project 26-9876 (Sepulchi & Co) for the following reason:

Attach the signed change order.

The project value remains $0. You can correct the change and send it for approval again here:

%s`, path})
	})
}

// Workflow emails are sent when the action commits, not left for a later
// scheduled send.
func TestProjectWorkflowEmailsAreSentWithTheAction(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	createValueChange(t, app, legacyWorkflowJob, 10000, "Value email delivery")
	notices, err := app.FindRecordsByFilter("notifications", "template.code='job_value_submitted'", "", 0, 0)
	if err != nil || len(notices) != 1 {
		t.Fatalf("value notices: %v, count %d", err, len(notices))
	}
	if notices[0].GetString("status") != "sent" || notices[0].GetString("user") != "f2j5a8vk006baub" || !notices[0].GetBool("system_notification") {
		t.Fatalf("notice was not sent with its actor: %v", notices[0].PublicExport())
	}
}
