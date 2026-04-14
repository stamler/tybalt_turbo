package routes

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"tybalt/errs"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type jobSetNumberRequest struct {
	Number string `json:"number"`
}

type jobSetNumberCodeError struct {
	Status  int
	Code    string
	Message string
}

func (e *jobSetNumberCodeError) Error() string {
	return e.Message
}

// Match first-level child job numbers using the same legacy-friendly shape as
// the collection regex. The optional P preserves compatibility with any
// existing legacy rows even though new proposal sub-jobs are not allowed.
var jobChildNumberRegex = regexp.MustCompile(`^(?:P)?\d{2}-\d{3,4}-\d{1,2}$`)

func jobNumberFieldError(status int, code string, message string) *errs.HookError {
	return &errs.HookError{
		Status:  status,
		Message: "validation failed",
		Data: map[string]errs.CodeError{
			"number": {
				Code:    code,
				Message: message,
			},
		},
	}
}

func convertJobNumberValidationError(status int, err error) *errs.HookError {
	if err == nil {
		return nil
	}

	code := "validation_error"
	if ce, ok := err.(codeError); ok && ce.Code() != "" {
		code = ce.Code()
	}

	return jobNumberFieldError(status, code, err.Error())
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "unique constraint failed") || strings.Contains(lower, "not unique")
}

func createSetJobNumberHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		if authRecord == nil || authRecord.Id == "" {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		enabled, err := utilities.IsJobsEditingEnabled(app)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to check jobs editing status", err)
		}
		if !enabled {
			return e.JSON(utilities.ErrJobsEditingDisabled.Status, utilities.ErrJobsEditingDisabled)
		}

		jobID := e.Request.PathValue("id")
		if jobID == "" {
			return e.Error(http.StatusBadRequest, "missing job id", nil)
		}

		var req jobSetNumberRequest
		if err := e.BindBody(&req); err != nil {
			return e.Error(http.StatusBadRequest, "invalid request body", err)
		}

		req.Number = strings.TrimSpace(req.Number)

		var updatedNumber string

		err = app.RunInTransaction(func(txApp core.App) error {
			hasAdminClaim, err := utilities.HasClaim(txApp, authRecord, "admin")
			if err != nil {
				return &jobSetNumberCodeError{
					Status:  http.StatusInternalServerError,
					Code:    "claim_check_failed",
					Message: "failed to verify admin claim",
				}
			}
			if !hasAdminClaim {
				return &jobSetNumberCodeError{
					Status:  http.StatusForbidden,
					Code:    "unauthorized",
					Message: "you are not authorized to change job numbers",
				}
			}

			jobRec, err := txApp.FindRecordById("jobs", jobID)
			if err != nil {
				return &jobSetNumberCodeError{
					Status:  http.StatusNotFound,
					Code:    "job_not_found",
					Message: "job not found",
				}
			}

			jobsCollection, err := txApp.FindCollectionByNameOrId("jobs")
			if err != nil {
				return &jobSetNumberCodeError{
					Status:  http.StatusInternalServerError,
					Code:    "error_finding_collection",
					Message: "failed to load jobs collection",
				}
			}

			numberField, ok := jobsCollection.Fields.GetByName("number").(*core.TextField)
			if !ok || numberField == nil {
				return &jobSetNumberCodeError{
					Status:  http.StatusInternalServerError,
					Code:    "number_field_invalid",
					Message: "jobs.number field is not configured as text",
				}
			}

			currentNumber := strings.TrimSpace(jobRec.GetString("number"))
			updatedNumber = currentNumber

			if req.Number == currentNumber {
				return nil
			}

			if err := numberField.ValidatePlainValue(req.Number); err != nil {
				return convertJobNumberValidationError(http.StatusBadRequest, err)
			}

			currentIsProposal := strings.HasPrefix(currentNumber, "P")
			nextIsProposal := strings.HasPrefix(req.Number, "P")
			if currentIsProposal != nextIsProposal {
				if currentIsProposal {
					return jobNumberFieldError(http.StatusBadRequest, "proposal_number_required", "proposals must keep a proposal-form number")
				}
				return jobNumberFieldError(http.StatusBadRequest, "project_number_required", "projects must not use a proposal-form number")
			}

			currentIsChild := strings.TrimSpace(jobRec.GetString("parent")) != ""
			nextIsChild := jobChildNumberRegex.MatchString(req.Number)
			if currentIsChild != nextIsChild {
				if currentIsChild {
					return jobNumberFieldError(http.StatusBadRequest, "child_number_required", "child jobs must keep a child-form number")
				}
				return jobNumberFieldError(http.StatusBadRequest, "top_level_number_required", "top-level jobs cannot use a child-form number")
			}

			// Intentionally update the row directly instead of calling txApp.Save:
			// this admin-only endpoint is a renumber escape hatch and must avoid
			// re-running full jobs request hooks and unrelated record validation
			// (for example cancelled-proposal edit locks or legacy blank fields).
			// Writeback still sees the change because we explicitly mark the job as
			// locally modified and bump the updated timestamp here.
			if _, err := txApp.DB().NewQuery(`
				UPDATE jobs
				SET number = {:number},
					_imported = false,
					updated = strftime('%Y-%m-%d %H:%M:%fZ', 'now')
				WHERE id = {:id}
			`).Bind(dbx.Params{
				"number": req.Number,
				"id":     jobID,
			}).Execute(); err != nil {
				if isUniqueConstraintError(err) {
					return jobNumberFieldError(http.StatusConflict, "validation_not_unique", "job number must be unique")
				}
				return &jobSetNumberCodeError{
					Status:  http.StatusInternalServerError,
					Code:    "error_saving_job",
					Message: err.Error(),
				}
			}

			updatedNumber = req.Number
			return nil
		})

		if err != nil {
			var hookErr *errs.HookError
			if errors.As(err, &hookErr) {
				return e.JSON(hookErr.Status, hookErr)
			}
			var codeErr *jobSetNumberCodeError
			if errors.As(err, &codeErr) {
				return e.JSON(codeErr.Status, map[string]any{
					"code":    codeErr.Code,
					"message": codeErr.Message,
				})
			}
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"id":     jobID,
			"number": updatedNumber,
		})
	}
}
