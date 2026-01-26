package hooks

import (
	"net/http"

	"tybalt/errs"

	"github.com/pocketbase/pocketbase/core"
)

// ProcessClientNote enforces business rules for client note creation.
func ProcessClientNote(app core.App, e *core.RecordRequestEvent) error {
	if err := cleanClientNote(e); err != nil {
		return err
	}

	if err := validateClientNote(app, e.Record); err != nil {
		return err
	}

	return nil
}

func cleanClientNote(e *core.RecordRequestEvent) error {
	if e.Auth == nil {
		return &errs.HookError{
			Status:  http.StatusForbidden,
			Message: "authentication required to create client note",
			Data: map[string]errs.CodeError{
				"uid": {
					Code:    "authentication_required",
					Message: "authentication required",
				},
			},
		}
	}

	e.Record.Set("uid", e.Auth.Id)

	return nil
}

func validateClientNote(app core.App, record *core.Record) error {
	clientID := record.GetString("client")
	jobID := record.GetString("job")
	jobNotApplicable := record.GetBool("job_not_applicable")

	if clientID == "" {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "client note validation error",
			Data: map[string]errs.CodeError{
				"client": {
					Code:    "required",
					Message: "client is required",
				},
			},
		}
	}

	if jobID == "" && !jobNotApplicable {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "client note validation error",
			Data: map[string]errs.CodeError{
				"job": {
					Code:    "job_or_flag_required",
					Message: "job is required unless marked not applicable",
				},
				"job_not_applicable": {
					Code:    "job_or_flag_required",
					Message: "job must be selected or marked not applicable",
				},
			},
		}
	}

	if jobID == "" {
		return nil
	}

	jobRecord, err := app.FindRecordById("jobs", jobID)
	if err != nil || jobRecord == nil {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "client note validation error",
			Data: map[string]errs.CodeError{
				"job": {
					Code:    "not_found",
					Message: "job not found",
				},
			},
		}
	}

	jobClientID := jobRecord.GetString("client")
	jobOwnerID := jobRecord.GetString("job_owner")

	if jobClientID != clientID && jobOwnerID != clientID {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "client note validation error",
			Data: map[string]errs.CodeError{
				"job": {
					Code:    "job_client_mismatch",
					Message: "job must belong to the selected client",
				},
			},
		}
	}

	// Validate job_status_changed_to against allowed values for the job type.
	// When adding new job_status_changed_to options to the schema, update the appropriate
	// list below to allow them for proposals and/or projects.
	statusChangeTo := record.GetString("job_status_changed_to")
	if statusChangeTo != "" {
		jobNumber := jobRecord.GetString("number")
		isProposal := len(jobNumber) > 0 && jobNumber[0] == 'P'

		// Allowed status_change_to values for proposals.
		// Update this list when adding new proposal status options that require comments.
		allowedForProposals := []string{"No Bid", "Cancelled"}

		// Allowed status_change_to values for projects.
		// Update this list when adding new project status options that require comments.
		allowedForProjects := []string{}

		var allowed []string
		var jobTypeName string
		if isProposal {
			allowed = allowedForProposals
			jobTypeName = "proposals"
		} else {
			allowed = allowedForProjects
			jobTypeName = "projects"
		}

		isAllowed := false
		for _, v := range allowed {
			if v == statusChangeTo {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "client note validation error",
				Data: map[string]errs.CodeError{
					"job_status_changed_to": {
						Code:    "invalid_for_job_type",
						Message: "job_status_changed_to value '" + statusChangeTo + "' is not valid for " + jobTypeName,
					},
				},
			}
		}
	}

	return nil
}
