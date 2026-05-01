package routes

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"tybalt/errs"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type setAdminProfileActiveRequest struct {
	Active *bool `json:"active"`
}

type adminProfileActiveResponse struct {
	ID     string `json:"id"`
	UID    string `json:"uid"`
	Active bool   `json:"active"`
}

type adminProfileActiveToggleTargetsResponse struct {
	AdminProfileIDs []string `json:"admin_profile_ids"`
}

func createGetAdminProfileActiveToggleTargetsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminProfileActiveClaim(app, e.Auth); err != nil {
			return err
		}

		var rows []struct {
			ID string `db:"id"`
		}
		if err := app.DB().NewQuery(`
			SELECT ap.id
			FROM admin_profiles ap
			WHERE NOT EXISTS (
				SELECT 1
				FROM user_claims uc
				INNER JOIN claims c ON c.id = uc.cid
				WHERE uc.uid = ap.uid
				  AND c.name = 'admin'
			)
			ORDER BY ap.id
		`).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load active toggle targets", err)
		}

		ids := make([]string, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ID)
		}

		return e.JSON(http.StatusOK, adminProfileActiveToggleTargetsResponse{
			AdminProfileIDs: ids,
		})
	}
}

func createSetAdminProfileActiveHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminProfileActiveClaim(app, e.Auth); err != nil {
			return err
		}

		recordID := strings.TrimSpace(e.Request.PathValue("id"))
		if recordID == "" {
			return e.JSON(http.StatusBadRequest, map[string]any{
				"code":    "missing_record_id",
				"message": "record id is required",
			})
		}

		var req setAdminProfileActiveRequest
		if err := e.BindBody(&req); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]any{
				"code":    "invalid_request_body",
				"message": "invalid request body",
			})
		}
		if req.Active == nil {
			return e.JSON(http.StatusBadRequest, map[string]any{
				"code":    "missing_active",
				"message": "active is required",
			})
		}

		var savedRecord *core.Record
		err := app.RunInTransaction(func(txApp core.App) error {
			record, saveErr := setAdminProfileActive(txApp, recordID, *req.Active)
			if saveErr != nil {
				return saveErr
			}
			savedRecord = record
			return nil
		})
		if err != nil {
			var hookErr *errs.HookError
			if errors.As(err, &hookErr) {
				return e.JSON(hookErr.Status, hookErr)
			}
			if ve, ok := asValidationErrors(err); ok {
				return apis.NewBadRequestError("Validation error", ve)
			}
			return e.Error(http.StatusInternalServerError, "failed to set admin profile active state", err)
		}

		return e.JSON(http.StatusOK, adminProfileActiveResponse{
			ID:     savedRecord.Id,
			UID:    savedRecord.GetString("uid"),
			Active: savedRecord.GetBool("active"),
		})
	}
}

func requireAdminProfileActiveClaim(app core.App, authRecord *core.Record) error {
	for _, claimName := range []string{"admin", "it", "hr"} {
		hasClaim, err := utilities.HasClaim(app, authRecord, claimName)
		if err != nil {
			return err
		}
		if hasClaim {
			return nil
		}
	}

	return apis.NewForbiddenError("you do not have permission to set admin profile active state", nil)
}

func setAdminProfileActive(app core.App, recordID string, active bool) (*core.Record, error) {
	adminProfile, err := app.FindRecordById("admin_profiles", recordID)
	if err != nil {
		return nil, &errs.HookError{
			Status:  http.StatusNotFound,
			Message: "admin profile not found",
			Data: map[string]errs.CodeError{
				"id": {
					Code:    "not_found",
					Message: "admin profile not found",
				},
			},
		}
	}

	uid := strings.TrimSpace(adminProfile.GetString("uid"))
	if uid == "" {
		return nil, &errs.HookError{
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

	hasAdminClaim, err := userHasClaim(app, uid, "admin")
	if err != nil {
		return nil, err
	}
	if hasAdminClaim {
		return nil, &errs.HookError{
			Status:  http.StatusForbidden,
			Message: "admin users cannot be activated or deactivated through this endpoint",
			Data: map[string]errs.CodeError{
				"id": {
					Code:    "admin_target_forbidden",
					Message: "admin users cannot be activated or deactivated through this endpoint",
				},
			},
		}
	}

	adminProfile.Set("active", active)
	if err := app.SaveNoValidate(adminProfile); err != nil {
		return nil, err
	}

	return adminProfile, nil
}

func userHasClaim(app core.App, uid string, claimName string) (bool, error) {
	_, err := app.FindFirstRecordByFilter(
		"user_claims",
		"uid={:uid} && cid.name={:claimName}",
		dbx.Params{"uid": uid, "claimName": claimName},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
