package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"
)

const legacyWorkflowJob = "pawflegacy00001"

func createValueChange(t *testing.T, app *tests.TestApp, job string, value int, label string) string {
	t.Helper()
	response := workflowUpload(t, app, "/api/job_attachments", workflowAuthor, "%PDF-1.4\n"+label+"\n%%EOF", map[string]string{"job": job, "project_value": fmt.Sprint(value), "assigned_approver": "6bq4j0eb26631dy"}, http.StatusOK)
	result := map[string]any{}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	id, ok := result["id"].(string)
	if !ok || id == "" {
		t.Fatalf("missing attachment id: %#v", result)
	}
	return id
}
func attachmentActionData(t *testing.T, app *tests.TestApp, id string) map[string]any {
	t.Helper()
	a := workflowRecord(t, app, "job_attachments", id)
	return map[string]any{"version": a.GetInt("version"), "document_hash": a.GetString("document_hash")}
}

// attachmentRejectData returns the rejection the approver's page sends,
// including the review token of the job data it shows.
func attachmentRejectData(t *testing.T, app *tests.TestApp, id, reason string) map[string]any {
	t.Helper()
	data := attachmentActionData(t, app, id)
	data["rejection_reason"] = reason
	job := workflowRecord(t, app, "job_attachments", id).GetString("job")
	list := clientSetupJSON(t, app, http.MethodGet, "/api/job_attachments?job="+job, workflowApprover, nil, http.StatusOK)
	for _, item := range list["items"].([]any) {
		if row := item.(map[string]any); row["id"] == id {
			data["review_token"] = row["current_review_token"]
		}
	}
	return data
}

func TestJobValueApprovalAndHistoricalZeroRestoreWithRequestsDisabled(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	job := workflowRecord(t, app, "jobs", legacyWorkflowJob)
	if job.GetFloat("project_value") != 0 || !job.GetBool("time_and_materials") {
		t.Fatal("legacy fixture must be a zero-value T&M job")
	}
	id := createValueChange(t, app, legacyWorkflowJob, 10000, "First later approval")
	if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetFloat("project_value") != 0 {
		t.Fatal("pending change replaced effective value")
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/approve", workflowAuthor, attachmentActionData(t, app, id), http.StatusForbidden)
	stale := attachmentActionData(t, app, id)
	stale["version"] = 0
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/approve", workflowApprover, stale, http.StatusConflict)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/approve", workflowApprover, attachmentActionData(t, app, id), http.StatusOK)
	a := workflowRecord(t, app, "job_attachments", id)
	if a.GetFloat("previous_project_value") != 0 || a.GetString("approver") != "6bq4j0eb26631dy" || a.GetFloat("approval_max_value") != 1000000 {
		t.Fatal("approval did not retain historical value and authority")
	}
	if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetFloat("project_value") != 10000 {
		t.Fatal("approval did not set effective value")
	}
	clientSetupJSON(t, app, http.MethodDelete, "/api/job_attachments/"+id, workflowApprover, attachmentActionData(t, app, id), http.StatusForbidden)
	clientSetupJSON(t, app, http.MethodDelete, "/api/job_attachments/"+id, workflowAuthor, attachmentActionData(t, app, id), http.StatusNoContent)
	if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetFloat("project_value") != 0 {
		t.Fatal("admin deletion did not restore exact historical zero")
	}
	if _, err := app.FindRecordById("job_attachments", id); err == nil {
		t.Fatal("deleted value evidence remains")
	}
}

func TestJobValueChangePermissionsFractionsAndSinglePendingRule(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	fields := map[string]string{"job": legacyWorkflowJob, "project_value": "5000", "assigned_approver": "6bq4j0eb26631dy"}
	workflowUpload(t, app, "/api/job_attachments", workflowApprover, "%PDF-1.4\nDenied\n%%EOF", fields, http.StatusForbidden)
	for _, value := range []string{"0", "-1", "0.5", "1000001"} {
		fields["project_value"] = value
		response := workflowUpload(t, app, "/api/job_attachments", workflowAuthor, "%PDF-1.4\nInvalid value\n%%EOF", fields, http.StatusBadRequest)
		_ = response
	}
	id := createValueChange(t, app, legacyWorkflowJob, 5000, "Single pending change")
	fields["project_value"] = "6000"
	workflowUpload(t, app, "/api/job_attachments", workflowAuthor, "%PDF-1.4\nSecond pending\n%%EOF", fields, http.StatusBadRequest)
	body := map[string]any{"project_value": 7777}
	response := performJSONRequest(t, app, http.MethodPatch, "/api/collections/jobs/records/"+legacyWorkflowJob, workflowToken(t, app, workflowAuthor), body)
	if response.Code < 400 {
		t.Fatalf("generic job update bypassed approval: %s", response.Body.String())
	}
	if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetFloat("project_value") != 0 {
		t.Fatal("failed mutation changed effective value")
	}
	clientSetupJSON(t, app, http.MethodDelete, "/api/job_attachments/"+id, workflowAuthor, attachmentActionData(t, app, id), http.StatusNoContent)
}

func TestJobValueRejectionAndAdminHistoryDeletionOrder(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	first := createValueChange(t, app, legacyWorkflowJob, 10000, "Approval one")
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+first+"/approve", workflowApprover, attachmentActionData(t, app, first), http.StatusOK)
	second := createValueChange(t, app, legacyWorkflowJob, 9000, "Approval decrease")
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+second+"/reject", workflowApprover, attachmentRejectData(t, app, second, ""), http.StatusBadRequest)
	reject := attachmentRejectData(t, app, second, "Correct the value.")
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+second+"/reject", workflowApprover, reject, http.StatusOK)
	if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetFloat("project_value") != 10000 {
		t.Fatal("rejection changed effective value")
	}
	clientSetupJSON(t, app, http.MethodDelete, "/api/job_attachments/"+first, workflowAuthor, attachmentActionData(t, app, first), http.StatusBadRequest)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+second+"/approve", workflowApprover, attachmentActionData(t, app, second), http.StatusBadRequest)
	clientSetupJSON(t, app, http.MethodDelete, "/api/job_attachments/"+second, workflowAuthor, attachmentActionData(t, app, second), http.StatusNoContent)
	third := createValueChange(t, app, legacyWorkflowJob, 9000, "Corrected approval decrease")
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+third+"/approve", workflowApprover, attachmentActionData(t, app, third), http.StatusOK)
	clientSetupJSON(t, app, http.MethodDelete, "/api/job_attachments/"+first, workflowAuthor, attachmentActionData(t, app, first), http.StatusBadRequest)
	clientSetupJSON(t, app, http.MethodDelete, "/api/job_attachments/"+third, workflowAuthor, attachmentActionData(t, app, third), http.StatusNoContent)
	if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetFloat("project_value") != 10000 {
		t.Fatal("latest approval deletion did not restore predecessor")
	}
}

func TestJobApprovalHistoryAndJobDeletionAreProtected(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	id := createValueChange(t, app, legacyWorkflowJob, 10000, "Protected history")
	job := workflowRecord(t, app, "jobs", legacyWorkflowJob)
	if err := app.Delete(job); err == nil {
		t.Fatal("model deletion bypassed retained evidence")
	}
	a := workflowRecord(t, app, "job_attachments", id)
	if err := app.Delete(a); err == nil {
		t.Fatal("model deletion bypassed workflow history protection")
	}
	a.Set("project_value", 1)
	if err := app.Save(a); err == nil {
		t.Fatal("model save bypassed value approval")
	}
	clientSetupJSON(t, app, http.MethodGet, "/api/job_attachments/"+id+"/document", "u_no_claims@example.com", nil, http.StatusForbidden)
	response := performJSONRequest(t, app, http.MethodGet, "/api/job_attachments/"+id+"/document", workflowToken(t, app, workflowApprover), nil)
	mustStatus(t, response, http.StatusOK)
	if workflowRecord(t, app, "job_attachments", id).GetFloat("project_value") != 10000 {
		t.Fatal("failed writes changed stored evidence")
	}
}

func TestJobValueSharedEditRequiresFreshReviewAndRejectedNoOpStaysRejected(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	id := createValueChange(t, app, legacyWorkflowJob, 10000, "Shared review document")
	before := workflowRecord(t, app, "job_attachments", id)
	oldVersion := before.GetInt("version")
	// Change only the shared billing contact to invalidate the reviewed data.
	if _, err := app.DB().NewQuery("UPDATE client_contacts SET email='changed-review@example.com' WHERE id='184qbpajciwrrp2'").Execute(); err != nil {
		t.Fatal(err)
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/approve", workflowApprover, attachmentActionData(t, app, id), http.StatusConflict)
	fields := map[string]string{"job": legacyWorkflowJob, "project_value": "10000", "assigned_approver": "6bq4j0eb26631dy", "version": fmt.Sprint(oldVersion)}
	workflowUploadMethod(t, app, http.MethodPatch, "/api/job_attachments/"+id, workflowAuthor, "%PDF-1.4\nShared review document\n%%EOF", fields, http.StatusOK)
	refreshed := workflowRecord(t, app, "job_attachments", id)
	if refreshed.GetInt("version") <= oldVersion || refreshed.GetString("review_token") == before.GetString("review_token") {
		t.Fatal("same-data pending save did not refresh review")
	}
	rejection := attachmentRejectData(t, app, id, "Revise the scope.")
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/reject", workflowApprover, rejection, http.StatusOK)
	rejected := workflowRecord(t, app, "job_attachments", id)
	fields["version"] = fmt.Sprint(rejected.GetInt("version"))
	workflowUploadMethod(t, app, http.MethodPatch, "/api/job_attachments/"+id, workflowAuthor, "%PDF-1.4\nShared review document\n%%EOF", fields, http.StatusOK)
	after := workflowRecord(t, app, "job_attachments", id)
	if after.GetString("status") != "Rejected" || after.GetString("rejection_reason") != "Revise the scope." || after.GetInt("version") != rejected.GetInt("version") {
		t.Fatal("unchanged rejected save cleared rejection")
	}
	fields["project_value"] = "11000"
	workflowUploadMethod(t, app, http.MethodPatch, "/api/job_attachments/"+id, workflowAuthor, "%PDF-1.4\nShared review document\n%%EOF", fields, http.StatusOK)
	after = workflowRecord(t, app, "job_attachments", id)
	if after.GetString("status") != "Pending" || after.GetString("rejection_reason") != "" {
		t.Fatal("successful value correction did not clear rejection")
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/approve", workflowApprover, attachmentActionData(t, app, id), http.StatusOK)
}

// A rejected value change can return for approval once its related data is
// corrected, without changing the value, approver, or PDF.
func TestJobValueRejectedForSharedDataResubmitsAfterCorrection(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	id := createValueChange(t, app, legacyWorkflowJob, 10000, "Correction review document")
	rejection := attachmentRejectData(t, app, id, "The billing contact email is wrong.")
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/reject", workflowApprover, rejection, http.StatusOK)
	rejected := workflowRecord(t, app, "job_attachments", id)
	clientSetupJSON(t, app, http.MethodPatch, "/api/collections/client_contacts/records/184qbpajciwrrp2", workflowAuthor, map[string]any{"email": "corrected-billing@example.com"}, http.StatusOK)
	fields := map[string]string{"job": legacyWorkflowJob, "project_value": "10000", "assigned_approver": "6bq4j0eb26631dy", "version": fmt.Sprint(rejected.GetInt("version"))}
	workflowUploadMethod(t, app, http.MethodPatch, "/api/job_attachments/"+id, workflowAuthor, "%PDF-1.4\nCorrection review document\n%%EOF", fields, http.StatusOK)
	after := workflowRecord(t, app, "job_attachments", id)
	if after.GetString("status") != "Pending" || after.GetString("rejection_reason") != "" || after.GetInt("version") <= rejected.GetInt("version") || after.GetString("review_token") == rejected.GetString("review_token") {
		t.Fatalf("corrected value change stayed rejected: status %s, reason %q", after.GetString("status"), after.GetString("rejection_reason"))
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/approve", workflowApprover, attachmentActionData(t, app, id), http.StatusOK)
}

// Changes made while a value change was pending are part of what the approver
// rejected. Saving it again without a further correction keeps it rejected.
func TestJobValueRejectedAfterPendingChangeNeedsLaterCorrection(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	id := createValueChange(t, app, legacyWorkflowJob, 10000, "Pending change review document")
	clientSetupJSON(t, app, http.MethodPatch, "/api/collections/client_contacts/records/184qbpajciwrrp2", workflowAuthor, map[string]any{"email": "changed-while-pending@example.com"}, http.StatusOK)
	rejection := attachmentRejectData(t, app, id, "The billing contact email is still wrong.")
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/reject", workflowApprover, rejection, http.StatusOK)
	rejected := workflowRecord(t, app, "job_attachments", id)
	fields := map[string]string{"job": legacyWorkflowJob, "project_value": "10000", "assigned_approver": "6bq4j0eb26631dy", "version": fmt.Sprint(rejected.GetInt("version"))}
	document := "%PDF-1.4\nPending change review document\n%%EOF"
	workflowUploadMethod(t, app, http.MethodPatch, "/api/job_attachments/"+id, workflowAuthor, document, fields, http.StatusOK)
	after := workflowRecord(t, app, "job_attachments", id)
	if after.GetString("status") != "Rejected" || after.GetString("rejection_reason") != rejected.GetString("rejection_reason") || after.GetInt("version") != rejected.GetInt("version") {
		t.Fatalf("change made before rejection cleared it: status %s", after.GetString("status"))
	}
	// A correction made after rejection returns it for approval.
	clientSetupJSON(t, app, http.MethodPatch, "/api/collections/client_contacts/records/184qbpajciwrrp2", workflowAuthor, map[string]any{"email": "corrected-after-rejection@example.com"}, http.StatusOK)
	workflowUploadMethod(t, app, http.MethodPatch, "/api/job_attachments/"+id, workflowAuthor, document, fields, http.StatusOK)
	if status := workflowRecord(t, app, "job_attachments", id).GetString("status"); status != "Pending" {
		t.Fatalf("correction after rejection left status %s", status)
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/approve", workflowApprover, attachmentActionData(t, app, id), http.StatusOK)
}

// The approver rejects the data the page showed. A change made after the page
// loaded needs a reload, so the rejection records only data the approver saw.
func TestJobValueRejectionRequiresTheDataTheApproverSaw(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	id := createValueChange(t, app, legacyWorkflowJob, 10000, "Viewed data review document")
	shown := attachmentRejectData(t, app, id, "The billing contact email is wrong.")
	clientSetupJSON(t, app, http.MethodPatch, "/api/collections/client_contacts/records/184qbpajciwrrp2", workflowAuthor, map[string]any{"email": "changed-after-loading@example.com"}, http.StatusOK)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/reject", workflowApprover, shown, http.StatusConflict)
	if status := workflowRecord(t, app, "job_attachments", id).GetString("status"); status != "Pending" {
		t.Fatalf("stale rejection changed status to %s", status)
	}
	delete(shown, "review_token")
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/reject", workflowApprover, shown, http.StatusConflict)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/reject", workflowApprover, attachmentRejectData(t, app, id, "The billing contact email is wrong."), http.StatusOK)
}

func TestJobValueApprovalRestoresExactHistoricalNegativeValue(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	// Simulate an older stored value. New proposals still reject negative values.
	if _, err := app.DB().NewQuery("UPDATE jobs SET project_value=-100 WHERE id='pawflegacy00001'").Execute(); err != nil {
		t.Fatal(err)
	}
	id := createValueChange(t, app, legacyWorkflowJob, 5000, "Negative history correction")
	if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetFloat("project_value") != -100 {
		t.Fatal("pending value changed historical amount")
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/approve", workflowApprover, attachmentActionData(t, app, id), http.StatusOK)
	if workflowRecord(t, app, "job_attachments", id).GetFloat("previous_project_value") != -100 {
		t.Fatal("approval did not retain exact negative value")
	}
	clientSetupJSON(t, app, http.MethodDelete, "/api/job_attachments/"+id, workflowAuthor, attachmentActionData(t, app, id), http.StatusNoContent)
	if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetFloat("project_value") != -100 {
		t.Fatal("admin cleanup did not restore exact historical negative value")
	}
}

// Model a migrated job whose old effective value and new billing field are empty.
// This changes only the existing CSV fixture; no records are seeded in test code.
func clearLegacyJobBilling(t *testing.T, app *tests.TestApp) {
	t.Helper()
	if _, err := app.DB().NewQuery("UPDATE jobs SET invoicing_information='' WHERE id='pawflegacy00001'").Execute(); err != nil {
		t.Fatal(err)
	}
}

// An older project with no value and no invoicing profile gets its profile in
// the job editor, then a value through approval. Approval keeps the historical
// zero as the previous value.
func TestLegacyZeroValueCompletesBillingInEditorThenProposesValue(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	clearLegacyJobBilling(t, app)
	fields := map[string]string{"job": legacyWorkflowJob, "project_value": "10000", "assigned_approver": "6bq4j0eb26631dy"}
	document := "%PDF-1.4\nLegacy zero billing\n%%EOF"
	workflowUpload(t, app, "/api/job_attachments", workflowAuthor, document, fields, http.StatusBadRequest)
	if count, err := app.CountRecords("job_attachments", dbx.HashExp{"job": legacyWorkflowJob}); err != nil || count != 0 {
		t.Fatalf("refused value change was retained: %v %d", err, count)
	}
	edit := func(job string, status int) {
		t.Helper()
		body := `{"job":` + job + `,"allocations":[{"division":"fy4i9poneukvq9u","hours":10}]}`
		response := performTestAPIRequest(t, app, http.MethodPut, "/api/jobs/"+legacyWorkflowJob, strings.NewReader(body), map[string]string{"Authorization": workflowToken(t, app, workflowAuthor), "Content-Type": "application/json"})
		mustStatus(t, response, status)
	}
	// Making the zero-value project Closed still requires a value.
	edit(`{"status":"Closed"}`, http.StatusBadRequest)
	edit(`{"invoicing_information":"painvoice000001"}`, http.StatusOK)
	job := workflowRecord(t, app, "jobs", legacyWorkflowJob)
	if job.GetString("invoicing_information") != "painvoice000001" || job.GetFloat("project_value") != 0 || job.GetString("status") != "Active" {
		t.Fatal("editor did not save billing on the zero-value project")
	}
	response := workflowUpload(t, app, "/api/job_attachments", workflowAuthor, document, fields, http.StatusOK)
	result := map[string]any{}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	id, _ := result["id"].(string)
	if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetFloat("project_value") != 0 {
		t.Fatal("pending change replaced effective value")
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/approve", workflowApprover, attachmentActionData(t, app, id), http.StatusOK)
	if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetFloat("project_value") != 10000 {
		t.Fatal("approval did not apply proposed value")
	}
	if workflowRecord(t, app, "job_attachments", id).GetFloat("previous_project_value") != 0 {
		t.Fatal("approval lost historical zero")
	}
}

// Many older projects have no authorizing document. Their value change saves
// without one, and its approval records PA as the project's authorization.
func TestOlderProjectWithoutAuthorizingDocumentGetsPAOnApproval(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	if _, err := app.DB().NewQuery("UPDATE jobs SET authorizing_document='' WHERE id='pawflegacy00001'").Execute(); err != nil {
		t.Fatal(err)
	}
	id := createValueChange(t, app, legacyWorkflowJob, 10000, "No authorizing document")
	if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetString("authorizing_document") != "" {
		t.Fatal("pending change set the authorizing document")
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/approve", workflowApprover, attachmentActionData(t, app, id), http.StatusOK)
	job := workflowRecord(t, app, "jobs", legacyWorkflowJob)
	if job.GetFloat("project_value") != 10000 || job.GetString("authorizing_document") != "PA" {
		t.Fatalf("approval left value %v and authorizing document %q", job.GetFloat("project_value"), job.GetString("authorizing_document"))
	}
}

// A rejected value change is corrected in the job editor. Saving the job
// without a change leaves the value change rejected; a real correction returns
// it for approval.
func TestJobValueRejectedThenCorrectedInEditor(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	id := createValueChange(t, app, legacyWorkflowJob, 10000, "Editor correction document")
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/reject", workflowApprover, attachmentRejectData(t, app, id, "Fix the description."), http.StatusOK)
	rejected := workflowRecord(t, app, "job_attachments", id)
	rows, err := app.FindRecordsByFilter("job_time_allocations", "job={:id}", "division", 0, 0, dbx.Params{"id": legacyWorkflowJob})
	if err != nil {
		t.Fatal(err)
	}
	allocations := []map[string]any{}
	for _, row := range rows {
		allocations = append(allocations, map[string]any{"division": row.GetString("division"), "hours": row.GetFloat("hours")})
	}
	edit := func(description string) {
		t.Helper()
		body, err := json.Marshal(map[string]any{"job": map[string]any{"description": description}, "allocations": allocations})
		if err != nil {
			t.Fatal(err)
		}
		response := performTestAPIRequest(t, app, http.MethodPut, "/api/jobs/"+legacyWorkflowJob, strings.NewReader(string(body)), map[string]string{"Authorization": workflowToken(t, app, workflowAuthor), "Content-Type": "application/json"})
		mustStatus(t, response, http.StatusOK)
	}
	fields := map[string]string{"job": legacyWorkflowJob, "project_value": "10000", "assigned_approver": "6bq4j0eb26631dy", "version": fmt.Sprint(rejected.GetInt("version"))}
	document := "%PDF-1.4\nEditor correction document\n%%EOF"
	// The editor saves the same data, replacing the allocation rows.
	edit(workflowRecord(t, app, "jobs", legacyWorkflowJob).GetString("description"))
	workflowUploadMethod(t, app, http.MethodPatch, "/api/job_attachments/"+id, workflowAuthor, document, fields, http.StatusOK)
	if after := workflowRecord(t, app, "job_attachments", id); after.GetString("status") != "Rejected" || after.GetInt("version") != rejected.GetInt("version") {
		t.Fatalf("job save without a change cleared the rejection: status %s", after.GetString("status"))
	}
	edit("Corrected description")
	workflowUploadMethod(t, app, http.MethodPatch, "/api/job_attachments/"+id, workflowAuthor, document, fields, http.StatusOK)
	if status := workflowRecord(t, app, "job_attachments", id).GetString("status"); status != "Pending" {
		t.Fatalf("correction in the job editor left status %s", status)
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/approve", workflowApprover, attachmentActionData(t, app, id), http.StatusOK)
}

// A refused value change leaves the job, its allocations, and its history as
// they were.
func TestJobValueSaveFailuresLeaveNothing(t *testing.T) {
	duplicate, err := os.ReadFile("test_seed_data/files/job_attachments/paattachment001/authorization.pdf")
	if err != nil {
		t.Fatal(err)
	}
	scenarios := []struct {
		name, email, document string
		incomplete            bool
		status                int
	}{
		{"unauthorized editor", workflowApprover, "%PDF-1.4\nUnauthorized\n%%EOF", false, http.StatusForbidden},
		{"missing invoicing profile", workflowAuthor, "%PDF-1.4\nIncomplete billing\n%%EOF", true, http.StatusBadRequest},
		{"duplicate PDF", workflowAuthor, string(duplicate), false, http.StatusBadRequest},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			app := newWorkflowTestApp(t, false)
			if scenario.incomplete {
				clearLegacyJobBilling(t, app)
			}
			before := workflowRecord(t, app, "jobs", legacyWorkflowJob)
			fields := map[string]string{"job": legacyWorkflowJob, "project_value": "10000", "assigned_approver": "6bq4j0eb26631dy"}
			workflowUpload(t, app, "/api/job_attachments", scenario.email, scenario.document, fields, scenario.status)
			after := workflowRecord(t, app, "jobs", legacyWorkflowJob)
			for _, field := range []string{"invoicing_information", "description", "client", "contact", "updated", "project_value"} {
				if before.GetString(field) != after.GetString(field) {
					t.Fatalf("failed submission changed %s", field)
				}
			}
			if count, err := app.CountRecords("job_attachments", dbx.HashExp{"job": legacyWorkflowJob}); err != nil || count != 0 {
				t.Fatalf("failed submission retained attachment: %v %d", err, count)
			}
		})
	}
}

// Project details are edited in the job editor. Any sent with a value change
// are ignored, so they cannot change the client, value, or billing.
func TestJobValueSaveIgnoresProjectDetails(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	before := workflowRecord(t, app, "jobs", legacyWorkflowJob)
	fields := map[string]string{"job": legacyWorkflowJob, "project_value": "10000", "assigned_approver": "6bq4j0eb26631dy", "job_updates": `{"client":"lb0fnenkeyitsny","project_value":7000,"description":"Changed"}`, "allocations": `[{"division":"fy4i9poneukvq9u","hours":99}]`}
	workflowUpload(t, app, "/api/job_attachments", workflowAuthor, "%PDF-1.4\nIgnored details\n%%EOF", fields, http.StatusOK)
	after := workflowRecord(t, app, "jobs", legacyWorkflowJob)
	for _, field := range []string{"client", "project_value", "description", "invoicing_information"} {
		if before.GetString(field) != after.GetString(field) {
			t.Fatalf("value change changed %s", field)
		}
	}
	if hours := workflowRecord(t, app, "job_time_allocations", "pawflegacyalloc").GetFloat("hours"); hours != 10 {
		t.Fatalf("value change changed allocation hours to %v", hours)
	}
}

func TestJobValueSavesAndApprovalsRequireValidStoredAllocations(t *testing.T) {
	for _, invalid := range []string{"missing", "inactive"} {
		for _, stage := range []string{"save", "approve"} {
			t.Run(invalid+"_"+stage, func(t *testing.T) {
				app := newWorkflowTestApp(t, false)
				id := ""
				if stage == "approve" {
					id = createValueChange(t, app, legacyWorkflowJob, 10000, "Stored allocation "+invalid)
				}
				// Alter existing fixtures to represent allocations that were removed or made inactive.
				query := "DELETE FROM job_time_allocations WHERE job='pawflegacy00001'"
				if invalid == "inactive" {
					query = "UPDATE divisions SET active=0 WHERE id='fy4i9poneukvq9u'"
				}
				if _, err := app.DB().NewQuery(query).Execute(); err != nil {
					t.Fatal(err)
				}
				if stage == "save" {
					workflowUpload(t, app, "/api/job_attachments", workflowAuthor, "%PDF-1.4\nInvalid stored allocation\n%%EOF", map[string]string{"job": legacyWorkflowJob, "project_value": "10000", "assigned_approver": "6bq4j0eb26631dy"}, http.StatusBadRequest)
				} else {
					response := performJSONRequest(t, app, http.MethodPost, "/api/job_attachments/"+id+"/approve", workflowToken(t, app, workflowApprover), attachmentActionData(t, app, id))
					if response.Code != http.StatusBadRequest && response.Code != http.StatusConflict {
						t.Fatalf("invalid stored allocation approval status=%d; body=%s", response.Code, response.Body.String())
					}
					if workflowRecord(t, app, "job_attachments", id).GetString("status") != "Pending" {
						t.Fatal("failed approval changed evidence")
					}
				}
				if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetFloat("project_value") != 0 {
					t.Fatal("invalid stored allocations changed effective value")
				}
			})
		}
	}
}

// Each listed value change shows related data and a review token from the
// same read, even while that data is being edited.
func TestJobValueListTokenMatchesShownDataDuringEdits(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	id := createValueChange(t, app, legacyWorkflowJob, 10000, "Concurrent review document")
	profile := workflowRecord(t, app, "jobs", legacyWorkflowJob).GetString("invoicing_information")
	setInstructions := func(text string) {
		if _, err := app.DB().NewQuery("UPDATE client_invoicing_information SET invoicing_instructions = {:text} WHERE id = {:id}").Bind(dbx.Params{"text": text, "id": profile}).Execute(); err != nil {
			t.Error(err)
		}
	}
	listed := func() (string, string) {
		list := clientSetupJSON(t, app, http.MethodGet, "/api/job_attachments?job="+legacyWorkflowJob, workflowApprover, nil, http.StatusOK)
		for _, item := range list["items"].([]any) {
			if row := item.(map[string]any); row["id"] == id {
				billing := row["related"].(map[string]any)["invoicing_information"].(map[string]any)
				return billing["invoicing_instructions"].(string), row["current_review_token"].(string)
			}
		}
		t.Fatal("value change not listed")
		return "", ""
	}
	tokens := map[string]string{}
	for _, text := range []string{"Instructions A", "Instructions B"} {
		setInstructions(text)
		_, tokens[text] = listed()
	}
	stop, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
				setInstructions([]string{"Instructions A", "Instructions B"}[i%2])
			}
		}
	}()
	defer func() {
		close(stop)
		<-done
	}()
	for i := 0; i < 200; i++ {
		shown, token := listed()
		if tokens[shown] != token {
			t.Fatalf("listed %q with the review token of other data", shown)
		}
	}
}

// A project's manager and alternate manager can see its value history and
// PDFs without the job claim. Other users without a claim still cannot.
func TestJobValueHistoryVisibleToProjectManagers(t *testing.T) {
	for _, field := range []string{"manager", "alternate_manager"} {
		t.Run(field, func(t *testing.T) {
			app := newWorkflowTestApp(t, false)
			id := createValueChange(t, app, legacyWorkflowJob, 10000, "Manager view "+field)
			// Tester Time holds time but not job.
			if _, err := app.DB().NewQuery("UPDATE jobs SET " + field + "='rzr98oadsp9qc11' WHERE id={:id}").Bind(dbx.Params{"id": legacyWorkflowJob}).Execute(); err != nil {
				t.Fatal(err)
			}
			list := clientSetupJSON(t, app, http.MethodGet, "/api/job_attachments?job="+legacyWorkflowJob, "time@test.com", nil, http.StatusOK)
			if items := list["items"].([]any); len(items) != 1 || items[0].(map[string]any)["id"] != id {
				t.Fatalf("manager history = %#v", items)
			}
			mustStatus(t, performJSONRequest(t, app, http.MethodGet, "/api/job_attachments/"+id+"/document", workflowToken(t, app, "time@test.com"), nil), http.StatusOK)

			hidden := clientSetupJSON(t, app, http.MethodGet, "/api/job_attachments?job="+legacyWorkflowJob, "u_no_claims@example.com", nil, http.StatusOK)
			if items := hidden["items"].([]any); len(items) != 0 {
				t.Fatalf("unrelated user sees history: %#v", items)
			}
			clientSetupJSON(t, app, http.MethodGet, "/api/job_attachments/"+id+"/document", "u_no_claims@example.com", nil, http.StatusForbidden)
		})
	}
}

// Job details report the approved project value, which the job page shows.
func TestJobDetailsReturnApprovedProjectValue(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	id := createValueChange(t, app, legacyWorkflowJob, 10000, "Details value")
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/approve", workflowApprover, attachmentActionData(t, app, id), http.StatusOK)
	details := clientSetupJSON(t, app, http.MethodGet, "/api/jobs/"+legacyWorkflowJob+"/details", workflowAuthor, nil, http.StatusOK)
	if details["project_value"] != float64(10000) {
		t.Fatalf("project_value = %#v; want 10000", details["project_value"])
	}
}
