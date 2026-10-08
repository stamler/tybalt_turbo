package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"
)

func assertQueuedWorkflowNotices(t *testing.T, app *tests.TestApp, template, action, actor, link string, recipients ...string) {
	t.Helper()
	records, err := app.FindRecordsByFilter("notifications", "template.code={:code}", "", 0, 0, dbx.Params{"code": template})
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]bool{}
	for _, uid := range recipients {
		expected[uid] = true
	}
	matched := 0
	for _, notice := range records {
		data := map[string]any{}
		if err = json.Unmarshal([]byte(notice.GetString("data")), &data); err != nil {
			t.Fatal(err)
		}
		if data["Action"] != action {
			continue
		}
		matched++
		recipient := notice.GetString("recipient")
		if !expected[recipient] {
			t.Fatalf("unexpected or repeated %s recipient: %s", action, recipient)
		}
		delete(expected, recipient)
		if notice.GetString("status") != "pending" || !notice.GetBool("system_notification") || notice.GetString("user") != actor {
			t.Fatalf("notice must remain queued with actor and system flag: %v", notice.PublicExport())
		}
		url, _ := data["ActionURL"].(string)
		if !strings.HasSuffix(url, link) {
			t.Fatalf("obsolete or incorrect workflow link: %q", url)
		}
	}
	if matched != len(recipients) || len(expected) != 0 {
		t.Fatalf("%s/%s recipients: matched %d, missing %v", template, action, matched, expected)
	}
}

func TestProjectWorkflowActionsQueueNoticesForRequiredRecipients(t *testing.T) {
	t.Run("requests", func(t *testing.T) {
		app := newWorkflowTestApp(t, true)
		const id = "parequest000001"
		const creator = "rzr98oadsp9qc11"
		const approver = "6bq4j0eb26631dy"
		clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", "time@test.com", requestActionData(t, app, id), http.StatusOK)
		assertQueuedWorkflowNotices(t, app, "job_request_workflow", "submit", creator, "/job-requests/"+id, creator, approver)
		clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/approve", workflowApprover, requestActionData(t, app, id), http.StatusOK)
		assertQueuedWorkflowNotices(t, app, "job_request_workflow", "approve", approver, "/job-requests/"+id, creator, approver)
		rejection := requestActionData(t, app, id)
		rejection["rejection_reason"] = "Revise the signed scope."
		clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/reject", workflowApprover, rejection, http.StatusOK)
		assertQueuedWorkflowNotices(t, app, "job_request_workflow", "reject", approver, "/job-requests/"+id, creator, approver)
	})
	for _, action := range []string{"approve", "reject"} {
		t.Run("value_"+action, func(t *testing.T) {
			app := newWorkflowTestApp(t, false)
			const editor = "f2j5a8vk006baub"
			const approver = "6bq4j0eb26631dy"
			id := createValueChange(t, app, legacyWorkflowJob, 10000, "Notice lifecycle "+action)
			link := "/jobs/" + legacyWorkflowJob + "/details"
			assertQueuedWorkflowNotices(t, app, "job_value_workflow", "submitted", editor, link, editor, approver)
			data := attachmentRejectData(t, app, id, "Correct the proposed amount.")
			clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/"+action, workflowApprover, data, http.StatusOK)
			result := "approved"
			if action == "reject" {
				result = "rejected"
			}
			assertQueuedWorkflowNotices(t, app, "job_value_workflow", result, approver, link, editor, approver)
		})
	}
}
