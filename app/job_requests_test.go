package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const workflowAuthor = "author@soup.com"
const workflowApprover = "tier2@poapprover.com"

func newWorkflowTestApp(t *testing.T, enabled bool) *tests.TestApp {
	t.Helper()
	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)
	// Change only the feature flag to exercise both supported release states.
	config, err := app.FindFirstRecordByData("app_config", "key", "jobs")
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]any{}
	if err = json.Unmarshal([]byte(config.GetString("value")), &values); err != nil {
		t.Fatal(err)
	}
	values["require_job_request_approval"] = enabled
	config.Set("value", values)
	if err = app.Save(config); err != nil {
		t.Fatal(err)
	}
	return app
}

func workflowToken(t *testing.T, app *tests.TestApp, email string) string {
	t.Helper()
	auth, err := app.FindAuthRecordByEmail("users", email)
	if err != nil {
		t.Fatal(err)
	}
	token, err := auth.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func workflowRecord(t *testing.T, app *tests.TestApp, collection, id string) *core.Record {
	t.Helper()
	r, err := app.FindRecordById(collection, id)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func createWorkflowRequest(t *testing.T, app *tests.TestApp) string {
	t.Helper()
	result := clientSetupJSON(t, app, http.MethodPost, "/api/job_requests", workflowAuthor, map[string]any{"request": map[string]any{"description": "New request for workflow acceptance"}}, http.StatusOK)
	if r, ok := result["request"].(map[string]any); ok {
		if id, ok := r["id"].(string); ok {
			return id
		}
	}
	if id, ok := result["id"].(string); ok {
		return id
	}
	t.Fatalf("request response has no id: %#v", result)
	return ""
}

func completeWorkflowRequest(t *testing.T, app *tests.TestApp, id string) {
	t.Helper()
	billing, err := app.FindFirstRecordByData("client_invoicing_information", "client", "pqpd90fqd5ohjcs")
	if err != nil {
		t.Fatalf("complete request requires invoicing fixture: %v", err)
	}
	r := workflowRecord(t, app, "job_requests", id)
	clientSetupJSON(t, app, http.MethodPatch, "/api/job_requests/"+id, workflowAuthor, map[string]any{
		"version":     r.GetInt("version"),
		"request":     map[string]any{"description": "Project request with complete client information", "client": "pqpd90fqd5ohjcs", "contact": "184qbpajciwrrp2", "manager": "f2j5a8vk006baub", "branch": "80875lm27v8wgi4", "rate_sheet": "c41ofep525bcacj", "location": "87Q8H976+2M", "project_award_date": "2026-09-01", "project_completion_date": "2027-09-30", "project_value": 10000, "time_and_materials": false, "fn_agreement_answer": "No", "invoicing_information": billing.Id, "approver": "6bq4j0eb26631dy"},
		"allocations": []map[string]any{{"division": "fy4i9poneukvq9u", "hours": 10}},
	}, http.StatusOK)
}

func workflowUpload(t *testing.T, app *tests.TestApp, path, email, content string, fields map[string]string, status int) *httptest.ResponseRecorder {
	t.Helper()
	return workflowUploadMethod(t, app, http.MethodPost, path, email, content, fields, status)
}

func workflowUploadMethod(t *testing.T, app *tests.TestApp, method, path, email, content string, fields map[string]string, status int) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="document"; filename="authorization.pdf"`)
	header.Set("Content-Type", "application/pdf")
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	response := performTestAPIRequest(t, app, method, path, &body, map[string]string{"Authorization": workflowToken(t, app, email), "Content-Type": writer.FormDataContentType()})
	mustStatus(t, response, status)
	return response
}

func uploadRequestDocument(t *testing.T, app *tests.TestApp, id string) {
	t.Helper()
	r := workflowRecord(t, app, "job_requests", id)
	workflowUpload(t, app, "/api/job_requests/"+id+"/document", workflowAuthor, "%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\n%%EOF", map[string]string{"version": fmt.Sprint(r.GetInt("version"))}, http.StatusOK)
}

func requestActionData(t *testing.T, app *tests.TestApp, id string) map[string]any {
	t.Helper()
	detail := clientSetupJSON(t, app, http.MethodGet, "/api/job_requests/"+id, workflowAuthor, nil, http.StatusOK)
	r := workflowRecord(t, app, "job_requests", id)
	answers := map[string]any{}
	for _, key := range []string{"fn_agreement_reviewed", "no_known_conflict", "no_executive_restriction", "no_duplicate_work", "pdf_sufficient", "allocations_complete"} {
		answers[key] = true
	}
	result := map[string]any{"version": r.GetInt("version"), "review_token": detail["review_token"], "checklist_version": 1, "checklist_answers": answers, "duplicate_work_checked": true}
	attachment, err := app.FindFirstRecordByData("job_attachments", "request", id)
	if err == nil {
		result["document_hash"] = attachment.GetString("document_hash")
	}
	return result
}

func TestJobRequestFeatureFlagAndCreatePermissions(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	body := map[string]any{"request": map[string]any{"description": "Draft"}}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests", workflowAuthor, body, http.StatusForbidden)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests", "", body, http.StatusUnauthorized)
}

func TestJobRequestDraftValidationAndVersion(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests", "u_no_claims@example.com", map[string]any{"request": map[string]any{}}, http.StatusForbidden)
	id := createWorkflowRequest(t, app)
	for _, data := range []map[string]any{{"project_value": 10.5}, {"project_value": -1}, {"project_status": "Cancelled"}, {"status": "Awaiting creation"}, {"time_and_materials": "false"}} {
		clientSetupJSON(t, app, http.MethodPatch, "/api/job_requests/"+id, workflowAuthor, map[string]any{"version": 1, "request": data}, http.StatusBadRequest)
		if r := workflowRecord(t, app, "job_requests", id); r.GetInt("version") != 1 {
			t.Fatal("failed edit changed request version")
		}
	}
	// Job allocations store whole hours, so a request cannot hold part hours.
	clientSetupJSON(t, app, http.MethodPatch, "/api/job_requests/"+id, workflowAuthor, map[string]any{"version": 1, "request": map[string]any{}, "allocations": []map[string]any{{"division": "fy4i9poneukvq9u", "hours": 0.25}}}, http.StatusBadRequest)
	if allocations, err := app.FindRecordsByFilter("job_request_allocations", "request={:id}", "", 0, 0, dbx.Params{"id": id}); err != nil || len(allocations) != 0 {
		t.Fatalf("part-hour allocation saved: %v %v", allocations, err)
	}
	clientSetupJSON(t, app, http.MethodPatch, "/api/job_requests/"+id, workflowAuthor, map[string]any{"version": 0, "request": map[string]any{"description": "Stale edit"}}, http.StatusConflict)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", workflowAuthor, map[string]any{"version": 1}, http.StatusBadRequest)
	clientSetupJSON(t, app, http.MethodGet, "/api/job_requests/"+id, "u_no_claims@example.com", nil, http.StatusForbidden)
}

func TestJobRequestApprovalCreationLifecycle(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	id := createWorkflowRequest(t, app)
	completeWorkflowRequest(t, app, id)
	uploadRequestDocument(t, app, id)
	initial := workflowRecord(t, app, "job_requests", id)
	edited := initial.GetString("last_edited")
	clientSetupJSON(t, app, http.MethodGet, "/api/job_requests/"+id+"/print", workflowAuthor, nil, http.StatusOK)
	if workflowRecord(t, app, "job_requests", id).GetString("last_edited") != edited {
		t.Fatal("printing changed edit time")
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", workflowAuthor, requestActionData(t, app, id), http.StatusOK)
	for _, key := range []string{"fn_agreement_reviewed", "no_known_conflict", "no_executive_restriction", "no_duplicate_work", "pdf_sufficient", "allocations_complete"} {
		data := requestActionData(t, app, id)
		data["checklist_answers"].(map[string]any)[key] = false
		clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/approve", workflowApprover, data, http.StatusBadRequest)
	}
	stale := requestActionData(t, app, id)
	stale["review_token"] = "stale"
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/approve", workflowApprover, stale, http.StatusConflict)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/approve", workflowApprover, requestActionData(t, app, id), http.StatusOK)
	approved := workflowRecord(t, app, "job_requests", id)
	if approved.GetString("status") != "Awaiting creation" {
		t.Fatal("approval did not enter creation queue")
	}
	if approved.GetString("last_edited") != edited {
		t.Fatal("approval changed last edit time")
	}
	before, err := app.FindFirstRecordByData("job_attachments", "request", id)
	if err != nil {
		t.Fatal(err)
	}
	hash, filename, approvedAt := before.GetString("document_hash"), before.GetString("document"), before.GetString("approved")
	noCheck := requestActionData(t, app, id)
	noCheck["duplicate_work_checked"] = false
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/create", workflowAuthor, noCheck, http.StatusBadRequest)
	result := clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/create", workflowAuthor, requestActionData(t, app, id), http.StatusOK)
	jobID, _ := result["job_id"].(string)
	job := workflowRecord(t, app, "jobs", jobID)
	if job.GetFloat("project_value") != 10000 || job.GetString("number") == "" || strings.HasPrefix(job.GetString("number"), "P") {
		t.Fatal("creation did not preserve value and assign project number")
	}
	if _, err = app.FindRecordById("job_requests", id); err == nil {
		t.Fatal("created request still exists")
	}
	after := workflowRecord(t, app, "job_attachments", before.Id)
	if after.GetString("job") != jobID || after.GetString("request") != "" || after.GetString("document_hash") != hash || after.GetString("document") != filename || after.GetString("approved") != approvedAt || !after.GetBool("initial_approval") {
		t.Fatal("creation did not preserve and transfer final approval")
	}
	clientSetupJSON(t, app, http.MethodDelete, "/api/job_attachments/"+before.Id, workflowAuthor, attachmentActionData(t, app, before.Id), http.StatusBadRequest)
	allocs, err := app.FindRecordsByFilter("job_time_allocations", "job={:id}", "", 0, 0, dbx.Params{"id": jobID})
	if err != nil || len(allocs) != 1 {
		t.Fatalf("allocation transfer: %v, count %d", err, len(allocs))
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/create", workflowAuthor, map[string]any{"version": approved.GetInt("version")}, http.StatusNotFound)
}

// Contact email is optional, but a request's project and billing contacts must
// each have one or be marked as having none before submission.
func TestJobRequestContactsNeedEmailOrNoEmailMark(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	id := createWorkflowRequest(t, app)
	completeWorkflowRequest(t, app, id)
	uploadRequestDocument(t, app, id)
	contact := workflowRecord(t, app, "client_contacts", workflowRecord(t, app, "job_requests", id).GetString("contact"))
	contact.Set("email", "")
	if err := app.Save(contact); err != nil {
		t.Fatalf("contact without email rejected: %v", err)
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", workflowAuthor, requestActionData(t, app, id), http.StatusBadRequest)
	contact.Set("no_email", true)
	if err := app.Save(contact); err != nil {
		t.Fatal(err)
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", workflowAuthor, requestActionData(t, app, id), http.StatusOK)
}

func TestJobRequestRejectionRequiresEditAndRecallClearsApproval(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	id := createWorkflowRequest(t, app)
	completeWorkflowRequest(t, app, id)
	uploadRequestDocument(t, app, id)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", workflowAuthor, requestActionData(t, app, id), http.StatusOK)
	reject := requestActionData(t, app, id)
	reject["rejection_reason"] = "Update the scope."
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/reject", workflowApprover, reject, http.StatusOK)
	r := workflowRecord(t, app, "job_requests", id)
	clientSetupJSON(t, app, http.MethodPatch, "/api/job_requests/"+id, workflowAuthor, map[string]any{"version": r.GetInt("version"), "request": map[string]any{"description": r.GetString("description")}}, http.StatusOK)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", workflowAuthor, requestActionData(t, app, id), http.StatusBadRequest)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/recall", workflowAuthor, requestActionData(t, app, id), http.StatusBadRequest)
	clientSetupJSON(t, app, http.MethodPatch, "/api/job_requests/"+id, workflowAuthor, map[string]any{"version": r.GetInt("version"), "request": map[string]any{"description": "Corrected scope of work"}}, http.StatusOK)
	r = workflowRecord(t, app, "job_requests", id)
	if r.GetString("rejected") != "" || r.GetString("status") != "Draft" {
		t.Fatal("successful edit did not clear rejection")
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", workflowAuthor, requestActionData(t, app, id), http.StatusOK)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/approve", workflowApprover, requestActionData(t, app, id), http.StatusOK)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/recall", workflowAuthor, requestActionData(t, app, id), http.StatusOK)
	a, err := app.FindFirstRecordByData("job_attachments", "request", id)
	if err != nil {
		t.Fatal(err)
	}
	if a.GetString("approved") != "" || a.GetString("approver") != "" || a.GetInt("checklist_version") != 0 || a.GetString("document") == "" {
		t.Fatal("recall did not clear approval and retain PDF")
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/create", workflowAuthor, requestActionData(t, app, id), http.StatusBadRequest)
}

func TestJobRequestSelectedApproverAndCreatorPermissions(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	const id = "parequest000001"
	// The fixture creator has time but no job claim. Approval and creation stay separate.
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", "time@test.com", requestActionData(t, app, id), http.StatusOK)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/approve", workflowAuthor, requestActionData(t, app, id), http.StatusForbidden)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/recall", workflowAuthor, requestActionData(t, app, id), http.StatusForbidden)
	clientSetupJSON(t, app, http.MethodPatch, "/api/job_requests/"+id, workflowAuthor, map[string]any{"version": 1, "request": map[string]any{"description": "Another employee edit"}}, http.StatusForbidden)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/approve", workflowApprover, requestActionData(t, app, id), http.StatusOK)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/create", workflowApprover, requestActionData(t, app, id), http.StatusForbidden)
	queue := clientSetupJSON(t, app, http.MethodGet, "/api/job_requests?queue=creation&perPage=1", workflowAuthor, nil, http.StatusOK)
	if queue["totalItems"] != float64(1) {
		t.Fatalf("creation queue must include other creators: %#v", queue)
	}
	clientSetupJSON(t, app, http.MethodGet, "/api/job_requests?queue=creation", workflowApprover, nil, http.StatusForbidden)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/recall", "time@test.com", requestActionData(t, app, id), http.StatusOK)
	queue = clientSetupJSON(t, app, http.MethodGet, "/api/job_requests?queue=creation", workflowAuthor, nil, http.StatusOK)
	if queue["totalItems"] != float64(0) {
		t.Fatalf("recall must remove queue entry: %#v", queue)
	}
}

func TestJobRequestApprovalRechecksCurrentAuthority(t *testing.T) {
	for _, limit := range []float64{0, 99999, 100000, 100001} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			app := newWorkflowTestApp(t, true)
			const id = "parequest000001"
			clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", "time@test.com", requestActionData(t, app, id), http.StatusOK)
			// Change the existing authority fixture after submission to test revalidation.
			if _, err := app.DB().NewQuery("UPDATE project_authorization_props SET max_value={:value} WHERE id='paprops00000001'").Bind(dbx.Params{"value": limit}).Execute(); err != nil {
				t.Fatal(err)
			}
			response := performJSONRequest(t, app, http.MethodPost, "/api/job_requests/"+id+"/approve", workflowToken(t, app, workflowApprover), requestActionData(t, app, id))
			if limit >= 100000 {
				mustStatus(t, response, http.StatusOK)
			} else if response.Code < 400 {
				t.Fatalf("insufficient current authority approved request: %s", response.Body.String())
			}
			if limit < 100000 && workflowRecord(t, app, "job_requests", id).GetString("status") != "Awaiting approval" {
				t.Fatal("failed approval changed request state")
			}
		})
	}
}

func TestJobRequestDuplicateComparisonDoesNotExposeOtherRequestsToApprover(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	id := createWorkflowRequest(t, app)
	completeWorkflowRequest(t, app, id)
	result := clientSetupJSON(t, app, http.MethodGet, "/api/job_requests/parequest000001/duplicates", workflowApprover, nil, http.StatusOK)
	if requests, ok := result["requests"].([]any); !ok || len(requests) != 0 {
		t.Fatal("approver comparison exposed other requests")
	}
	result = clientSetupJSON(t, app, http.MethodGet, "/api/job_requests/parequest000001/duplicates", workflowAuthor, nil, http.StatusOK)
	if requests, ok := result["requests"].([]any); !ok || len(requests) == 0 {
		t.Fatal("job holder cannot compare pending requests")
	}
	// Remove only the fixture assignment to verify direct and generic access immediately changes.
	if _, err := app.DB().NewQuery("UPDATE job_requests SET approver='' WHERE id='parequest000001'").Execute(); err != nil {
		t.Fatal(err)
	}
	clientSetupJSON(t, app, http.MethodGet, "/api/job_requests/parequest000001", workflowApprover, nil, http.StatusForbidden)
	clientSetupJSON(t, app, http.MethodGet, "/api/collections/job_requests/records/parequest000001", workflowApprover, nil, http.StatusNotFound)
}

func TestJobRequestQueueBadgeAndDisabledStateRetention(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	const id = "parequest000001"
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", "time@test.com", requestActionData(t, app, id), http.StatusOK)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/approve", workflowApprover, requestActionData(t, app, id), http.StatusOK)
	badges := clientSetupJSON(t, app, http.MethodGet, "/api/nav/badges", workflowAuthor, nil, http.StatusOK)
	if badges["/job-requests?queue=creation"] != float64(1) {
		t.Fatalf("queue badge mismatch: %#v", badges)
	}
	queue := clientSetupJSON(t, app, http.MethodGet, "/api/job_requests?queue=creation&perPage=1&page=2", workflowAuthor, nil, http.StatusOK)
	if queue["totalItems"] != float64(1) || len(queue["items"].([]any)) != 0 {
		t.Fatalf("count should cover all pages: %#v", queue)
	}
	r := workflowRecord(t, app, "job_requests", id)
	version, edited := r.GetInt("version"), r.GetString("last_edited")
	// Toggle the existing config row to prove retained requests survive both states.
	config, err := app.FindFirstRecordByData("app_config", "key", "jobs")
	if err != nil {
		t.Fatal(err)
	}
	settings := map[string]any{}
	if err = json.Unmarshal([]byte(config.GetString("value")), &settings); err != nil {
		t.Fatal(err)
	}
	settings["require_job_request_approval"] = false
	config.Set("value", settings)
	if err = app.Save(config); err != nil {
		t.Fatal(err)
	}
	clientSetupJSON(t, app, http.MethodGet, "/api/job_requests?queue=creation", workflowAuthor, nil, http.StatusForbidden)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/create", workflowAuthor, requestActionData(t, app, id), http.StatusForbidden)
	badges = clientSetupJSON(t, app, http.MethodGet, "/api/nav/badges", workflowAuthor, nil, http.StatusOK)
	if _, ok := badges["/job-requests?queue=creation"]; ok {
		t.Fatal("disabled queue badge remains")
	}
	r = workflowRecord(t, app, "job_requests", id)
	if r.GetInt("version") != version || r.GetString("last_edited") != edited || r.GetString("status") != "Awaiting creation" {
		t.Fatal("flag change altered retained request")
	}
	settings["require_job_request_approval"] = true
	config.Set("value", settings)
	if err = app.Save(config); err != nil {
		t.Fatal(err)
	}
	queue = clientSetupJSON(t, app, http.MethodGet, "/api/job_requests?queue=creation", workflowAuthor, nil, http.StatusOK)
	if queue["totalItems"] != float64(1) {
		t.Fatal("re-enabled queue lost retained request")
	}
}

func TestJobRequestAbandonmentAndDuplicatePDFCleanup(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	const id = "parequest000001"
	newID := createWorkflowRequest(t, app)
	content, err := os.ReadFile("test_seed_data/files/job_attachments/paattachment001/authorization.pdf")
	if err != nil {
		t.Fatal(err)
	}
	workflowUpload(t, app, "/api/job_requests/"+newID+"/document", workflowAuthor, string(content), map[string]string{"version": "1"}, http.StatusBadRequest)
	// Adjust only the fixture edit time to check both sides of the 60-day boundary.
	for _, age := range []time.Duration{60*24*time.Hour - time.Hour, 60*24*time.Hour + time.Second} {
		if _, err = app.DB().NewQuery("UPDATE job_requests SET last_edited={:edited} WHERE id={:id}").Bind(dbx.Params{"id": id, "edited": time.Now().UTC().Add(-age).Format("2006-01-02 15:04:05.000Z")}).Execute(); err != nil {
			t.Fatal(err)
		}
		result := clientSetupJSON(t, app, http.MethodGet, "/api/job_requests?queue=abandoned", workflowAuthor, nil, http.StatusOK)
		want := float64(0)
		if age > 60*24*time.Hour {
			want = 1
		}
		if result["totalItems"] != want {
			t.Fatalf("age %s: list=%#v", age, result)
		}
	}
	response := workflowUpload(t, app, "/api/job_requests/"+newID+"/document", workflowAuthor, string(content), map[string]string{"version": "1"}, http.StatusBadRequest)
	if !strings.Contains(response.Body.String(), id) || !strings.Contains(response.Body.String(), "admin") {
		t.Fatal("duplicate message does not identify abandoned request and admin resolution")
	}
	clientSetupJSON(t, app, http.MethodDelete, "/api/job_requests/"+id, "time@test.com", nil, http.StatusForbidden)
	clientSetupJSON(t, app, http.MethodDelete, "/api/job_requests/"+id, workflowAuthor, nil, http.StatusNoContent)
	workflowUpload(t, app, "/api/job_requests/"+newID+"/document", workflowAuthor, string(content), map[string]string{"version": "1"}, http.StatusOK)
}

func TestJobRequestConcurrentCreationMakesOneJob(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	const id = "parequest000001"
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", "time@test.com", requestActionData(t, app, id), http.StatusOK)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/approve", workflowApprover, requestActionData(t, app, id), http.StatusOK)
	data := requestActionData(t, app, id)
	token := workflowToken(t, app, workflowAuthor)
	before, err := app.CountRecords("jobs")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	responses := make(chan *httptest.ResponseRecorder, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			responses <- performJSONRequest(t, app, http.MethodPost, "/api/job_requests/"+id+"/create", token, data)
		}()
	}
	wg.Wait()
	close(responses)
	success := 0
	for response := range responses {
		if response.Code == http.StatusOK {
			success++
		} else if response.Code < 400 {
			t.Fatalf("unexpected creation response: %d", response.Code)
		}
	}
	if success != 1 {
		t.Fatalf("successful creations=%d, want 1", success)
	}
	after, err := app.CountRecords("jobs")
	if err != nil {
		t.Fatal(err)
	}
	if after != before+1 {
		t.Fatalf("created %d jobs, want 1", after-before)
	}
}

func TestJobRequestChangedSharedDataPreventsCreationAndKeepsEvidence(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	const id = "parequest000001"
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", "time@test.com", requestActionData(t, app, id), http.StatusOK)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/approve", workflowApprover, requestActionData(t, app, id), http.StatusOK)
	// Simulate a later shared contact edit without changing the request or signed PDF.
	if _, err := app.DB().NewQuery("UPDATE client_contacts SET email='revised@example.com' WHERE id='184qbpajciwrrp2'").Execute(); err != nil {
		t.Fatal(err)
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/create", workflowAuthor, requestActionData(t, app, id), http.StatusConflict)
	if workflowRecord(t, app, "job_requests", id).GetString("status") != "Awaiting creation" {
		t.Fatal("failed creation discarded approved request")
	}
	a := workflowRecord(t, app, "job_attachments", "paattachment001")
	if a.GetString("request") != id || a.GetString("approved") == "" || a.GetString("document") == "" {
		t.Fatal("failed creation changed retained approval evidence")
	}
	if workflowRecord(t, app, "job_request_allocations", "paallocation001").GetString("request") != id {
		t.Fatal("failed creation discarded allocations")
	}
}

func TestJobRequestDraftRelationsAndChecklistShape(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	id := createWorkflowRequest(t, app)
	for _, fields := range []map[string]any{
		{"client": "pqpd90fqd5ohjcs", "parent": "cjf0kt0defhq480"},
		{"client": "lb0fnenkeyitsny", "parent": "pcproposal00001"},
		{"client": "iq7ge8osf6kuj3l", "parent": "testsubjob01id"},
		{"client": "pqpd90fqd5ohjcs", "proposal": "pcproposal00001"},
		{"client": "pqpd90fqd5ohjcs", "proposal": legacyWorkflowJob},
	} {
		clientSetupJSON(t, app, http.MethodPatch, "/api/job_requests/"+id, workflowAuthor, map[string]any{"version": 1, "request": fields}, http.StatusBadRequest)
	}
	const ready = "parequest000001"
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+ready+"/submit", "time@test.com", requestActionData(t, app, ready), http.StatusOK)
	for _, mutation := range []func(map[string]any){
		func(data map[string]any) { data["checklist_version"] = 2 },
		func(data map[string]any) { delete(data["checklist_answers"].(map[string]any), "pdf_sufficient") },
		func(data map[string]any) { data["checklist_answers"].(map[string]any)["pdf_sufficient"] = "true" },
		func(data map[string]any) { data["checklist_answers"].(map[string]any)["unexpected_answer"] = true },
	} {
		data := requestActionData(t, app, ready)
		mutation(data)
		clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+ready+"/approve", workflowApprover, data, http.StatusBadRequest)
	}
	reviewed := requestActionData(t, app, ready)
	// Change the fixture billing instructions after the review, before approval.
	if _, err := app.DB().NewQuery("UPDATE client_invoicing_information SET invoicing_instructions='Updated instructions' WHERE id='painvoice000001'").Execute(); err != nil {
		t.Fatal(err)
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+ready+"/approve", workflowApprover, reviewed, http.StatusConflict)
}

func TestJobRequestCreationRetainsValidApprovalAfterAuthorityRemoval(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	const id = "parequest000001"
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/submit", "time@test.com", requestActionData(t, app, id), http.StatusOK)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/approve", workflowApprover, requestActionData(t, app, id), http.StatusOK)
	// Remove only the existing authority assignment after its completed approval.
	if _, err := app.DB().NewQuery("DELETE FROM user_claims WHERE id='paucapprover001'").Execute(); err != nil {
		t.Fatal(err)
	}
	result := clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+id+"/create", workflowAuthor, requestActionData(t, app, id), http.StatusOK)
	jobID, _ := result["job_id"].(string)
	a := workflowRecord(t, app, "job_attachments", "paattachment001")
	if a.GetString("job") != jobID || a.GetString("approver") != "6bq4j0eb26631dy" || a.GetFloat("approval_max_value") != 1000000 {
		t.Fatal("creation changed completed approval evidence after claim removal")
	}
}
