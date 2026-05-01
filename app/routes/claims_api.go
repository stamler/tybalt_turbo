package routes

import (
	_ "embed"
	"net/http"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed claims_list.sql
var claimListQuery string

//go:embed claims_details.sql
var claimDetailsQuery string

//go:embed claim_assignable_users.sql
var claimAssignableUsersQuery string

type ClaimListItem struct {
	ID          string `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description" db:"description"`
	HolderCount int    `json:"holder_count" db:"holder_count"`
}

type ClaimHolder struct {
	AdminProfileID string `json:"admin_profile_id" db:"admin_profile_id"`
	GivenName      string `json:"given_name" db:"given_name"`
	Surname        string `json:"surname" db:"surname"`
}

type ClaimAssignableUser struct {
	ID             string `json:"id" db:"id"`
	AdminProfileID string `json:"admin_profile_id" db:"admin_profile_id"`
	GivenName      string `json:"given_name" db:"given_name"`
	Surname        string `json:"surname" db:"surname"`
	Name           string `json:"name" db:"name"`
	Username       string `json:"username" db:"username"`
	Email          string `json:"email" db:"email"`
}

type ClaimDetails struct {
	ID          string        `json:"id" db:"id"`
	Name        string        `json:"name" db:"name"`
	Description string        `json:"description" db:"description"`
	Holders     []ClaimHolder `json:"holders"`
}

type ClaimAssignableUsers struct {
	ID              string                `json:"id" db:"id"`
	Name            string                `json:"name" db:"name"`
	Description     string                `json:"description" db:"description"`
	AssignableUsers []ClaimAssignableUser `json:"assignable_users"`
}

type bulkAssignClaimRequest struct {
	UIDs []string `json:"uids"`
}

type bulkAssignClaimResponse struct {
	AssignedCount int `json:"assigned_count"`
	SkippedCount  int `json:"skipped_count"`
}

func requireAdminClaimForClaims(app core.App, e *core.RequestEvent, action string) error {
	hasAdminClaim, err := utilities.HasClaim(app, e.Auth, "admin")
	if err != nil {
		return e.Error(http.StatusInternalServerError, "failed to check claims", err)
	}
	if !hasAdminClaim {
		return e.Error(http.StatusForbidden, "you do not have permission to "+action, nil)
	}
	return nil
}

func createGetClaimDetailsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminClaimForClaims(app, e, "view claim details"); err != nil {
			return err
		}

		id := e.Request.PathValue("id")

		// Get the claim record
		claim, err := app.FindRecordById("claims", id)
		if err != nil {
			return e.Error(http.StatusNotFound, "claim not found", err)
		}

		// Get users who hold this claim via SQL
		var holders []ClaimHolder
		if err := app.DB().NewQuery(claimDetailsQuery).Bind(map[string]any{
			"claimId": id,
		}).All(&holders); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query claim holders", err)
		}

		return e.JSON(http.StatusOK, ClaimDetails{
			ID:          claim.Id,
			Name:        claim.GetString("name"),
			Description: claim.GetString("description"),
			Holders:     holders,
		})
	}
}

func createGetClaimAssignableUsersHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminClaimForClaims(app, e, "view claim assignment details"); err != nil {
			return err
		}

		id := e.Request.PathValue("id")
		claim, err := app.FindRecordById("claims", id)
		if err != nil {
			return e.Error(http.StatusNotFound, "claim not found", err)
		}

		var assignableUsers []ClaimAssignableUser
		if err := app.DB().NewQuery(claimAssignableUsersQuery).Bind(map[string]any{
			"claimId": id,
		}).All(&assignableUsers); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query claim assignable users", err)
		}

		return e.JSON(http.StatusOK, ClaimAssignableUsers{
			ID:              claim.Id,
			Name:            claim.GetString("name"),
			Description:     claim.GetString("description"),
			AssignableUsers: assignableUsers,
		})
	}
}

func createBulkAssignClaimHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminClaimForClaims(app, e, "bulk assign claims"); err != nil {
			return err
		}

		claimID := e.Request.PathValue("id")
		if _, err := app.FindRecordById("claims", claimID); err != nil {
			return e.Error(http.StatusNotFound, "claim not found", err)
		}

		var req bulkAssignClaimRequest
		if err := e.BindBody(&req); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]any{
				"code":    "invalid_request_body",
				"message": "invalid request body",
			})
		}

		uids := normalizeDistinctRecordIDs(req.UIDs)
		var response bulkAssignClaimResponse
		err := app.RunInTransaction(func(txApp core.App) error {
			assigned, skipped, err := bulkAssignClaimToUsers(txApp, claimID, uids)
			response.AssignedCount = assigned
			response.SkippedCount = skipped
			return err
		})
		if err != nil {
			return e.Error(http.StatusBadRequest, "failed to bulk assign claim", err)
		}

		return e.JSON(http.StatusOK, response)
	}
}

func bulkAssignClaimToUsers(app core.App, claimID string, uids []string) (int, int, error) {
	if len(uids) == 0 {
		return 0, 0, nil
	}

	userClaims, err := app.FindCollectionByNameOrId("user_claims")
	if err != nil {
		return 0, 0, err
	}

	assigned := 0
	skipped := 0
	for _, uid := range uids {
		if _, err := app.FindRecordById("users", uid); err != nil {
			return assigned, skipped, err
		}

		existingClaims, err := app.FindRecordsByFilter(
			"user_claims",
			"uid={:uid} && cid={:cid}",
			"",
			1,
			0,
			dbx.Params{"uid": uid, "cid": claimID},
		)
		if err != nil {
			return assigned, skipped, err
		}
		if len(existingClaims) > 0 {
			skipped++
			continue
		}

		record := core.NewRecord(userClaims)
		record.Set("uid", uid)
		record.Set("cid", claimID)
		if err := app.Save(record); err != nil {
			return assigned, skipped, err
		}
		assigned++
	}

	return assigned, skipped, nil
}

func createGetClaimsListHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var items []ClaimListItem
		if err := app.DB().NewQuery(claimListQuery).All(&items); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query claims", err)
		}
		return e.JSON(http.StatusOK, items)
	}
}
