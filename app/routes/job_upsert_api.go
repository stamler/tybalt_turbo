package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"tybalt/errs"
	"tybalt/hooks"
	"tybalt/utilities"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// JobUpsertRequest models the request body for creating/updating a job and its allocations.
type JobUpsertRequest struct {
	Job         map[string]any        `json:"job"`
	Allocations []JobAllocationUpdate `json:"allocations"`
}

type JobAllocationUpdate struct {
	Division string  `json:"division"`
	Hours    float64 `json:"hours"`
}

// createUpsertJobHandler returns a handler that updates a job and replaces its allocations in a single transaction.
func createUpsertJobHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		if authRecord == nil || authRecord.Id == "" {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		jobID := e.Request.PathValue("id")
		if jobID == "" {
			return e.Error(http.StatusBadRequest, "missing job id", nil)
		}

		var req JobUpsertRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return e.Error(http.StatusBadRequest, "invalid JSON body", err)
		}

		// Basic allocations validation
		seen := make(map[string]struct{})
		for _, a := range req.Allocations {
			if a.Division == "" {
				return e.Error(http.StatusBadRequest, "division is required for all allocations", nil)
			}
			if a.Hours < 0 {
				return e.Error(http.StatusBadRequest, "hours must be >= 0", nil)
			}
			if _, ok := seen[a.Division]; ok {
				return e.Error(http.StatusBadRequest, "duplicate division in allocations", nil)
			}
			seen[a.Division] = struct{}{}
		}

		var httpResponseStatusCode = http.StatusOK

		err := app.RunInTransaction(func(txApp core.App) error {
			// Load the job
			jobRec, err := txApp.FindRecordById("jobs", jobID)
			if err != nil {
				httpResponseStatusCode = http.StatusNotFound
				return &CodeError{
					Code:    "job_not_found",
					Message: fmt.Sprintf("job not found: %v", err),
				}
			}

			// Authorization: holders of 'job' claim OR job manager/alternate_manager
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
				return apis.NewForbiddenError("you are not authorized to update this job", nil)
			}

			// Update job fields (ignore id/system fields)
			if req.Job != nil {
				for k, v := range req.Job {
					switch k {
					case "id", "collectionId", "collectionName", "created", "updated", "divisions":
						// ignore
					default:
						jobRec.Set(k, v)
					}
				}
			}

			// Run job validation and business rules (this will also handle number generation for creates)
			if err := hooks.ProcessJobCore(txApp, jobRec, authRecord); err != nil {
				// If it's a HookError, preserve the status code
				var hookErr *errs.HookError
				if errors.As(err, &hookErr) {
					httpResponseStatusCode = hookErr.Status
				} else {
					httpResponseStatusCode = http.StatusBadRequest
				}
				return err
			}

			if err := txApp.Save(jobRec); err != nil {
				httpResponseStatusCode = http.StatusBadRequest
				// Check if it's a validation error with field-level details
				var validationErrs validation.Errors
				if errors.As(err, &validationErrs) {
					// Convert to HookError format for consistent frontend handling
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
					Code:    "error_saving_job",
					Message: fmt.Sprintf("error saving job: %v", err),
				}
			}

			// Validate all divisions exist and are active
			for _, a := range req.Allocations {
				divRec, err := txApp.FindRecordById("divisions", a.Division)
				if err != nil {
					httpResponseStatusCode = http.StatusBadRequest
					return &CodeError{
						Code:    "invalid_division",
						Message: fmt.Sprintf("division not found: %s", a.Division),
					}
				}
				if !divRec.GetBool("active") {
					httpResponseStatusCode = http.StatusBadRequest
					return &CodeError{
						Code:    "division_not_active",
						Message: fmt.Sprintf("division is inactive: %s", a.Division),
					}
				}
			}

			// Replace strategy: delete all existing allocations for this job,
			// then insert the provided set. This avoids driver-specific IN bindings.
			if _, err := txApp.DB().NewQuery("DELETE FROM job_time_allocations WHERE job = {:job}").
				Bind(dbx.Params{"job": jobID}).Execute(); err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_deleting_allocations",
					Message: fmt.Sprintf("error deleting allocations: %v", err),
				}
			}

			// Prepare allocations collection
			allocCol, err := txApp.FindCollectionByNameOrId("job_time_allocations")
			if err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_finding_collection",
					Message: fmt.Sprintf("error finding job_time_allocations: %v", err),
				}
			}

			// Insert provided allocations
			for _, a := range req.Allocations {
				// Create new allocation via DAO to trigger rules/hooks if any
				rec := core.NewRecord(allocCol)
				rec.Set("job", jobID)
				rec.Set("division", a.Division)
				rec.Set("hours", a.Hours)
				if err := txApp.Save(rec); err != nil {
					httpResponseStatusCode = http.StatusBadRequest
					return &CodeError{
						Code:    "error_creating_allocation",
						Message: fmt.Sprintf("error creating allocation: %v", err),
					}
				}
			}

			return nil
		})

		if err != nil {
			// Check if it's a HookError and return it directly (same format as AnnotateHookError)
			var hookErr *errs.HookError
			if errors.As(err, &hookErr) {
				return e.JSON(httpResponseStatusCode, hookErr)
			}
			// Otherwise handle as CodeError or generic error
			if codeError, ok := err.(*CodeError); ok {
				return e.JSON(httpResponseStatusCode, map[string]any{
					"error": codeError.Message,
					"code":  codeError.Code,
				})
			}
			return e.JSON(httpResponseStatusCode, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]any{"id": jobID})
	}
}

// createCreateJobHandler returns a handler that creates a job and its allocations in a single transaction.
func createCreateJobHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		if authRecord == nil || authRecord.Id == "" {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		var req JobUpsertRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return e.Error(http.StatusBadRequest, "invalid JSON body", err)
		}

		// Authorization: holders of 'job' claim can create
		hasJobClaim, claimErr := utilities.HasClaim(app, authRecord, "job")
		if claimErr != nil {
			return e.Error(http.StatusInternalServerError, "claim check failed", claimErr)
		}
		if !hasJobClaim {
			return apis.NewForbiddenError("you are not authorized to create jobs", nil)
		}

		// Validate allocations
		seen := make(map[string]struct{})
		for _, a := range req.Allocations {
			if a.Division == "" {
				return e.Error(http.StatusBadRequest, "division is required for all allocations", nil)
			}
			if a.Hours < 0 {
				return e.Error(http.StatusBadRequest, "hours must be >= 0", nil)
			}
			if _, ok := seen[a.Division]; ok {
				return e.Error(http.StatusBadRequest, "duplicate division in allocations", nil)
			}
			seen[a.Division] = struct{}{}
		}

		var newJobID string
		var httpResponseStatusCode = http.StatusOK

		err := app.RunInTransaction(func(txApp core.App) error {
			// Create job
			jobsCol, err := txApp.FindCollectionByNameOrId("jobs")
			if err != nil {
				return &CodeError{
					Code:    "error_finding_collection",
					Message: fmt.Sprintf("error finding jobs collection: %v", err),
				}
			}
			jobRec := core.NewRecord(jobsCol)
			if req.Job != nil {
				for k, v := range req.Job {
					switch k {
					case "id", "collectionId", "collectionName", "created", "updated", "divisions":
						// ignore
					default:
						jobRec.Set(k, v)
					}
				}
			}

			// Run job validation and business rules (this will generate the job number)
			if err := hooks.ProcessJobCore(txApp, jobRec, authRecord); err != nil {
				// If it's a HookError, preserve the status code
				var hookErr *errs.HookError
				if errors.As(err, &hookErr) {
					httpResponseStatusCode = hookErr.Status
				} else {
					httpResponseStatusCode = http.StatusBadRequest
				}
				return err
			}

			if err := txApp.Save(jobRec); err != nil {
				httpResponseStatusCode = http.StatusBadRequest
				// Check if it's a validation error with field-level details
				var validationErrs validation.Errors
				if errors.As(err, &validationErrs) {
					// Convert to HookError format for consistent frontend handling
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
					Code:    "error_creating_job",
					Message: fmt.Sprintf("error creating job: %v", err),
				}
			}
			newJobID = jobRec.Id

			// Validate all divisions exist and are active
			for _, a := range req.Allocations {
				divRec, err := txApp.FindRecordById("divisions", a.Division)
				if err != nil {
					httpResponseStatusCode = http.StatusBadRequest
					return &CodeError{
						Code:    "invalid_division",
						Message: fmt.Sprintf("division not found: %s", a.Division),
					}
				}
				if !divRec.GetBool("active") {
					httpResponseStatusCode = http.StatusBadRequest
					return &CodeError{
						Code:    "division_not_active",
						Message: fmt.Sprintf("division is inactive: %s", a.Division),
					}
				}
			}

			// Prepare allocations collection
			allocCol, err := txApp.FindCollectionByNameOrId("job_time_allocations")
			if err != nil {
				httpResponseStatusCode = http.StatusInternalServerError
				return &CodeError{
					Code:    "error_finding_collection",
					Message: fmt.Sprintf("error finding job_time_allocations: %v", err),
				}
			}

			// Create allocations
			for _, a := range req.Allocations {
				rec := core.NewRecord(allocCol)
				rec.Set("job", newJobID)
				rec.Set("division", a.Division)
				rec.Set("hours", a.Hours)
				if err := txApp.Save(rec); err != nil {
					httpResponseStatusCode = http.StatusBadRequest
					return &CodeError{
						Code:    "error_creating_allocation",
						Message: fmt.Sprintf("error creating allocation: %v", err),
					}
				}
			}
			return nil
		})

		if err != nil {
			// Check if it's a HookError and return it directly (same format as AnnotateHookError)
			var hookErr *errs.HookError
			if errors.As(err, &hookErr) {
				return e.JSON(httpResponseStatusCode, hookErr)
			}
			// Otherwise handle as CodeError or generic error
			if codeError, ok := err.(*CodeError); ok {
				return e.JSON(httpResponseStatusCode, map[string]any{
					"error": codeError.Message,
					"code":  codeError.Code,
				})
			}
			return e.JSON(httpResponseStatusCode, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]any{"id": newJobID})
	}
}
