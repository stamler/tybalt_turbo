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

const itClaimName = "it"

type adminProfileIdentityListRow struct {
	ID            string `db:"id" json:"id"`
	UID           string `db:"uid" json:"uid"`
	Active        bool   `db:"active" json:"active"`
	LegacyUID     string `db:"legacy_uid" json:"legacy_uid"`
	Email         string `db:"email" json:"email"`
	Name          string `db:"name" json:"name"`
	GivenName     string `db:"given_name" json:"given_name"`
	Surname       string `db:"surname" json:"surname"`
	ProviderCount int    `db:"provider_count" json:"provider_count"`
}

type authorizedProviderResponse struct {
	ID         string `json:"id"`
	Provider   string `json:"provider"`
	ProviderID string `json:"provider_id"`
	Created    string `json:"created"`
	Updated    string `json:"updated"`
}

type adminProfileIdentityResponse struct {
	ID                  string                       `json:"id"`
	UID                 string                       `json:"uid"`
	LegacyUID           string                       `json:"legacy_uid"`
	Email               string                       `json:"email"`
	Name                string                       `json:"name"`
	GivenName           string                       `json:"given_name"`
	Surname             string                       `json:"surname"`
	AuthorizedProviders []authorizedProviderResponse `json:"authorized_providers"`
}

type saveAdminProfileIdentityRequest struct {
	LegacyUID string `json:"legacy_uid"`
}

func createGetAdminProfileIdentityListHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminProfileIdentityClaim(app, e.Auth); err != nil {
			return err
		}

		usersCollection, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load users collection", err)
		}

		rows := []adminProfileIdentityListRow{}
		err = app.DB().NewQuery(`
			SELECT
				ap.id,
				ap.uid,
				ap.active,
				COALESCE(ap.legacy_uid, '') AS legacy_uid,
				COALESCE(u.email, '') AS email,
				COALESCE(u.name, '') AS name,
				COALESCE(p.given_name, '') AS given_name,
				COALESCE(p.surname, '') AS surname,
				COUNT(ea.id) AS provider_count
			FROM admin_profiles ap
			LEFT JOIN users u ON u.id = ap.uid
			LEFT JOIN profiles p ON p.uid = ap.uid
			LEFT JOIN _externalAuths ea
				ON ea.collectionRef = {:usersCollection}
				AND ea.recordRef = ap.uid
			GROUP BY ap.id
			ORDER BY p.surname, p.given_name, u.email, ap.uid
		`).Bind(dbx.Params{"usersCollection": usersCollection.Id}).All(&rows)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load admin profile identities", err)
		}

		return e.JSON(http.StatusOK, rows)
	}
}

func createGetAdminProfileIdentityHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminProfileIdentityClaim(app, e.Auth); err != nil {
			return err
		}

		recordID := strings.TrimSpace(e.Request.PathValue("id"))
		response, err := adminProfileIdentity(app, recordID)
		if err != nil {
			return adminProfileIdentityError(e, "failed to load admin profile identity", err)
		}

		return e.JSON(http.StatusOK, response)
	}
}

func createSaveAdminProfileIdentityHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminProfileIdentityClaim(app, e.Auth); err != nil {
			return err
		}

		var req saveAdminProfileIdentityRequest
		if err := e.BindBody(&req); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]any{
				"code":    "invalid_request_body",
				"message": "invalid request body",
			})
		}

		recordID := strings.TrimSpace(e.Request.PathValue("id"))
		legacyUID := strings.TrimSpace(req.LegacyUID)

		var response *adminProfileIdentityResponse
		err := app.RunInTransaction(func(txApp core.App) error {
			adminProfile, saveErr := saveAdminProfileLegacyUID(txApp, recordID, legacyUID)
			if saveErr != nil {
				return saveErr
			}
			response, saveErr = adminProfileIdentityFromRecord(txApp, adminProfile)
			return saveErr
		})
		if err != nil {
			return adminProfileIdentityError(e, "failed to save admin profile identity", err)
		}

		return e.JSON(http.StatusOK, response)
	}
}

func createClearAdminProfileAuthorizedProviderHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminProfileIdentityClaim(app, e.Auth); err != nil {
			return err
		}

		recordID := strings.TrimSpace(e.Request.PathValue("id"))
		externalAuthID := strings.TrimSpace(e.Request.PathValue("externalAuthId"))

		var response *adminProfileIdentityResponse
		err := app.RunInTransaction(func(txApp core.App) error {
			adminProfile, clearErr := clearAdminProfileAuthorizedProvider(txApp, recordID, externalAuthID)
			if clearErr != nil {
				return clearErr
			}
			response, clearErr = adminProfileIdentityFromRecord(txApp, adminProfile)
			return clearErr
		})
		if err != nil {
			return adminProfileIdentityError(e, "failed to clear authorized provider", err)
		}

		return e.JSON(http.StatusOK, response)
	}
}

func requireAdminProfileIdentityClaim(app core.App, authRecord *core.Record) error {
	hasAdminClaim, err := utilities.HasClaim(app, authRecord, "admin")
	if err != nil {
		return err
	}
	hasITClaim, err := utilities.HasClaim(app, authRecord, itClaimName)
	if err != nil {
		return err
	}
	if !hasAdminClaim && !hasITClaim {
		return apis.NewForbiddenError("you do not have permission to manage admin profile identity fields", nil)
	}

	return nil
}

func saveAdminProfileLegacyUID(app core.App, recordID string, legacyUID string) (*core.Record, error) {
	adminProfile, err := findAdminProfileForIdentity(app, recordID)
	if err != nil {
		return nil, err
	}

	if legacyUID != "" {
		duplicate, err := app.FindFirstRecordByFilter(
			"admin_profiles",
			"legacy_uid={:legacyUID} && id!={:id}",
			dbx.Params{"legacyUID": legacyUID, "id": adminProfile.Id},
		)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if duplicate != nil {
			return nil, &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: "admin profile identity validation error",
				Data: map[string]errs.CodeError{
					"legacy_uid": {
						Code:    "duplicate_legacy_uid",
						Message: "legacy_uid is already assigned to another admin profile",
					},
				},
			}
		}
	}

	adminProfile.Set("legacy_uid", legacyUID)
	if err := app.Save(adminProfile); err != nil {
		if ve, ok := asValidationErrors(err); ok {
			return nil, ve
		}
		return nil, err
	}

	return adminProfile, nil
}

func clearAdminProfileAuthorizedProvider(app core.App, recordID string, externalAuthID string) (*core.Record, error) {
	adminProfile, err := findAdminProfileForIdentity(app, recordID)
	if err != nil {
		return nil, err
	}
	if externalAuthID == "" {
		return nil, &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "authorized provider validation error",
			Data: map[string]errs.CodeError{
				"external_auth": {
					Code:    "required",
					Message: "authorized provider id is required",
				},
			},
		}
	}

	usersCollection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return nil, err
	}

	externalAuth, err := app.FindFirstExternalAuthByExpr(dbx.HashExp{
		"id":            externalAuthID,
		"collectionRef": usersCollection.Id,
		"recordRef":     adminProfile.GetString("uid"),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &errs.HookError{
				Status:  http.StatusNotFound,
				Message: "authorized provider not found",
				Data: map[string]errs.CodeError{
					"external_auth": {
						Code:    "not_found",
						Message: "authorized provider not found for this admin profile",
					},
				},
			}
		}
		return nil, err
	}

	if err := app.Delete(externalAuth); err != nil {
		return nil, err
	}

	return adminProfile, nil
}

func findAdminProfileForIdentity(app core.App, recordID string) (*core.Record, error) {
	if recordID == "" {
		return nil, &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "admin profile identity validation error",
			Data: map[string]errs.CodeError{
				"id": {
					Code:    "required",
					Message: "admin profile id is required",
				},
			},
		}
	}

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

	return adminProfile, nil
}

func adminProfileIdentity(app core.App, recordID string) (*adminProfileIdentityResponse, error) {
	adminProfile, err := findAdminProfileForIdentity(app, recordID)
	if err != nil {
		return nil, err
	}

	return adminProfileIdentityFromRecord(app, adminProfile)
}

func adminProfileIdentityFromRecord(app core.App, adminProfile *core.Record) (*adminProfileIdentityResponse, error) {
	uid := strings.TrimSpace(adminProfile.GetString("uid"))
	response := &adminProfileIdentityResponse{
		ID:        adminProfile.Id,
		UID:       uid,
		LegacyUID: strings.TrimSpace(adminProfile.GetString("legacy_uid")),
	}

	if uid == "" {
		return response, nil
	}

	userRecord, err := app.FindRecordById("users", uid)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if userRecord != nil {
		response.Email = strings.TrimSpace(userRecord.GetString("email"))
		response.Name = strings.TrimSpace(userRecord.GetString("name"))

		auths, err := app.FindAllExternalAuthsByRecord(userRecord)
		if err != nil {
			return nil, err
		}
		response.AuthorizedProviders = make([]authorizedProviderResponse, 0, len(auths))
		for _, auth := range auths {
			response.AuthorizedProviders = append(response.AuthorizedProviders, authorizedProviderResponse{
				ID:         auth.Id,
				Provider:   auth.Provider(),
				ProviderID: auth.ProviderId(),
				Created:    auth.Created().String(),
				Updated:    auth.Updated().String(),
			})
		}
	}

	profile, err := app.FindFirstRecordByFilter("profiles", "uid={:uid}", dbx.Params{"uid": uid})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if profile != nil {
		response.GivenName = strings.TrimSpace(profile.GetString("given_name"))
		response.Surname = strings.TrimSpace(profile.GetString("surname"))
	}

	return response, nil
}

func adminProfileIdentityError(e *core.RequestEvent, message string, err error) error {
	var hookErr *errs.HookError
	if errors.As(err, &hookErr) {
		return e.JSON(hookErr.Status, hookErr)
	}
	if ve, ok := asValidationErrors(err); ok {
		return apis.NewBadRequestError("Validation error", ve)
	}
	return e.Error(http.StatusInternalServerError, message, err)
}
