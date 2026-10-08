package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func authorizationPDF(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("test_seed_data/documents/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertAuthorizationBytes(t *testing.T, app *tests.TestApp, attachment *core.Record, want []byte) {
	t.Helper()
	if got, expected := attachment.GetString("document_hash"), fmt.Sprintf("%x", sha256.Sum256(want)); got != expected {
		t.Fatalf("stored hash = %s, want SHA-256 of uploaded bytes %s", got, expected)
	}
	fs, err := app.NewFilesystem()
	if err != nil {
		t.Fatal(err)
	}
	defer fs.Close()
	reader, err := fs.GetReader(attachment.BaseFilesPath() + "/" + attachment.GetString("document"))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("stored PDF differs from uploaded bytes")
	}
}

func TestProjectAuthorizationSeedHashMatchesFile(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	data, err := os.ReadFile("test_seed_data/files/job_attachments/paattachment001/authorization.pdf")
	if err != nil {
		t.Fatal(err)
	}
	assertAuthorizationBytes(t, app, workflowRecord(t, app, "job_attachments", "paattachment001"), data)
}

func TestProjectAuthorizationUploadAndReplacementHashes(t *testing.T) {
	for _, kind := range []string{"request", "value"} {
		t.Run(kind, func(t *testing.T) {
			app := newWorkflowTestApp(t, true)
			fields := map[string]string{"document_hash": strings.Repeat("0", 64)}
			path := "/api/job_attachments"
			requestID := ""
			if kind == "request" {
				requestID = createWorkflowRequest(t, app)
				path = "/api/job_requests/" + requestID + "/document"
				fields["version"] = "1"
			} else {
				fields["job"] = legacyWorkflowJob
				fields["project_value"] = "10000"
				fields["assigned_approver"] = "6bq4j0eb26631dy"
			}
			for index, name := range []string{"authorization-upload.pdf", "authorization-replacement.pdf"} {
				data := authorizationPDF(t, name)
				method := http.MethodPost
				var previous *core.Record
				if index > 0 {
					field, id := "job", legacyWorkflowJob
					if kind == "request" {
						field, id = "request", requestID
						fields["version"] = fmt.Sprint(workflowRecord(t, app, "job_requests", requestID).GetInt("version"))
					}
					var err error
					previous, err = app.FindFirstRecordByData("job_attachments", field, id)
					if err != nil {
						t.Fatal(err)
					}
					if kind == "value" {
						method, path = http.MethodPatch, path+"/"+previous.Id
						fields["version"] = fmt.Sprint(previous.GetInt("version"))
					}
				}
				workflowUploadMethod(t, app, method, path, workflowAuthor, string(data), fields, http.StatusOK)
				field, id := "job", legacyWorkflowJob
				if kind == "request" {
					field, id = "request", requestID
				}
				attachment, err := app.FindFirstRecordByData("job_attachments", field, id)
				if err != nil {
					t.Fatal(err)
				}
				assertAuthorizationBytes(t, app, attachment, data)
				if previous != nil && (attachment.Id != previous.Id || attachment.GetString("document_hash") == previous.GetString("document_hash")) {
					t.Fatal("replacement must keep the attachment ID and change its hash")
				}
			}
		})
	}
}

// Remove only the isolated test app's file to simulate a missing object.
// Do not change the recorded hash, review token, or source fixture.
func removeAuthorizationFile(t *testing.T, app *tests.TestApp, attachment *core.Record) func() {
	t.Helper()
	fs, err := app.NewFilesystem()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { fs.Close() })
	path := attachment.BaseFilesPath() + "/" + attachment.GetString("document")
	reader, err := fs.GetReader(path)
	if err != nil {
		t.Fatal(err)
	}
	original, err := io.ReadAll(reader)
	reader.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = fs.Delete(path)
	if err != nil {
		t.Fatal(err)
	}
	return func() {
		if err := fs.Upload(original, path); err != nil {
			t.Fatal(err)
		}
	}
}

func TestProjectAuthorizationActionsRejectMissingFiles(t *testing.T) {
	for _, action := range []string{"submit", "approve", "create", "value approval"} {
		t.Run(action, func(t *testing.T) {
			app := newWorkflowTestApp(t, true)
			const requestID = "parequest000001"
			attachment := workflowRecord(t, app, "job_attachments", "paattachment001")
			path, actor := "/api/job_requests/"+requestID+"/"+action, "time@test.com"
			if action == "value approval" {
				data := authorizationPDF(t, "authorization-upload.pdf")
				workflowUpload(t, app, "/api/job_attachments", workflowAuthor, string(data), map[string]string{"job": legacyWorkflowJob, "project_value": "10000", "assigned_approver": "6bq4j0eb26631dy"}, http.StatusOK)
				var err error
				attachment, err = app.FindFirstRecordByData("job_attachments", "job", legacyWorkflowJob)
				if err != nil {
					t.Fatal(err)
				}
				path, actor = "/api/job_attachments/"+attachment.Id+"/approve", workflowApprover
			} else if action != "submit" {
				clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+requestID+"/submit", actor, requestActionData(t, app, requestID), http.StatusOK)
				actor = workflowApprover
				if action == "create" {
					clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+requestID+"/approve", actor, requestActionData(t, app, requestID), http.StatusOK)
					actor = workflowAuthor
				}
				attachment = workflowRecord(t, app, "job_attachments", attachment.Id)
			}
			input := requestActionData(t, app, requestID)
			if action == "value approval" {
				input = attachmentActionData(t, app, attachment.Id)
			}
			requestBefore := workflowRecord(t, app, "job_requests", requestID).PublicExport()
			attachmentBefore := attachment.PublicExport()
			jobBefore := workflowRecord(t, app, "jobs", legacyWorkflowJob).PublicExport()
			countBefore, err := app.CountRecords("jobs")
			if err != nil {
				t.Fatal(err)
			}
			restore := removeAuthorizationFile(t, app, attachment)
			response := performJSONRequest(t, app, http.MethodPost, path, workflowToken(t, app, actor), input)
			mustStatus(t, response, http.StatusBadRequest)
			if !strings.Contains(response.Body.String(), "PDF is unavailable") {
				t.Fatalf("expected file-integrity failure: %s", response.Body.String())
			}
			countAfter, err := app.CountRecords("jobs")
			if err != nil {
				t.Fatal(err)
			}
			if countAfter != countBefore || !reflect.DeepEqual(requestBefore, workflowRecord(t, app, "job_requests", requestID).PublicExport()) || !reflect.DeepEqual(attachmentBefore, workflowRecord(t, app, "job_attachments", attachment.Id).PublicExport()) || !reflect.DeepEqual(jobBefore, workflowRecord(t, app, "jobs", legacyWorkflowJob).PublicExport()) {
				t.Fatal("failed file check changed workflow state")
			}
			restore()
			clientSetupJSON(t, app, http.MethodPost, path, actor, input, http.StatusOK)
		})
	}
}

func TestProjectAuthorizationMatchingUploadRestoresOnlyMissingFiles(t *testing.T) {
	for _, kind := range []string{"request", "value"} {
		for _, rejected := range []bool{false, true} {
			for _, missing := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/rejected=%t/missing=%t", kind, rejected, missing), func(t *testing.T) {
					app := newWorkflowTestApp(t, true)
					const requestID = "parequest000001"
					actor, method := "time@test.com", http.MethodPost
					path := "/api/job_requests/" + requestID + "/document"
					attachment := workflowRecord(t, app, "job_attachments", "paattachment001")
					data, err := os.ReadFile("test_seed_data/files/job_attachments/paattachment001/authorization.pdf")
					if err != nil {
						t.Fatal(err)
					}
					fields := map[string]string{}
					if kind == "value" {
						actor, method = workflowAuthor, http.MethodPatch
						data = authorizationPDF(t, "authorization-upload.pdf")
						fields = map[string]string{"job": legacyWorkflowJob, "project_value": "10000", "assigned_approver": "6bq4j0eb26631dy"}
						workflowUpload(t, app, "/api/job_attachments", actor, string(data), fields, http.StatusOK)
						attachment, err = app.FindFirstRecordByData("job_attachments", "job", legacyWorkflowJob)
						if err != nil {
							t.Fatal(err)
						}
						path = "/api/job_attachments/" + attachment.Id
					}
					if rejected {
						rejectPath := path + "/reject"
						input := attachmentActionData(t, app, attachment.Id)
						if kind == "request" {
							clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+requestID+"/submit", actor, requestActionData(t, app, requestID), http.StatusOK)
							rejectPath = "/api/job_requests/" + requestID + "/reject"
							input = requestActionData(t, app, requestID)
						}
						input["rejection_reason"] = "Correct the project details."
						clientSetupJSON(t, app, http.MethodPost, rejectPath, workflowApprover, input, http.StatusOK)
					}
					attachment = workflowRecord(t, app, "job_attachments", attachment.Id)
					fields["version"] = fmt.Sprint(attachment.GetInt("version"))
					requestBefore := workflowRecord(t, app, "job_requests", requestID).PublicExport()
					if kind == "request" {
						fields["version"] = fmt.Sprint(workflowRecord(t, app, "job_requests", requestID).GetInt("version"))
					}
					before := attachment.PublicExport()
					if missing {
						removeAuthorizationFile(t, app, attachment)
					}
					workflowUploadMethod(t, app, method, path, workflowApprover, string(data), fields, http.StatusForbidden)
					workflowUploadMethod(t, app, method, path, actor, string(data), fields, http.StatusOK)
					after := workflowRecord(t, app, "job_attachments", attachment.Id)
					assertAuthorizationBytes(t, app, after, data)
					if !reflect.DeepEqual(before, after.PublicExport()) || !reflect.DeepEqual(requestBefore, workflowRecord(t, app, "job_requests", requestID).PublicExport()) {
						t.Fatal("matching upload changed workflow data")
					}
				})
			}
		}
	}
}

type authorizationStorageTransport func(*http.Request) (*http.Response, error)

func (f authorizationStorageTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestProjectAuthorizationRecoveryDoesNotUnlockApprovedEvidence(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	const requestID = "parequest000001"
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+requestID+"/submit", "time@test.com", requestActionData(t, app, requestID), http.StatusOK)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_requests/"+requestID+"/approve", workflowApprover, requestActionData(t, app, requestID), http.StatusOK)
	requestAttachment := workflowRecord(t, app, "job_attachments", "paattachment001")
	removeAuthorizationFile(t, app, requestAttachment)
	data, err := os.ReadFile("test_seed_data/files/job_attachments/paattachment001/authorization.pdf")
	if err != nil {
		t.Fatal(err)
	}
	workflowUpload(t, app, "/api/job_requests/"+requestID+"/document", "time@test.com", string(data), map[string]string{"version": fmt.Sprint(workflowRecord(t, app, "job_requests", requestID).GetInt("version"))}, http.StatusBadRequest)

	fields := map[string]string{"job": legacyWorkflowJob, "project_value": "10000", "assigned_approver": "6bq4j0eb26631dy"}
	data = authorizationPDF(t, "authorization-upload.pdf")
	workflowUpload(t, app, "/api/job_attachments", workflowAuthor, string(data), fields, http.StatusOK)
	valueAttachment, err := app.FindFirstRecordByData("job_attachments", "job", legacyWorkflowJob)
	if err != nil {
		t.Fatal(err)
	}
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+valueAttachment.Id+"/approve", workflowApprover, attachmentActionData(t, app, valueAttachment.Id), http.StatusOK)
	valueAttachment = workflowRecord(t, app, "job_attachments", valueAttachment.Id)
	removeAuthorizationFile(t, app, valueAttachment)
	fields["version"] = fmt.Sprint(valueAttachment.GetInt("version"))
	workflowUploadMethod(t, app, http.MethodPatch, "/api/job_attachments/"+valueAttachment.Id, workflowAuthor, string(data), fields, http.StatusBadRequest)
	fs, err := app.NewFilesystem()
	if err != nil {
		t.Fatal(err)
	}
	defer fs.Close()
	for _, attachment := range []*core.Record{requestAttachment, valueAttachment} {
		exists, err := fs.Exists(attachment.BaseFilesPath() + "/" + attachment.GetString("document"))
		if err != nil || exists {
			t.Fatalf("rejected upload restored approved object: exists=%t, err=%v", exists, err)
		}
		if !reflect.DeepEqual(attachment.PublicExport(), workflowRecord(t, app, "job_attachments", attachment.Id).PublicExport()) {
			t.Fatal("rejected upload changed approved evidence")
		}
	}
}

func TestProjectAuthorizationActionsUseS3ExistenceChecks(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	valueID := createValueChange(t, app, legacyWorkflowJob, 10000, "S3 existence check")
	// Use an in-process S3 transport. No request reaches a real storage service.
	app.Settings().S3 = core.S3Config{Enabled: true, Bucket: "authorization-tests", Region: "test-region", Endpoint: "https://storage.example.test", AccessKey: "test", Secret: "test", ForcePathStyle: true}
	previous := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = previous })
	requests := 0
	status := http.StatusServiceUnavailable
	http.DefaultClient = &http.Client{Transport: authorizationStorageTransport(func(r *http.Request) (*http.Response, error) {
		requests++
		if r.Method != http.MethodHead || r.URL.Host != "storage.example.test" {
			t.Errorf("unexpected storage request: %s %s", r.Method, r.URL)
			return nil, fmt.Errorf("only object-existence checks are allowed")
		}
		return &http.Response{StatusCode: status, Header: http.Header{}, Body: http.NoBody, Request: r}, nil
	})}
	const requestID = "parequest000001"
	path := "/api/job_requests/" + requestID
	// Storage errors and missing objects must both stop submission.
	clientSetupJSON(t, app, http.MethodPost, path+"/submit", "time@test.com", requestActionData(t, app, requestID), http.StatusBadRequest)
	status = http.StatusNotFound
	clientSetupJSON(t, app, http.MethodPost, path+"/submit", "time@test.com", requestActionData(t, app, requestID), http.StatusBadRequest)
	status = http.StatusOK
	requests = 0
	data, err := os.ReadFile("test_seed_data/files/job_attachments/paattachment001/authorization.pdf")
	if err != nil {
		t.Fatal(err)
	}
	workflowUpload(t, app, path+"/document", "time@test.com", string(data), map[string]string{"version": fmt.Sprint(workflowRecord(t, app, "job_requests", requestID).GetInt("version"))}, http.StatusOK)
	clientSetupJSON(t, app, http.MethodPost, path+"/submit", "time@test.com", requestActionData(t, app, requestID), http.StatusOK)
	clientSetupJSON(t, app, http.MethodPost, path+"/approve", workflowApprover, requestActionData(t, app, requestID), http.StatusOK)
	clientSetupJSON(t, app, http.MethodPost, path+"/create", workflowAuthor, requestActionData(t, app, requestID), http.StatusOK)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+valueID+"/approve", workflowApprover, attachmentActionData(t, app, valueID), http.StatusOK)
	if requests != 5 {
		t.Fatalf("storage requests = %d, want one HEAD for the matching upload and each workflow action", requests)
	}
}
