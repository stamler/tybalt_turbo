package main

import (
	"net/http"
	"reflect"
	"regexp"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// These tests use the CSV request, allocation, attachment, and authority records.
// State changes go through workflow APIs unless a test documents corruption.
const submissionTokenRequest = "parequest000001"
const submissionTokenCreator = "time@test.com"

func TestJobRequestSubmissionTokenRejectsOldReviewPages(t *testing.T) {
	for _, approvedBeforeRecall := range []bool{false, true} {
		name := "recall before approval"
		if approvedBeforeRecall {
			name = "recall after approval"
		}
		t.Run(name, func(t *testing.T) {
			app := newWorkflowTestApp(t, true)
			path := "/api/job_requests/" + submissionTokenRequest
			initial := workflowRecord(t, app, "job_requests", submissionTokenRequest)
			if initial.GetString("submission_token") != "" {
				t.Fatal("draft fixture must have no submission token")
			}
			clientSetupJSON(t, app, http.MethodPost, path+"/submit", submissionTokenCreator, requestActionData(t, app, submissionTokenRequest), http.StatusOK)
			first := workflowRecord(t, app, "job_requests", submissionTokenRequest).GetString("submission_token")
			if !regexp.MustCompile(`^[a-z0-9]{32}$`).MatchString(first) {
				t.Fatalf("invalid submission token: %q", first)
			}
			oldReview := requestActionData(t, app, submissionTokenRequest)
			if approvedBeforeRecall {
				clientSetupJSON(t, app, http.MethodPost, path+"/approve", workflowApprover, oldReview, http.StatusOK)
			}
			clientSetupJSON(t, app, http.MethodPost, path+"/recall", submissionTokenCreator, requestActionData(t, app, submissionTokenRequest), http.StatusOK)
			recalled := workflowRecord(t, app, "job_requests", submissionTokenRequest)
			attachment := workflowRecord(t, app, "job_attachments", "paattachment001")
			if recalled.GetString("submission_token") != "" || attachment.GetString("approver") != "" || attachment.GetString("review_token") != "" || attachment.GetInt("checklist_version") != 0 {
				t.Fatal("recall must clear the submission identity and saved approval together")
			}
			// No field, PDF, allocation, or business version changes between submissions.
			clientSetupJSON(t, app, http.MethodPost, path+"/submit", submissionTokenCreator, requestActionData(t, app, submissionTokenRequest), http.StatusOK)
			current := workflowRecord(t, app, "job_requests", submissionTokenRequest)
			second := current.GetString("submission_token")
			if second == first || !regexp.MustCompile(`^[a-z0-9]{32}$`).MatchString(second) {
				t.Fatal("resubmission must issue a different random identity")
			}
			if current.GetInt("version") != initial.GetInt("version") || current.GetString("last_edited") != initial.GetString("last_edited") {
				t.Fatal("submission identity must not change business version or abandonment age")
			}
			currentReview := requestActionData(t, app, submissionTokenRequest)
			if currentReview["review_token"] == oldReview["review_token"] {
				t.Fatal("review digest did not change with the submission identity")
			}
			clientSetupJSON(t, app, http.MethodPost, path+"/approve", workflowApprover, oldReview, http.StatusConflict)
			if workflowRecord(t, app, "job_attachments", attachment.Id).GetString("status") != "Pending" {
				t.Fatal("old browser page changed the pending approval")
			}
			// A new page still needs all six confirmations; a current token alone
			// is not approval authority or a substitute for the checklist.
			unchecked := requestActionData(t, app, submissionTokenRequest)
			unchecked["checklist_answers"] = map[string]any{}
			clientSetupJSON(t, app, http.MethodPost, path+"/approve", workflowApprover, unchecked, http.StatusBadRequest)
			clientSetupJSON(t, app, http.MethodPost, path+"/approve", workflowApprover, currentReview, http.StatusOK)
			clientSetupJSON(t, app, http.MethodGet, path+"/print", submissionTokenCreator, nil, http.StatusOK)
			if workflowRecord(t, app, "job_requests", submissionTokenRequest).GetString("submission_token") != second {
				t.Fatal("approval or print rotated the submission token")
			}
			clientSetupJSON(t, app, http.MethodPost, path+"/create", workflowAuthor, oldReview, http.StatusConflict)
			clientSetupJSON(t, app, http.MethodPost, path+"/create", workflowAuthor, requestActionData(t, app, submissionTokenRequest), http.StatusOK)
			attachment = workflowRecord(t, app, "job_attachments", attachment.Id)
			if attachment.GetString("review_token") != currentReview["review_token"] || attachment.GetInt("checklist_version") != 1 || attachment.GetString("job") == "" {
				t.Fatal("creation did not retain the final submission's approved digest and checklist")
			}
		})
	}
}

func TestJobRequestSubmissionTokenClearedOnRejection(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	path := "/api/job_requests/" + submissionTokenRequest
	clientSetupJSON(t, app, http.MethodPost, path+"/submit", submissionTokenCreator, requestActionData(t, app, submissionTokenRequest), http.StatusOK)
	first := workflowRecord(t, app, "job_requests", submissionTokenRequest).GetString("submission_token")
	input := requestActionData(t, app, submissionTokenRequest)
	clientSetupJSON(t, app, http.MethodPost, path+"/reject", workflowApprover, input, http.StatusBadRequest)
	if workflowRecord(t, app, "job_requests", submissionTokenRequest).GetString("submission_token") != first {
		t.Fatal("failed rejection cleared the live token")
	}
	input["rejection_reason"] = "Correct the description."
	clientSetupJSON(t, app, http.MethodPost, path+"/reject", workflowApprover, input, http.StatusOK)
	if workflowRecord(t, app, "job_requests", submissionTokenRequest).GetString("submission_token") != "" {
		t.Fatal("rejection retained a live submission token")
	}
	clientSetupJSON(t, app, http.MethodPost, path+"/submit", submissionTokenCreator, requestActionData(t, app, submissionTokenRequest), http.StatusBadRequest)
	request := workflowRecord(t, app, "job_requests", submissionTokenRequest)
	clientSetupJSON(t, app, http.MethodPatch, path, submissionTokenCreator, map[string]any{"version": request.GetInt("version"), "request": map[string]any{"description": "Corrected project description"}}, http.StatusOK)
	if workflowRecord(t, app, "job_requests", submissionTokenRequest).GetString("submission_token") != "" {
		t.Fatal("business edit issued a submission token")
	}
	clientSetupJSON(t, app, http.MethodPost, path+"/submit", submissionTokenCreator, requestActionData(t, app, submissionTokenRequest), http.StatusOK)
	if next := workflowRecord(t, app, "job_requests", submissionTokenRequest).GetString("submission_token"); next == "" || next == first {
		t.Fatal("corrected resubmission reused the rejected submission's identity")
	}
}

func TestJobRequestSubmissionTokenRollsBackWithWorkflow(t *testing.T) {
	for _, action := range []string{"submit", "recall"} {
		t.Run(action, func(t *testing.T) {
			app := newWorkflowTestApp(t, true)
			path := "/api/job_requests/" + submissionTokenRequest
			if action == "recall" {
				clientSetupJSON(t, app, http.MethodPost, path+"/submit", submissionTokenCreator, requestActionData(t, app, submissionTokenRequest), http.StatusOK)
				clientSetupJSON(t, app, http.MethodPost, path+"/approve", workflowApprover, requestActionData(t, app, submissionTokenRequest), http.StatusOK)
			}
			before := workflowRecord(t, app, "job_requests", submissionTokenRequest).PublicExport()
			approvalBefore := workflowRecord(t, app, "job_attachments", "paattachment001").PublicExport()
			// Fail after the route has changed the token and saved the attachment
			// within the transaction. No source fixture or production hook changes.
			fail := true
			app.OnRecordUpdate("job_requests").BindFunc(func(e *core.RecordEvent) error {
				if fail && e.Record.Id == submissionTokenRequest {
					return apis.NewBadRequestError("Simulated request save failure.", nil)
				}
				return e.Next()
			})
			input := requestActionData(t, app, submissionTokenRequest)
			clientSetupJSON(t, app, http.MethodPost, path+"/"+action, submissionTokenCreator, input, http.StatusBadRequest)
			if !reflect.DeepEqual(before, workflowRecord(t, app, "job_requests", submissionTokenRequest).PublicExport()) || !reflect.DeepEqual(approvalBefore, workflowRecord(t, app, "job_attachments", "paattachment001").PublicExport()) {
				t.Fatal("failed action did not roll back token, status, and approval together")
			}
			fail = false
			clientSetupJSON(t, app, http.MethodPost, path+"/"+action, submissionTokenCreator, input, http.StatusOK)
		})
	}
}

func TestJobRequestSubmissionTokenCannotBeSuppliedByCaller(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	request := workflowRecord(t, app, "job_requests", submissionTokenRequest)
	clientSetupJSON(t, app, http.MethodPatch, "/api/job_requests/"+submissionTokenRequest, submissionTokenCreator, map[string]any{"version": request.GetInt("version"), "request": map[string]any{"submission_token": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}, http.StatusBadRequest)
	if workflowRecord(t, app, "job_requests", submissionTokenRequest).GetString("submission_token") != "" {
		t.Fatal("caller set the server-owned submission token")
	}
}

func TestJobRequestSubmissionTokenRequiredForApprovalAndCreation(t *testing.T) {
	for _, action := range []string{"approve", "create"} {
		t.Run(action, func(t *testing.T) {
			app := newWorkflowTestApp(t, true)
			path := "/api/job_requests/" + submissionTokenRequest
			clientSetupJSON(t, app, http.MethodPost, path+"/submit", submissionTokenCreator, requestActionData(t, app, submissionTokenRequest), http.StatusOK)
			actor := workflowApprover
			if action == "create" {
				clientSetupJSON(t, app, http.MethodPost, path+"/approve", actor, requestActionData(t, app, submissionTokenRequest), http.StatusOK)
				actor = workflowAuthor
			}
			// Simulate corrupt persisted state. Normal APIs cannot clear this field
			// while retaining a submitted/approved state; neither action may accept it.
			if _, err := app.DB().NewQuery("UPDATE job_requests SET submission_token = '' WHERE id = 'parequest000001'").Execute(); err != nil {
				t.Fatal(err)
			}
			clientSetupJSON(t, app, http.MethodPost, path+"/"+action, actor, requestActionData(t, app, submissionTokenRequest), http.StatusConflict)
		})
	}
}
