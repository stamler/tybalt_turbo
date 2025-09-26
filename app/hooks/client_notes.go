package hooks

import (
	"net/http"

	"tybalt/errs"

	"github.com/pocketbase/pocketbase/core"
)

// ProcessClientNote enforces business rules for client note creation.
func ProcessClientNote(app core.App, e *core.RecordRequestEvent) error {
	if err := cleanClientNote(app, e); err != nil {
		return err
	}

	if err := validateClientNote(app, e.Record); err != nil {
		return err
	}

	return nil
}

func cleanClientNote(app core.App, e *core.RecordRequestEvent) error {
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

	return nil
}
