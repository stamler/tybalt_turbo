package main

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestJobValueDeleteRequiresCurrentReview(t *testing.T) {
	for _, change := range []string{"approve", "reject", "edit", "missing version", "wrong hash"} {
		t.Run(change, func(t *testing.T) {
			app := newWorkflowTestApp(t, false)
			pdf := authorizationPDF(t, "authorization-upload.pdf")
			fields := map[string]string{"job": legacyWorkflowJob, "project_value": "10000", "assigned_approver": "6bq4j0eb26631dy"}
			workflowUpload(t, app, "/api/job_attachments", workflowAuthor, string(pdf), fields, http.StatusOK)
			attachment, err := app.FindFirstRecordByData("job_attachments", "job", legacyWorkflowJob)
			if err != nil {
				t.Fatal(err)
			}
			id := attachment.Id
			stale := attachmentActionData(t, app, id)
			version := attachment.GetInt("version")
			switch change {
			case "approve", "reject":
				action := attachmentActionData(t, app, id)
				action["rejection_reason"] = "Correct the proposed value."
				clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/"+change, workflowApprover, action, http.StatusOK)
			case "edit":
				fields["version"] = fmt.Sprint(version)
				fields["project_value"] = "9000"
				workflowUploadMethod(t, app, http.MethodPatch, "/api/job_attachments/"+id, workflowAuthor, string(pdf), fields, http.StatusOK)
			case "missing version":
				delete(stale, "version")
			case "wrong hash":
				stale["document_hash"] = "not the reviewed PDF"
			}
			attachment = workflowRecord(t, app, "job_attachments", id)
			if change == "approve" || change == "reject" || change == "edit" {
				if attachment.GetInt("version") != version+1 {
					t.Errorf("%s must advance the attachment version", change)
				}
			}
			before := attachment.PublicExport()
			jobBefore := workflowRecord(t, app, "jobs", legacyWorkflowJob).PublicExport()
			clientSetupJSON(t, app, http.MethodDelete, "/api/job_attachments/"+id, workflowAuthor, stale, http.StatusConflict)
			after := workflowRecord(t, app, "job_attachments", id)
			if !reflect.DeepEqual(before, after.PublicExport()) || !reflect.DeepEqual(jobBefore, workflowRecord(t, app, "jobs", legacyWorkflowJob).PublicExport()) {
				t.Fatal("stale deletion changed approval evidence or the effective job value")
			}
			assertAuthorizationBytes(t, app, after, pdf)

			// A new confirmation for the current record can perform the permitted deletion.
			clientSetupJSON(t, app, http.MethodDelete, "/api/job_attachments/"+id, workflowAuthor, attachmentActionData(t, app, id), http.StatusNoContent)
			if _, err := app.FindRecordById("job_attachments", id); err == nil {
				t.Fatal("confirmed deletion retained the attachment")
			}
			if workflowRecord(t, app, "jobs", legacyWorkflowJob).GetFloat("project_value") != 0 {
				t.Fatal("confirmed deletion did not retain or restore the original effective value")
			}
		})
	}
}
