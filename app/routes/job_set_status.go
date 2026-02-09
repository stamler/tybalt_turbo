package routes

import (
	"fmt"
	"net/http"
	"strings"

	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
)

// jobSetStatusRequest models the request body for setting a proposal status
// that requires a comment (Cancelled or No Bid).
type jobSetStatusRequest struct {
	Status  string `json:"status"`
	Comment string `json:"comment"`
}

// createSetJobStatusHandler returns a handler that atomically updates a
// proposal's status to Cancelled or No Bid and creates the accompanying
// client_note in a single transaction. This avoids the data-integrity problem
// of creating a note first and then failing to save the status change.
func createSetJobStatusHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		if authRecord == nil || authRecord.Id == "" {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		jobID := e.Request.PathValue("id")
		if jobID == "" {
			return e.Error(http.StatusBadRequest, "missing job id", nil)
		}

		var req jobSetStatusRequest
		if err := e.BindBody(&req); err != nil {
			return e.Error(http.StatusBadRequest, "invalid request body", err)
		}

		// Only Cancelled and No Bid are handled by this endpoint.
		if req.Status != "Cancelled" && req.Status != "No Bid" {
			return e.JSON(http.StatusBadRequest, map[string]any{
				"code":    "invalid_status",
				"message": "this endpoint only supports setting status to Cancelled or No Bid",
			})
		}

		if strings.TrimSpace(req.Comment) == "" {
			return e.JSON(http.StatusBadRequest, map[string]any{
				"code":    "comment_required",
				"message": "a comment is required when setting status to " + req.Status,
			})
		}

		var httpResponseStatusCode int

		err := app.RunInTransaction(func(txApp core.App) error {
			// Load the job
			jobRec, err := txApp.FindRecordById("jobs", jobID)
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{Code: "job_not_found", Message: "job not found"}
			}

			// Must be a proposal (number starts with P)
			if !strings.HasPrefix(jobRec.GetString("number"), "P") {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{Code: "not_a_proposal", Message: "this endpoint is only for proposals"}
			}

			// Must not already be cancelled
			currentStatus := jobRec.GetString("status")
			if currentStatus == "Cancelled" {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{Code: "already_cancelled", Message: "this proposal is already cancelled"}
			}

			// Authorization: holders of 'job' claim OR job manager/alternate_manager
			hasJobClaim, claimErr := utilities.HasClaim(txApp, authRecord, "job")
			if claimErr != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{Code: "claim_check_failed", Message: claimErr.Error()}
			}
			managerID := jobRec.GetString("manager")
			altManagerID := jobRec.GetString("alternate_manager")
			isJobManager := authRecord.Id == managerID || authRecord.Id == altManagerID
			if !hasJobClaim && !isJobManager {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{Code: "unauthorized", Message: "you are not authorized to update this job"}
			}

			// Create the client_note
			notesCol, err := txApp.FindCollectionByNameOrId("client_notes")
			if err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{Code: "collection_not_found", Message: "client_notes collection not found"}
			}

			clientID := jobRec.GetString("client")
			if clientID == "" {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{Code: "no_client", Message: "this proposal has no client; a client is required to add a note"}
			}

			noteRec := core.NewRecord(notesCol)
			noteRec.Set("job", jobID)
			noteRec.Set("client", clientID)
			noteRec.Set("uid", authRecord.Id)
			noteRec.Set("note", strings.TrimSpace(req.Comment))
			noteRec.Set("job_status_changed_to", req.Status)

			if err := txApp.Save(noteRec); err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_creating_note",
					Message: fmt.Sprintf("error creating client note: %v", err),
				}
			}

			// Update the job status
			jobRec.Set("status", req.Status)
			if err := txApp.Save(jobRec); err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_updating_status",
					Message: fmt.Sprintf("error updating job status: %v", err),
				}
			}

			return nil
		})

		if err != nil {
			if codeError, ok := err.(*CodeError); ok {
				return e.JSON(httpResponseStatusCode, map[string]any{
					"code":    codeError.Code,
					"message": codeError.Message,
				})
			}
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"status": req.Status,
		})
	}
}
