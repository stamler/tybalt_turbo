package routes

import (
	"encoding/json"
	"math"
	"net/http"

	"tybalt/hooks"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
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
