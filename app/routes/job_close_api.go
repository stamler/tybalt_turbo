package routes

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"tybalt/errs"
	"tybalt/hooks"
	"tybalt/utilities"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
)

type closeJobProposalResult struct {
	ID                string `json:"id"`
	AutoAwarded       bool   `json:"auto_awarded"`
	FromStatus        string `json:"from_status,omitempty"`
	ToStatus          string `json:"to_status,omitempty"`
	ProposalNoteAdded bool   `json:"proposal_note_created,omitempty"`
}

type closeJobResponse struct {
	ID               string                  `json:"id"`
	Status           string                  `json:"status"`
	Imported         bool                    `json:"imported"`
	ProjectNoteAdded bool                    `json:"project_note_created"`
	Proposal         *closeJobProposalResult `json:"proposal,omitempty"`
}

// createCloseJobHandler handles the "Close project" action for any Active
// project.
//
// Closing has no side effects on time, expenses or purchase orders, so it needs
// no billing data. It uses the same retirement rule as a status-only close in
// the editor (hooks.ProcessJobCore): the job claim, but no invoicing profile,
// project value or authorizing document. The endpoint adds what the editor
// cannot do in one save: it reconciles the referenced proposal and writes the
// audit notes in the same transaction as the close.
func createCloseJobHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		if authRecord == nil || authRecord.Id == "" {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		jobID := e.Request.PathValue("id")
		if strings.TrimSpace(jobID) == "" {
			return e.Error(http.StatusBadRequest, "missing job id", nil)
		}

		var (
			httpResponseStatusCode = http.StatusOK
			resp                   closeJobResponse
		)

		err := app.RunInTransaction(func(txApp core.App) error {
			jobRec, err := txApp.FindRecordById("jobs", jobID)
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{Code: "job_not_found", Message: "job not found"}
			}

			hasJobClaim, claimErr := utilities.HasClaim(txApp, authRecord, "job")
			if claimErr != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{Code: "claim_check_failed", Message: claimErr.Error()}
			}
			// Managers without the job claim may change allocations only.
			if !hasJobClaim {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{Code: "unauthorized", Message: "the job claim is required to close projects"}
			}

			if strings.HasPrefix(jobRec.GetString("number"), "P") {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{Code: "not_a_project", Message: "only projects can be closed with this endpoint"}
			}

			if jobRec.GetString("status") != "Active" {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{Code: "invalid_status_for_close", Message: "only Active projects can be closed"}
			}

			var proposalResult *closeJobProposalResult
			proposalRef := strings.TrimSpace(jobRec.GetString("proposal"))
			if proposalRef != "" {
				proposalRec, err := txApp.FindRecordById("jobs", proposalRef)
				if err != nil {
					httpResponseStatusCode = http.StatusBadRequest
					return &CodeError{Code: "proposal_not_awarded", Message: "referenced proposal was not found"}
				}

				refStatus := proposalRec.GetString("status")
				if refStatus == "Not Awarded" || refStatus == "Cancelled" || refStatus == "No Bid" {
					httpResponseStatusCode = http.StatusBadRequest
					return &CodeError{
						Code:    "proposal_terminal_status_blocks_close",
						Message: "referenced proposal has a terminal status and cannot be auto-awarded for close",
					}
				}

				proposalResult = &closeJobProposalResult{
					ID:          proposalRec.Id,
					AutoAwarded: false,
				}

				if refStatus != "Awarded" {
					// IMPORTANT:
					// We only auto-award imported proposals in the intermediate
					// workflow states that represent "not yet final" outcomes.
					//
					// This avoids rewriting explicit business decisions made in
					// this app while still reconciling imported legacy data.
					if (refStatus == "In Progress" || refStatus == "Submitted") && proposalRec.GetBool("_imported") {
						proposalRec.Set("status", "Awarded")
						proposalRec.Set("_imported", false)

						if err := txApp.Save(proposalRec); err != nil {
							httpResponseStatusCode = http.StatusBadRequest
							return &CodeError{
								Code:    "error_updating_job",
								Message: fmt.Sprintf("error auto-awarding proposal: %v", err),
							}
						}

						// We intentionally do not set job_status_changed_to for this
						// audit note. That field currently models proposal close-out
						// comment workflows (No Bid/Cancelled), while this note is a
						// reconciliation artifact for the project close.
						if err := createJobAuditNote(
							txApp,
							proposalRec,
							authRecord.Id,
							fmt.Sprintf("Proposal auto-awarded when project %s (%s) was closed", jobRec.GetString("number"), jobRec.Id),
							"",
						); err != nil {
							httpResponseStatusCode = http.StatusBadRequest
							return &CodeError{
								Code:    "error_creating_proposal_auto_award_note",
								Message: err.Error(),
							}
						}

						proposalResult.AutoAwarded = true
						proposalResult.FromStatus = refStatus
						proposalResult.ToStatus = "Awarded"
						proposalResult.ProposalNoteAdded = true
					} else {
						httpResponseStatusCode = http.StatusBadRequest
						return &CodeError{
							Code:    "proposal_not_awarded",
							Message: "referenced proposal must be Awarded before project close",
						}
					}
				}
			}

			// A status-only change is a retirement, so ProcessJobCore skips the
			// billing checks. It also clears _imported, because the status changed.
			jobRec.Set("status", "Closed")
			if err := hooks.ProcessJobCore(txApp, jobRec, authRecord); err != nil {
				var hookErr *errs.HookError
				if errors.As(err, &hookErr) {
					httpResponseStatusCode = hookErr.Status
				} else {
					httpResponseStatusCode = http.StatusBadRequest
				}
				return err
			}

			if err := txApp.SaveWithContext(e.Request.Context(), jobRec); err != nil {
				httpResponseStatusCode = http.StatusBadRequest
				var validationErrs validation.Errors
				if errors.As(err, &validationErrs) {
					fieldErrors := make(map[string]errs.CodeError)
					for field, fieldErr := range validationErrs {
						fieldErrors[field] = errs.CodeError{
							Code:    "validation_error",
							Message: fieldErr.Error(),
						}
					}
					return &errs.HookError{
						Status:  http.StatusBadRequest,
						Message: "validation failed",
						Data:    fieldErrors,
					}
				}
				return &CodeError{
					Code:    "error_updating_job",
					Message: fmt.Sprintf("error updating job status: %v", err),
				}
			}

			if err := createJobAuditNote(
				txApp,
				jobRec,
				authRecord.Id,
				"Project closed with the Close project action",
				"",
			); err != nil {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{
					Code:    "error_creating_project_close_note",
					Message: err.Error(),
				}
			}

			resp = closeJobResponse{
				ID:               jobRec.Id,
				Status:           jobRec.GetString("status"),
				Imported:         jobRec.GetBool("_imported"),
				ProjectNoteAdded: true,
				Proposal:         proposalResult,
			}
			return nil
		})

		if err != nil {
			var hookErr *errs.HookError
			if errors.As(err, &hookErr) {
				return e.JSON(httpResponseStatusCode, hookErr)
			}
			if codeError, ok := err.(*CodeError); ok {
				return e.JSON(httpResponseStatusCode, map[string]any{
					"code":    codeError.Code,
					"message": codeError.Message,
				})
			}
			return e.JSON(httpResponseStatusCode, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, resp)
	}
}

// createJobAuditNote inserts a client note tied to a job.
//
// We keep this helper in route code (instead of hooks) because closing a project
// requires deterministic audit creation inside the same explicit transaction as
// status mutations, and route handlers already coordinate that transaction.
func createJobAuditNote(
	txApp core.App,
	jobRec *core.Record,
	userID string,
	note string,
	jobStatusChangedTo string,
) error {
	clientID := strings.TrimSpace(jobRec.GetString("client"))
	if clientID == "" {
		return fmt.Errorf("client is required to create audit note")
	}
	return createClientNote(txApp, clientID, jobRec.Id, userID, note, jobStatusChangedTo)
}

// createClientNote inserts a client note. A note without a job is marked as not
// applying to one.
func createClientNote(txApp core.App, clientID, jobID, userID, note, jobStatusChangedTo string) error {
	notesCol, err := txApp.FindCollectionByNameOrId("client_notes")
	if err != nil {
		return fmt.Errorf("client_notes collection not found: %w", err)
	}

	noteRec := core.NewRecord(notesCol)
	noteRec.Set("client", clientID)
	if jobID != "" {
		noteRec.Set("job", jobID)
	} else {
		noteRec.Set("job_not_applicable", true)
	}
	noteRec.Set("uid", userID)
	noteRec.Set("note", strings.TrimSpace(note))
	if strings.TrimSpace(jobStatusChangedTo) != "" {
		noteRec.Set("job_status_changed_to", jobStatusChangedTo)
	}

	if err := txApp.Save(noteRec); err != nil {
		return fmt.Errorf("error creating client note: %w", err)
	}
	return nil
}
