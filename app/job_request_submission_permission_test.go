package main

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/pocketbase/dbx"
)

func TestJobRequestSubmissionRechecksTimeClaim(t *testing.T) {
	for _, state := range []string{"Draft", "Recalled"} {
		for _, removeClaim := range []bool{false, true} {
			name := state + "/claim-present"
			if removeClaim {
				name = state + "/claim-removed"
			}
			t.Run(name, func(t *testing.T) {
				app := newWorkflowTestApp(t, true)
				const requestID = "parequest000001"
				const creatorEmail = "time@test.com"
				path := "/api/job_requests/" + requestID
				if state == "Recalled" {
					clientSetupJSON(t, app, http.MethodPost, path+"/submit", creatorEmail, requestActionData(t, app, requestID), http.StatusOK)
					clientSetupJSON(t, app, http.MethodPost, path+"/recall", creatorEmail, requestActionData(t, app, requestID), http.StatusOK)
				}
				token := workflowToken(t, app, creatorEmail)
				input := requestActionData(t, app, requestID)
				request := workflowRecord(t, app, "job_requests", requestID)
				requestBefore := request.PublicExport()
				attachmentBefore := workflowRecord(t, app, "job_attachments", "paattachment001").PublicExport()
				if removeClaim {
					// Remove only the fixture creator's time assignment after the draft
					// and authentication token exist, to test permission revocation.
					assignment, err := app.FindFirstRecordByFilter("user_claims", "uid={:uid} && cid.name='time'", dbx.Params{"uid": request.GetString("creator")})
					if err != nil {
						t.Fatal(err)
					}
					if err := app.Delete(assignment); err != nil {
						t.Fatal(err)
					}
				}
				response := performJSONRequest(t, app, http.MethodPost, path+"/submit", token, input)
				if !removeClaim {
					mustStatus(t, response, http.StatusOK)
					if workflowRecord(t, app, "job_requests", requestID).GetString("status") != "Awaiting approval" {
						t.Fatal("authorized submission did not enter the approval queue")
					}
					return
				}
				mustStatus(t, response, http.StatusForbidden)
				if !reflect.DeepEqual(requestBefore, workflowRecord(t, app, "job_requests", requestID).PublicExport()) {
					t.Fatal("denied submission changed the request")
				}
				if !reflect.DeepEqual(attachmentBefore, workflowRecord(t, app, "job_attachments", "paattachment001").PublicExport()) {
					t.Fatal("denied submission changed the attachment")
				}
			})
		}
	}
}
