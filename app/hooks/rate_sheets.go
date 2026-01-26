package hooks

import (
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

	missingRoles, err := validateRateSheetComplete(app, e.Record.Id)
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

// validateRateSheetComplete checks if a rate sheet has entries for all roles.
// Returns the names of any missing roles, or nil if complete.
func validateRateSheetComplete(app core.App, rateSheetId string) ([]string, error) {
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
