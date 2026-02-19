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
	Mode             string                  `json:"mode"`
	ProjectNoteAdded bool                    `json:"project_note_created"`
	Proposal         *closeJobProposalResult `json:"proposal,omitempty"`
}

// createCloseJobHandler handles the dedicated "fast close" workflow.
//
// Why this endpoint exists:
// The normal job update flow supports status-only edits and can still trip over
// legacy-data completeness issues depending on the payload and normalization
// path. Product policy for fast close is explicit:
//  1. Imported legacy jobs may close with a controlled bypass.
//  2. Non-imported jobs must use strict/full validation.
//  3. Close operations must emit audit notes deterministically.
//
// Putting this in a dedicated endpoint keeps the policy boundary narrow and
// avoids changing the behavior of existing update/edit routes.
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
			managerID := jobRec.GetString("manager")
			altManagerID := jobRec.GetString("alternate_manager")
			isJobManager := authRecord.Id != "" && (authRecord.Id == managerID || authRecord.Id == altManagerID)
			if !hasJobClaim && !isJobManager {
				httpResponseStatusCode = http.StatusForbidden
				return &CodeError{Code: "unauthorized", Message: "you are not authorized to close this job"}
			}

			if strings.HasPrefix(jobRec.GetString("number"), "P") {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{Code: "not_a_project", Message: "only projects can be closed with this endpoint"}
			}

			if jobRec.GetString("status") != "Active" {
				httpResponseStatusCode = http.StatusBadRequest
				return &CodeError{Code: "invalid_status_for_close", Message: "only Active projects can be closed"}
			}

			mode := "validated"
			if jobRec.GetBool("_imported") {
				mode = "bypass"
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
					// This avoids rewriting explicit terminal business decisions
					// while still allowing legacy reconciliation for imported data.
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
						// reconciliation artifact for fast close.
						if err := createJobAuditNote(
							txApp,
							proposalRec,
							authRecord.Id,
							fmt.Sprintf("Proposal auto-awarded during imported fast close of project %s (%s)", jobRec.GetString("number"), jobRec.Id),
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

			jobRec.Set("status", "Closed")

			if mode == "validated" {
				if err := hooks.ProcessJobCoreStrict(txApp, jobRec, authRecord); err != nil {
					var hookErr *errs.HookError
					if errors.As(err, &hookErr) {
						httpResponseStatusCode = hookErr.Status
					} else {
						httpResponseStatusCode = http.StatusBadRequest
					}
					return err
				}
			} else {
				jobRec.Set("_imported", false)
			}

			if err := txApp.Save(jobRec); err != nil {
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
				"Project closed via imported fast close flow",
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
				Mode:             mode,
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
// We keep this helper in route code (instead of hooks) because fast-close
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

	notesCol, err := txApp.FindCollectionByNameOrId("client_notes")
	if err != nil {
		return fmt.Errorf("client_notes collection not found: %w", err)
	}

	noteRec := core.NewRecord(notesCol)
	noteRec.Set("job", jobRec.Id)
	noteRec.Set("client", clientID)
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
