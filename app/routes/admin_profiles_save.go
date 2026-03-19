package routes

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"tybalt/errs"
	"tybalt/hooks"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type saveAdminProfileWithClaimsRequest struct {
	AdminProfile map[string]any `json:"admin_profile"`
	ClaimIDs     []string       `json:"claim_ids"`
}

func createSaveAdminProfileWithClaimsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		hasAdminClaim, err := utilities.HasClaim(app, e.Auth, "admin")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to check claims", err)
		}
		if !hasAdminClaim {
			return apis.NewForbiddenError("you do not have permission to save admin profiles with claims", nil)
		}

		var req saveAdminProfileWithClaimsRequest
		if err := e.BindBody(&req); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]any{
				"code":    "invalid_request_body",
				"message": "invalid request body",
			})
		}

		recordID := strings.TrimSpace(e.Request.PathValue("id"))
		var savedRecord *core.Record
		err = app.RunInTransaction(func(txApp core.App) error {
			record, saveErr := saveAdminProfileWithClaims(txApp, e.Auth, recordID, req)
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
			return e.Error(http.StatusInternalServerError, "failed to save admin profile with claims", err)
		}

		status := http.StatusOK
		if recordID == "" {
			status = http.StatusCreated
		}
		return e.JSON(status, savedRecord)
	}
}

func saveAdminProfileWithClaims(
	app core.App,
	authRecord *core.Record,
	recordID string,
	req saveAdminProfileWithClaimsRequest,
) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId("admin_profiles")
	if err != nil {
		return nil, err
	}

	var record *core.Record
	originalUID := ""
	if recordID == "" {
		record = core.NewRecord(collection)
	} else {
		record, err = app.FindRecordById("admin_profiles", recordID)
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
		originalUID = strings.TrimSpace(record.GetString("uid"))
	}

	if req.AdminProfile == nil {
		req.AdminProfile = map[string]any{}
	}
	record.Load(req.AdminProfile)

	uid := strings.TrimSpace(record.GetString("uid"))
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
	if originalUID != "" && uid != originalUID {
		return nil, &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "admin profile validation error",
			Data: map[string]errs.CodeError{
				"uid": {
					Code:    "immutable_field",
					Message: "uid cannot be changed",
				},
			},
		}
	}

	claimIDs := normalizeDistinctRecordIDs(req.ClaimIDs)
	if err := syncUserClaims(app, uid, claimIDs); err != nil {
		return nil, err
	}

	if err := hooks.ProcessAdminProfile(app, &core.RecordRequestEvent{
		RequestEvent: &core.RequestEvent{App: app, Auth: authRecord},
		Record:       record,
	}); err != nil {
		return nil, err
	}

	if err := app.Save(record); err != nil {
		if ve, ok := asValidationErrors(err); ok {
			return nil, ve
		}
		return nil, err
	}

	return record, nil
}

func normalizeDistinctRecordIDs(ids []string) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || slices.Contains(result, id) {
			continue
		}
		result = append(result, id)
	}
	return result
}

func syncUserClaims(app core.App, uid string, desiredClaimIDs []string) error {
	currentClaims, err := app.FindRecordsByFilter(
		"user_claims",
		"uid={:uid}",
		"",
		0,
		0,
		dbx.Params{"uid": uid},
	)
	if err != nil {
		return err
	}

	desiredSet := make(map[string]struct{}, len(desiredClaimIDs))
	currentByClaimID := make(map[string]*core.Record, len(currentClaims))
	for _, claimID := range desiredClaimIDs {
		desiredSet[claimID] = struct{}{}
	}
	for _, current := range currentClaims {
		currentByClaimID[strings.TrimSpace(current.GetString("cid"))] = current
	}

	for _, current := range currentClaims {
		claimID := strings.TrimSpace(current.GetString("cid"))
		if _, keep := desiredSet[claimID]; keep {
			continue
		}
		if err := app.Delete(current); err != nil {
			return err
		}
	}

	if len(desiredClaimIDs) == 0 {
		return nil
	}

	collection, err := app.FindCollectionByNameOrId("user_claims")
	if err != nil {
		return err
	}
	for _, claimID := range desiredClaimIDs {
		if _, exists := currentByClaimID[claimID]; exists {
			continue
		}
		record := core.NewRecord(collection)
		record.Set("uid", uid)
		record.Set("cid", claimID)
		if err := app.Save(record); err != nil {
			if ve, ok := asValidationErrors(err); ok {
				return ve
			}
			return err
		}
	}

	return nil
}
