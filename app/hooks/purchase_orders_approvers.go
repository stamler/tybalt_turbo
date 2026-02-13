package hooks

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"tybalt/errs"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// ProcessPOApproverProps enforces rules for po_approver_props updates.
func ProcessPOApproverProps(app core.App, e *core.RecordRequestEvent) error {
	record := e.Record

	divisionsRaw := record.Get("divisions")

	var divisions []string
	switch v := divisionsRaw.(type) {
	case types.JSONRaw:
		if len(v) > 0 {
			if err := json.Unmarshal(v, &divisions); err != nil {
				return &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "division validation error",
					Data: map[string]errs.CodeError{
						"divisions": {Code: "invalid_json", Message: "divisions must be an array of strings"},
					},
				}
			}
		}
	case []string:
		divisions = v
	case []any:
		divisions = make([]string, 0, len(v))
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				return &errs.HookError{
					Status:  http.StatusBadRequest,
					Message: "division validation error",
					Data: map[string]errs.CodeError{
						"divisions": {Code: "invalid_json", Message: "divisions must be an array of strings"},
					},
				}
			}
			divisions = append(divisions, str)
		}
	case nil:
		// no divisions provided, treat as empty list
	default:
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "divisions must be a JSON array",
			Data: map[string]errs.CodeError{
				"divisions": {Code: "invalid_type", Message: "divisions must be a JSON array"},
			},
		}
	}

	for _, id := range divisions {
		if err := ensureActiveDivision(app, id, "divisions"); err != nil {
			return err
		}
	}

	userClaimID := strings.TrimSpace(record.GetString("user_claim"))
	if userClaimID == "" {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "po_approver_props validation error",
			Data: map[string]errs.CodeError{
				"user_claim": {Code: "required", Message: "user_claim is required"},
			},
		}
	}

	existing, err := app.FindFirstRecordByFilter(
		"po_approver_props",
		"user_claim = {:userClaim}",
		dbx.Params{"userClaim": userClaimID},
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "po_approver_props validation failed",
			Data: map[string]errs.CodeError{
				"user_claim": {
					Code:    "validation_query_failed",
					Message: "unable to validate user_claim uniqueness",
				},
			},
		}
	}
	if existing != nil && existing.Id != record.Id {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "po_approver_props validation error",
			Data: map[string]errs.CodeError{
				"user_claim": {
					Code:    "duplicate_user_claim",
					Message: "po_approver_props already exists for this user_claim",
				},
			},
		}
	}

	return nil
}
