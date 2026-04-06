package hooks

import (
	"net/http"
	"strings"
	"tybalt/errs"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// EnforceAdminProfileRequestPermissions keeps direct collection creates
// admin-only. Limited editors update through the custom route layer.
func EnforceAdminProfileRequestPermissions(app core.App, e *core.RecordRequestEvent) error {
	hasAdminClaim, err := utilities.HasClaim(app, e.Auth, "admin")
	if err != nil {
		return err
	}
	if hasAdminClaim {
		return nil
	}

	return apis.NewForbiddenError("you do not have permission to edit admin profiles directly", nil)
}

// ProcessAdminProfile enforces business rules for admin_profiles create/update.
func ProcessAdminProfile(app core.App, e *core.RecordRequestEvent) error {
	record := e.Record
	uid := strings.TrimSpace(record.GetString("uid"))
	if uid == "" {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "admin profile validation error",
			Data: map[string]errs.CodeError{
				"uid": {
					Code:    "required",
					Message: "uid is required",
				},
			},
		}
	}

	openingDate := strings.TrimSpace(record.GetString("opening_date"))
	openingOP := record.GetFloat("opening_op")
	openingOV := record.GetFloat("opening_ov")
	if (openingOP != 0 || openingOV != 0) && openingDate == "" {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "admin profile validation error",
			Data: map[string]errs.CodeError{
				"opening_date": {
					Code:    "invalid_opening_date",
					Message: "opening_date is required when opening balances are non-zero",
				},
			},
		}
	}

	if openingDate != "" {
		if err := utilities.ValidateTimeOffOpeningDate(openingDate); err != nil {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "admin profile validation error",
				Data: map[string]errs.CodeError{
					"opening_date": {
						Code:    "invalid_opening_date",
						Message: "opening_date must be the Sunday after a pay period ending date",
					},
				},
			}
		}
	}

	return utilities.EnsureUserCanUseBranch(app, record.GetString("default_branch"), uid, "default_branch")
}
