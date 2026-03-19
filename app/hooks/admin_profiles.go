package hooks

import (
	"net/http"
	"strings"
	"tybalt/errs"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
)

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

	return utilities.EnsureUserCanUseBranch(app, record.GetString("default_branch"), uid, "default_branch")
}
