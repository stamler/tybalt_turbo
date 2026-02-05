package routes

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"tybalt/hooks"
	"tybalt/utilities"
)

// createActivateRateSheetHandler creates a handler that activates a rate sheet.
// Only users with the 'job' claim can activate rate sheets.
// The rate sheet must have entries for all roles before it can be activated.
func createActivateRateSheetHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")

		// Check for job claim
		hasJobClaim, err := utilities.HasClaim(app, e.Auth, "job")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to check claims", err)
		}
		if !hasJobClaim {
			return e.Error(http.StatusForbidden, "you do not have permission to activate rate sheets", nil)
		}

		// Get the rate sheet
		record, err := app.FindRecordById("rate_sheets", id)
		if err != nil {
			return e.Error(http.StatusNotFound, "rate sheet not found", err)
		}

		// Check if already active
		if record.GetBool("active") {
			return e.JSON(http.StatusOK, map[string]any{
				"message": "rate sheet is already active",
				"id":      id,
				"active":  true,
			})
		}

		// Check if a newer revision exists (cannot activate older revisions)
		name := record.GetString("name")
		revision := record.GetInt("revision")
		newerExists, err := hooks.CheckNewerRevisionExists(app, name, revision)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to check for newer revisions", err)
		}
		if newerExists {
			return e.Error(http.StatusBadRequest, "cannot activate - a newer revision exists", nil)
		}

		// Validate completeness using the existing hook logic
		missingRoles, err := hooks.ValidateRateSheetComplete(app, id)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to validate rate sheet", err)
		}
		if len(missingRoles) > 0 {
			return e.Error(http.StatusBadRequest, "rate sheet is incomplete - missing entries for some roles", map[string]any{
				"missing_roles": missingRoles,
			})
		}

		// Deactivate other revisions with same name BEFORE activating this one
		// (no transaction, so order matters - ensures never two active revisions)
		if err := hooks.DeactivateOtherRevisions(app, name, id); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to deactivate previous revisions", err)
		}

		// Update the record
		record.Set("active", true)
		if err := app.Save(record); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to activate rate sheet", err)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"message": "rate sheet activated",
			"id":      id,
			"active":  true,
		})
	}
}

// createDeactivateRateSheetHandler creates a handler that deactivates a rate sheet.
// Only users with the 'job' claim can deactivate rate sheets.
func createDeactivateRateSheetHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")

		// Check for job claim
		hasJobClaim, err := utilities.HasClaim(app, e.Auth, "job")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to check claims", err)
		}
		if !hasJobClaim {
			return e.Error(http.StatusForbidden, "you do not have permission to deactivate rate sheets", nil)
		}

		// Get the rate sheet
		record, err := app.FindRecordById("rate_sheets", id)
		if err != nil {
			return e.Error(http.StatusNotFound, "rate sheet not found", err)
		}

		// Check if already inactive
		if !record.GetBool("active") {
			return e.JSON(http.StatusOK, map[string]any{
				"message": "rate sheet is already inactive",
				"id":      id,
				"active":  false,
			})
		}

		// Update the record
		record.Set("active", false)
		if err := app.Save(record); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to deactivate rate sheet", err)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"message": "rate sheet deactivated",
			"id":      id,
			"active":  false,
		})
	}
}

// UpdateRateSheetEntryRequest defines the allowed fields for updating a rate sheet entry
type UpdateRateSheetEntryRequest struct {
	Rate         *float64 `json:"rate"`
	OvertimeRate *float64 `json:"overtime_rate"`
}

// createUpdateRateSheetEntryHandler creates a handler that updates a rate sheet entry.
// Only users with the 'admin' claim can update entries.
// Only rate and overtime_rate fields can be updated.
func createUpdateRateSheetEntryHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")

		// Check for admin claim
		hasAdminClaim, err := utilities.HasClaim(app, e.Auth, "admin")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to check claims", err)
		}
		if !hasAdminClaim {
			return e.Error(http.StatusForbidden, "you do not have permission to update rate sheet entries", nil)
		}

		// Parse request body
		var req UpdateRateSheetEntryRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return e.Error(http.StatusBadRequest, "invalid request body", err)
		}

		// Validate at least one field is being updated
		if req.Rate == nil && req.OvertimeRate == nil {
			return e.Error(http.StatusBadRequest, "at least one of rate or overtime_rate must be provided", nil)
		}

		// Get the entry
		record, err := app.FindRecordById("rate_sheet_entries", id)
		if err != nil {
			return e.Error(http.StatusNotFound, "rate sheet entry not found", err)
		}

		// Update only the provided fields
		if req.Rate != nil {
			if *req.Rate < 1 {
				return e.Error(http.StatusBadRequest, "rate must be at least 1", nil)
			}
			// Validate rate is a whole number
			if *req.Rate != math.Floor(*req.Rate) {
				return e.Error(http.StatusBadRequest, "rate must be a whole number", nil)
			}
			record.Set("rate", int(*req.Rate))
		}
		if req.OvertimeRate != nil {
			if *req.OvertimeRate < 1 {
				return e.Error(http.StatusBadRequest, "overtime_rate must be at least 1", nil)
			}
			record.Set("overtime_rate", *req.OvertimeRate)
		}

		// Save the record
		if err := app.Save(record); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to update rate sheet entry", err)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"message":       "rate sheet entry updated",
			"id":            id,
			"rate":          record.GetInt("rate"),
			"overtime_rate": record.GetFloat("overtime_rate"),
		})
	}
}

// CreateRateSheetEntry defines a single entry in the rate sheet creation request
type CreateRateSheetEntry struct {
	Role         string  `json:"role"`
	Rate         float64 `json:"rate"`
	OvertimeRate float64 `json:"overtime_rate"`
}

// CreateRateSheetRequest defines the request body for creating a rate sheet with entries
type CreateRateSheetRequest struct {
	Name          string                 `json:"name"`
	EffectiveDate string                 `json:"effective_date"`
	Revision      int                    `json:"revision"`
	Entries       []CreateRateSheetEntry `json:"entries"`
}

type entryValidationError struct {
	index  int
	errors validation.Errors
}

func (e *entryValidationError) Error() string {
	return "entry validation error"
}

type codeError interface {
	Code() string
}

func setValidationError(errs validation.Errors, key string, err error) {
	if err == nil {
		return
	}
	if _, exists := errs[key]; !exists {
		errs[key] = err
	}
}

func rateSheetUniqueValidationErrors() validation.Errors {
	return validation.Errors{
		"name": validation.NewError("validation_not_unique", "name and revision must be unique"),
	}
}

func hasUniqueValidationError(errs validation.Errors) bool {
	for _, fieldErr := range errs {
		if fieldErr == nil {
			continue
		}
		if ce, ok := fieldErr.(codeError); ok && ce.Code() == "validation_not_unique" {
			return true
		}
		if strings.Contains(strings.ToLower(fieldErr.Error()), "unique") {
			return true
		}
	}
	return false
}

func asValidationErrors(err error) (validation.Errors, bool) {
	if err == nil {
		return nil, false
	}
	if ve, ok := err.(validation.Errors); ok {
		return ve, true
	}
	if ve, ok := err.(*validation.Errors); ok && ve != nil {
		return *ve, true
	}
	return nil, false
}

func prefixEntryErrors(index int, errs validation.Errors) validation.Errors {
	prefixed := validation.Errors{}
	for field, fieldErr := range errs {
		prefixed[fmt.Sprintf("entries.%d.%s", index, field)] = fieldErr
	}
	return prefixed
}

// createRateSheetHandler creates a handler that atomically creates a rate sheet
// and all its entries in a single transaction. This prevents partial-save states
// where the rate sheet exists but some entries are missing.
//
// Permission model:
//   - revision == 0 (new/copy): requires "job" claim
//   - revision > 0 (revise): requires "rate_sheet_revise" or "admin" claim
func createRateSheetHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Parse request body
		var req CreateRateSheetRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return e.Error(http.StatusBadRequest, "invalid request body", err)
		}

		// Check claims based on revision
		if req.Revision == 0 {
			// New rate sheet or copy: requires "job" claim
			hasJobClaim, err := utilities.HasClaim(app, e.Auth, "job")
			if err != nil {
				return e.Error(http.StatusInternalServerError, "failed to check claims", err)
			}
			if !hasJobClaim {
				return e.Error(http.StatusForbidden, "you do not have permission to create rate sheets", nil)
			}
		} else {
			// Revision: requires "rate_sheet_revise" or "admin" claim
			hasReviseClaim, err := utilities.HasClaim(app, e.Auth, "rate_sheet_revise")
			if err != nil {
				return e.Error(http.StatusInternalServerError, "failed to check claims", err)
			}
			hasAdminClaim, err := utilities.HasClaim(app, e.Auth, "admin")
			if err != nil {
				return e.Error(http.StatusInternalServerError, "failed to check claims", err)
			}
			if !hasReviseClaim && !hasAdminClaim {
				return e.Error(http.StatusForbidden, "you do not have permission to revise rate sheets", nil)
			}
		}

		validationErrors := validation.Errors{}

		// Validate name
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			setValidationError(validationErrors, "name", validation.NewError("value_required", "name is required"))
		}

		// Validate effective_date format (YYYY-MM-DD)
		req.EffectiveDate = strings.TrimSpace(req.EffectiveDate)
		datePattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
		if req.EffectiveDate == "" {
			setValidationError(validationErrors, "effective_date", validation.NewError("value_required", "effective date is required"))
		} else if !datePattern.MatchString(req.EffectiveDate) {
			setValidationError(validationErrors, "effective_date", validation.NewError("validation_invalid_date", "effective date must be in YYYY-MM-DD format"))
		} else if err := utilities.IsValidDate(req.EffectiveDate); err != nil {
			setValidationError(validationErrors, "effective_date", err)
		}

		// Validate revision
		if req.Revision < 0 {
			setValidationError(validationErrors, "revision", validation.NewError("validation_negative_number", "revision must be 0 or greater"))
		}

		// Validate entries
		if len(req.Entries) == 0 {
			setValidationError(validationErrors, "entries", validation.NewError("value_required", "at least one entry is required"))
		}

		// Validate each entry
		seenRoles := map[string]bool{}
		for i, entry := range req.Entries {
			entryPrefix := fmt.Sprintf("entries.%d.", i)
			if entry.Role == "" {
				setValidationError(validationErrors, entryPrefix+"role", validation.NewError("value_required", "entry role is required"))
			} else if seenRoles[entry.Role] {
				setValidationError(validationErrors, entryPrefix+"role", validation.NewError("validation_not_unique", "entry role must be unique"))
			} else {
				seenRoles[entry.Role] = true
			}
			if entry.Rate < 1 {
				setValidationError(validationErrors, entryPrefix+"rate", validation.NewError("validation_negative_number", "entry rate must be at least 1"))
			} else if entry.Rate != math.Floor(entry.Rate) {
				setValidationError(validationErrors, entryPrefix+"rate", validation.NewError("validation_not_whole_number", "entry rate must be a whole number"))
			}
			if entry.OvertimeRate < 1 {
				setValidationError(validationErrors, entryPrefix+"overtime_rate", validation.NewError("validation_negative_number", "entry overtime_rate must be at least 1"))
			}
		}

		if len(validationErrors) > 0 {
			return apis.NewBadRequestError("Validation error", validationErrors)
		}

		// Validate revision effective date (for revisions, must be >= previous revision's date)
		if req.Revision > 0 {
			prevEffectiveDate, err := hooks.CheckRevisionEffectiveDate(app, req.Name, req.Revision, req.EffectiveDate)
			if err != nil {
				return e.Error(http.StatusInternalServerError, "failed to validate effective date", err)
			}
			if prevEffectiveDate != "" {
				return e.Error(http.StatusBadRequest, "effective date must be on or after the previous revision's effective date", map[string]any{
					"previous_effective_date": prevEffectiveDate,
				})
			}
		}

		// Run in transaction for atomicity
		var createdSheetId string
		var entriesCreated int

		err := app.RunInTransaction(func(txApp core.App) error {
			// Get the rate_sheets collection
			rateSheetsCollection, err := txApp.FindCollectionByNameOrId("rate_sheets")
			if err != nil {
				return err
			}

			// Check for existing rate sheet to provide a consistent conflict response
			existingSheets, err := txApp.FindRecordsByFilter(
				"rate_sheets",
				"name={:name} && revision={:revision}",
				"",
				1,
				0,
				dbx.Params{"name": req.Name, "revision": req.Revision},
			)
			if err != nil {
				return err
			}
			if len(existingSheets) > 0 {
				return &uniqueConstraintError{
					message: "a rate sheet with this name and revision already exists",
					data:    rateSheetUniqueValidationErrors(),
				}
			}

			// Create the rate sheet record
			rateSheet := core.NewRecord(rateSheetsCollection)
			rateSheet.Set("name", req.Name)
			rateSheet.Set("effective_date", req.EffectiveDate)
			rateSheet.Set("revision", req.Revision)
			rateSheet.Set("active", false)

			if err := txApp.Save(rateSheet); err != nil {
				return err
			}

			createdSheetId = rateSheet.Id

			// Get the rate_sheet_entries collection
			entriesCollection, err := txApp.FindCollectionByNameOrId("rate_sheet_entries")
			if err != nil {
				return err
			}

			// Create all entries
			for i, entry := range req.Entries {
				entryRecord := core.NewRecord(entriesCollection)
				entryRecord.Set("rate_sheet", createdSheetId)
				entryRecord.Set("role", entry.Role)
				entryRecord.Set("rate", int(entry.Rate))
				entryRecord.Set("overtime_rate", entry.OvertimeRate)

				if err := txApp.Save(entryRecord); err != nil {
					if ve, ok := asValidationErrors(err); ok {
						return &entryValidationError{index: i, errors: ve}
					}
					if strings.Contains(err.Error(), "UNIQUE constraint failed") {
						return &entryValidationError{
							index: i,
							errors: validation.Errors{
								"role": validation.NewError("validation_not_unique", "entry role must be unique"),
							},
						}
					}
					return err
				}
				entriesCreated++
			}

			return nil
		})

		if err != nil {
			// Check if it's our custom unique constraint error
			if uce, ok := err.(*uniqueConstraintError); ok {
				return apis.NewApiError(http.StatusConflict, uce.message, uce.data)
			}
			if eve, ok := err.(*entryValidationError); ok {
				return apis.NewBadRequestError("Validation error", prefixEntryErrors(eve.index, eve.errors))
			}
			if ve, ok := asValidationErrors(err); ok {
				if hasUniqueValidationError(ve) {
					return apis.NewApiError(http.StatusConflict, "a rate sheet with this name and revision already exists", ve)
				}
				return apis.NewBadRequestError("Validation error", ve)
			}
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return apis.NewApiError(http.StatusConflict, "a rate sheet with this name and revision already exists", rateSheetUniqueValidationErrors())
			}
			return e.Error(http.StatusInternalServerError, "failed to create rate sheet", err)
		}

		return e.JSON(http.StatusCreated, map[string]any{
			"id":              createdSheetId,
			"name":            req.Name,
			"revision":        req.Revision,
			"entries_created": entriesCreated,
		})
	}
}

// uniqueConstraintError is a custom error type for unique constraint violations
type uniqueConstraintError struct {
	message string
	data    validation.Errors
}

func (e *uniqueConstraintError) Error() string {
	return e.message
}
