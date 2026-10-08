package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

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

func TestLegacyZeroValueCanCompleteBillingWithPendingApproval(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	clearLegacyJobBilling(t, app)
	fields := map[string]string{"job": legacyWorkflowJob, "project_value": "10000", "assigned_approver": "6bq4j0eb26631dy"}
	document := "%PDF-1.4\nLegacy zero billing correction\n%%EOF"
	workflowUpload(t, app, "/api/job_attachments", workflowAuthor, document, fields, http.StatusBadRequest)
	if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetFloat("project_value") != 0 {
		t.Fatal("incomplete save changed effective value")
	}
	fields["job_updates"] = `{"invoicing_information":"painvoice000001","contact":"184qbpajciwrrp2","description":"Legacy project billing completed"}`
	fields["job_updated"] = workflowRecord(t, app, "jobs", legacyWorkflowJob).GetString("updated")
	fields["job_review_token"] = valueContextToken(t, app, legacyWorkflowJob)
	fields["allocations"] = `[{"division":"fy4i9poneukvq9u","hours":20}]`
	savedUpdated := fields["job_updated"]
	fields["job_updated"] = "2000-01-01 00:00:00.000Z"
	workflowUpload(t, app, "/api/job_attachments", workflowAuthor, document, fields, http.StatusConflict)
	if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetString("invoicing_information") != "" {
		t.Fatal("stale correction changed billing")
	}
	fields["job_updated"] = savedUpdated
	response := workflowUpload(t, app, "/api/job_attachments", workflowAuthor, document, fields, http.StatusOK)
	result := map[string]any{}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	id, _ := result["id"].(string)
	if id == "" {
		t.Fatal("pending response has no id")
	}
	job := workflowRecord(t, app, "jobs", legacyWorkflowJob)
	if job.GetFloat("project_value") != 0 || job.GetString("invoicing_information") != "painvoice000001" || job.GetString("description") != "Legacy project billing completed" {
		t.Fatal("pending save did not keep old value and complete billing")
	}
	allocations, err := app.FindRecordsByFilter("job_time_allocations", "job='pawflegacy00001'", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(allocations) != 1 || allocations[0].GetFloat("hours") != 20 {
		t.Fatal("pending save did not update allocations")
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/approve", workflowApprover, attachmentActionData(t, app, id), http.StatusOK)
	if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetFloat("project_value") != 10000 {
		t.Fatal("approval did not apply proposed value")
	}
	if workflowRecord(t, app, "job_attachments", id).GetFloat("previous_project_value") != 0 {
		t.Fatal("approval lost historical zero")
	}
}

func TestLegacyBillingAndValueSubmissionFailuresRollBackTogether(t *testing.T) {
	duplicate, err := os.ReadFile("test_seed_data/files/job_attachments/paattachment001/authorization.pdf")
	if err != nil {
		t.Fatal(err)
	}
	scenarios := []struct {
		name, email, updates, allocations, document string
		status                                      int
	}{
		{"unauthorized editor", workflowApprover, `{"invoicing_information":"painvoice000001"}`, "", "%PDF-1.4\nUnauthorized billing edit\n%%EOF", http.StatusForbidden},
		{"direct value bypass", workflowAuthor, `{"invoicing_information":"painvoice000001","project_value":7000}`, "", "%PDF-1.4\nValue bypass\n%%EOF", http.StatusBadRequest},
		{"client reassignment", workflowAuthor, `{"invoicing_information":"painvoice000001","client":"lb0fnenkeyitsny"}`, "", "%PDF-1.4\nClient bypass\n%%EOF", http.StatusBadRequest},
		{"wrong client contact", workflowAuthor, `{"invoicing_information":"painvoice000001","contact":"nh5u9z3cyknjclv"}`, "", "%PDF-1.4\nWrong billing contact\n%%EOF", http.StatusBadRequest},
		{"invalid allocation", workflowAuthor, `{"invoicing_information":"painvoice000001","description":"Must roll back"}`, `[{"division":"missingdivision","hours":20}]`, "%PDF-1.4\nInvalid allocation\n%%EOF", http.StatusBadRequest},
		{"part-hour allocation", workflowAuthor, `{"invoicing_information":"painvoice000001","description":"Must roll back"}`, `[{"division":"fy4i9poneukvq9u","hours":0.25}]`, "%PDF-1.4\nPart-hour allocation\n%%EOF", http.StatusBadRequest},
		{"duplicate PDF", workflowAuthor, `{"invoicing_information":"painvoice000001","description":"Must roll back"}`, `[{"division":"fy4i9poneukvq9u","hours":20}]`, string(duplicate), http.StatusBadRequest},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			app := newWorkflowTestApp(t, false)
			clearLegacyJobBilling(t, app)
			before := workflowRecord(t, app, "jobs", legacyWorkflowJob)
			fields := map[string]string{"job": legacyWorkflowJob, "project_value": "10000", "assigned_approver": "6bq4j0eb26631dy", "job_updates": scenario.updates, "job_updated": before.GetString("updated"), "job_review_token": valueContextToken(t, app, legacyWorkflowJob)}
			if scenario.allocations != "" {
				fields["allocations"] = scenario.allocations
			}
			workflowUpload(t, app, "/api/job_attachments", scenario.email, scenario.document, fields, scenario.status)
			after := workflowRecord(t, app, "jobs", legacyWorkflowJob)
			for _, field := range []string{"invoicing_information", "description", "client", "contact", "updated"} {
				if before.GetString(field) != after.GetString(field) {
					t.Fatalf("failed submission changed %s", field)
				}
			}
			if after.GetFloat("project_value") != 0 {
				t.Fatal("failed submission changed effective value")
			}
			allocations, err := app.FindRecordsByFilter("job_time_allocations", "job='pawflegacy00001'", "", 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			if len(allocations) != 1 || allocations[0].GetFloat("hours") != 10 {
				t.Fatal("failed submission changed allocations")
			}
			attachments, err := app.FindRecordsByFilter("job_attachments", "job='pawflegacy00001'", "", 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			if len(attachments) != 0 {
				t.Fatal("failed submission retained attachment")
			}
		})
	}
}

func valueContextToken(t *testing.T, app *tests.TestApp, job string) string {
	t.Helper()
	context := clientSetupJSON(t, app, http.MethodGet, "/api/job_attachments/context?job="+job, workflowAuthor, nil, http.StatusOK)
	token, ok := context["review_token"].(string)
	if !ok || token == "" {
		t.Fatalf("missing value review token: %#v", context)
	}
	return token
}

func TestJobValueCorrectionContextRequiresJobClaimAndIsReadOnly(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	before := workflowRecord(t, app, "jobs", legacyWorkflowJob)
	path := "/api/job_attachments/context?job=" + legacyWorkflowJob
	clientSetupJSON(t, app, http.MethodGet, path, "", nil, http.StatusUnauthorized)
	clientSetupJSON(t, app, http.MethodGet, path, workflowApprover, nil, http.StatusForbidden)
	result := clientSetupJSON(t, app, http.MethodGet, path, workflowAuthor, nil, http.StatusOK)
	job, ok := result["job"].(map[string]any)
	if !ok || job["id"] != legacyWorkflowJob {
		t.Fatal("context does not contain selected job")
	}
	allocations, ok := result["allocations"].([]any)
	if !ok || len(allocations) != 1 {
		t.Fatal("context does not contain saved allocations")
	}
	token, ok := result["review_token"].(string)
	if !ok || token == "" {
		t.Fatal("context has no review token")
	}
	if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetString("updated") != before.GetString("updated") {
		t.Fatal("reading correction context changed job")
	}
}

func TestJobValueCorrectionsRejectStaleAllocationContext(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	clearLegacyJobBilling(t, app)
	before := workflowRecord(t, app, "jobs", legacyWorkflowJob)
	fields := map[string]string{"job": legacyWorkflowJob, "project_value": "10000", "assigned_approver": "6bq4j0eb26631dy", "job_updates": `{"invoicing_information":"painvoice000001"}`, "job_updated": before.GetString("updated"), "job_review_token": valueContextToken(t, app, legacyWorkflowJob)}
	// Change the saved allocation without touching jobs.updated, as a separate edit can do.
	if _, err := app.DB().NewQuery("UPDATE job_time_allocations SET hours=15 WHERE id='pawflegacyalloc'").Execute(); err != nil {
		t.Fatal(err)
	}
	if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetString("updated") != fields["job_updated"] {
		t.Fatal("fixture mutation unexpectedly changed job timestamp")
	}
	document := "%PDF-1.4\nStale allocation correction\n%%EOF"
	workflowUpload(t, app, "/api/job_attachments", workflowAuthor, document, fields, http.StatusConflict)
	if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetString("invoicing_information") != "" {
		t.Fatal("stale correction changed billing")
	}
	if workflowRecord(t, app, "job_time_allocations", "pawflegacyalloc").GetFloat("hours") != 15 {
		t.Fatal("stale correction changed concurrent allocation edit")
	}
	fields["job_review_token"] = valueContextToken(t, app, legacyWorkflowJob)
	workflowUpload(t, app, "/api/job_attachments", workflowAuthor, document, fields, http.StatusOK)
	if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetFloat("project_value") != 0 {
		t.Fatal("fresh pending correction replaced effective value")
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
