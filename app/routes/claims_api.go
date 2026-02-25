package routes

import (
	_ "embed"
	"net/http"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
)

//go:embed claims_details.sql
var claimDetailsQuery string

type ClaimHolder struct {
	AdminProfileID string `json:"admin_profile_id" db:"admin_profile_id"`
	GivenName      string `json:"given_name" db:"given_name"`
	Surname        string `json:"surname" db:"surname"`
}

type ClaimDetails struct {
	ID          string        `json:"id" db:"id"`
	Name        string        `json:"name" db:"name"`
	Description string        `json:"description" db:"description"`
	Holders     []ClaimHolder `json:"holders"`
}

func createGetClaimDetailsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Check for admin claim
		hasAdminClaim, err := utilities.HasClaim(app, e.Auth, "admin")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to check claims", err)
		}
		if !hasAdminClaim {
			return e.Error(http.StatusForbidden, "you do not have permission to view claim details", nil)
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
