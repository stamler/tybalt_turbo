package hooks

import (
	"fmt"
	"net/http"

	"tybalt/errs"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// ProcessRateSheet validates rate sheet changes. When a rate sheet is being
// marked as active, it ensures that there is a corresponding rate_sheet_entries
// record for every role in rate_roles. This prevents incomplete rate sheets
// from being activated.
func ProcessRateSheet(app core.App, e *core.RecordRequestEvent) error {
	// Only validate when setting active=true
	newActive := e.Record.GetBool("active")
	if !newActive {
		return nil
	}

	// On update, skip validation if active isn't changing to true
	if !e.Record.IsNew() {
		originalActive := e.Record.Original().GetBool("active")
		if originalActive {
			// Already active, no need to re-validate
			return nil
		}
	}

	missingRoles, err := ValidateRateSheetComplete(app, e.Record.Id)
	if err != nil {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "failed to validate rate sheet entries",
			Data: map[string]errs.CodeError{
				"active": {Code: "validation_error", Message: "unable to verify rate sheet completeness"},
			},
		}
	}

	if len(missingRoles) > 0 {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "rate sheet is missing entries for some roles",
			Data: map[string]errs.CodeError{
				"active": {
					Code:    "incomplete_rate_sheet",
					Message: "rate sheet must have entries for all roles before activation",
					Data:    map[string]any{"missing_roles": missingRoles},
				},
			},
		}
	}

	return nil
}

// ValidateRateSheetComplete checks if a rate sheet has entries for all roles.
// Returns the names of any missing roles, or nil if complete.
func ValidateRateSheetComplete(app core.App, rateSheetId string) ([]string, error) {
	var missingRoleNames []string

	err := app.DB().NewQuery(`
		SELECT r.name
		FROM rate_roles r
		WHERE r.id NOT IN (
			SELECT rse.role
			FROM rate_sheet_entries rse
			WHERE rse.rate_sheet = {:rateSheetId}
		)
		ORDER BY r.name
	`).Bind(dbx.Params{
		"rateSheetId": rateSheetId,
	}).Column(&missingRoleNames)

	if err != nil {
		return nil, err
	}

	return missingRoleNames, nil
}

// ValidateRateSheetEffectiveDate validates that for a revised rate sheet,
// the effective_date is on or after the effective_date of the previous revision.
// This only applies to new rate sheets with revision > 0.
func ValidateRateSheetEffectiveDate(app core.App, e *core.RecordRequestEvent) error {
	// Only validate on create
	if !e.Record.IsNew() {
		return nil
	}

	revision := e.Record.GetInt("revision")
	// Skip validation for first revision (revision 0)
	if revision == 0 {
		return nil
	}

	name := e.Record.GetString("name")
	newEffectiveDate := e.Record.GetString("effective_date")

	prevEffectiveDate, err := CheckRevisionEffectiveDate(app, name, revision, newEffectiveDate)
	if err != nil {
		return err
	}
	if prevEffectiveDate != "" {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "effective date must be on or after the previous revision's effective date",
			Data: map[string]errs.CodeError{
				"effective_date": {
					Code:    "invalid_effective_date",
					Message: fmt.Sprintf("effective date must be on or after %s (previous revision's date)", prevEffectiveDate),
				},
			},
		}
	}

	return nil
}

// CheckRevisionEffectiveDate checks if a new rate sheet revision has a valid effective date.
// Returns the previous revision's effective_date if the new date is invalid (earlier than previous),
// or empty string if valid. Returns error only for database failures.
func CheckRevisionEffectiveDate(app core.App, name string, revision int, newEffectiveDate string) (string, error) {
	// Skip validation for first revision (revision 0)
	if revision == 0 {
		return "", nil
	}

	// Find the previous revision's effective_date
	var prevEffectiveDate string
	err := app.DB().NewQuery(`
		SELECT effective_date
		FROM rate_sheets
		WHERE name = {:name} AND revision = {:prevRevision}
	`).Bind(dbx.Params{
		"name":         name,
		"prevRevision": revision - 1,
	}).Row(&prevEffectiveDate)

	if err != nil {
		// No previous revision found - this shouldn't happen but allow it
		return "", nil
	}

	// Compare dates (YYYY-MM-DD format, so string comparison works)
	if newEffectiveDate < prevEffectiveDate {
		return prevEffectiveDate, nil
	}

	return "", nil
}
