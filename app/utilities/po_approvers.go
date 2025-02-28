package utilities

import (
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Approver represents a user who can approve purchase orders
type Approver struct {
	ID        string `db:"id" json:"id"`
	GivenName string `db:"given_name" json:"given_name"`
	Surname   string `db:"surname" json:"surname"`
}

// GetApproversByTier fetches a list of users who can approve a purchase order based on specified parameters.
//
// This function encapsulates the logic for finding both first-level approvers and second-level approvers
// for purchase orders, based on approval tiers and division permissions.
//
// How it works:
//  1. It first identifies the required claim for the given purchase order amount by checking the po_approval_tiers collection
//  2. It then finds users who have this claim and have permission for the specified division
//  3. Division permission is determined by the claim's payload - if the payload is empty (null, [], {}, or "),
//     the user has permission for all divisions; otherwise, the division must be in the payload
//  4. For second approvers, it only returns users with the appropriate tier claim that is higher than the minimum tier
//  5. It also identifies if the authenticated user (if provided) is among the eligible approvers
//
// Parameters:
// - app: the application context used to access the database and other core services
// - auth: the authenticated user record (optional, nil is valid) - when provided, checks if this user is among eligible approvers
// - division: the division ID for which approval is needed
// - amount: the purchase order amount used to determine the required approval tier
// - forSecondApproval: boolean flag indicating whether we're looking for second approvers (true) or first approvers (false)
//
// Returns:
// - []Approver: list of eligible approvers with their basic information
// - bool: whether the current user (auth) is among the eligible approvers (always false if auth is nil)
// - error: any error that occurred during the operation
func GetApproversByTier(
	app core.App,
	auth *core.Record,
	division string,
	amount float64,
	forSecondApproval bool,
) ([]Approver, bool, error) {
	if division == "" {
		return nil, false, fmt.Errorf("division is required")
	}

	// Determine the required claim based on amount
	// This is the same logic used in getSecondApproverClaimId
	requiredClaimId, err := FindRequiredApproverClaimIdForPOAmount(app, amount)
	if err != nil {
		return nil, false, fmt.Errorf("error determining approval tier: %v", err)
	}

	// Get the lowest tier claim ID and max amount (needed for both first and second approvers)
	// By default, set the target claim to the lowest tier (used for first approvers)
	targetClaimId, lowestTierMaxAmount, err := GetBoundClaimIdAndMaxAmount(app, false)
	if err != nil {
		return nil, false, fmt.Errorf("error determining lowest approval tier: %v", err)
	}

	// Special handling for second approvers
	if forSecondApproval {
		// If the amount is below the lowest tier max amount, return empty list
		// because no second approval is needed
		if amount <= lowestTierMaxAmount {
			return []Approver{}, false, nil
		}

		// For second approvers, override the target claim with the required claim
		targetClaimId = requiredClaimId
	}

	// Early check if the authenticated user has the target claim
	// If they do, return empty list (UI will auto-set to self)
	if auth != nil {
		// Check if the auth user has the required claim
		type ClaimResult struct {
			HasClaim bool `db:"has_claim"`
		}
		var result ClaimResult
		err = app.DB().NewQuery(`
			SELECT COUNT(*) > 0 AS has_claim
			FROM user_claims
			WHERE uid = {:userId} AND cid = {:claimId}
		`).Bind(dbx.Params{
			"userId":  auth.Id,
			"claimId": targetClaimId,
		}).One(&result)

		if err != nil {
			return nil, false, fmt.Errorf("error checking user claims: %v", err)
		}

		if result.HasClaim {
			// User has the required claim, return empty list and indicate they are an approver
			return []Approver{}, true, nil
		}
	}

	// Find users who have the required claim
	var approvers []Approver
	authIsApprover := false

	// Build the query to find users with the target claim and division permission
	// Note: Division permission is determined by checking if the payload is empty (null, [], {}, or '')
	// or if it contains the specified division
	query := app.DB().NewQuery(`
		SELECT p.uid AS id, p.given_name, p.surname
		FROM profiles p
		INNER JOIN user_claims u ON p.uid = u.uid
		WHERE u.cid = {:claimId} 
		AND (u.payload IS NULL OR u.payload = '[]' OR u.payload = '{}' OR u.payload = '' OR u.payload = 'null' OR JSON_EXTRACT(u.payload, '$') LIKE {:divisionPattern})
		ORDER BY p.surname, p.given_name
	`)

	// Execute the query with the appropriate parameters
	if err := query.Bind(dbx.Params{
		"claimId":         targetClaimId,
		"divisionPattern": "%\"" + division + "\"%",
	}).All(&approvers); err != nil {
		return nil, false, fmt.Errorf("error finding approvers: %v", err)
	}

	// Check if auth user is among the approvers
	if auth != nil {
		authUserId := auth.Id
		for _, approver := range approvers {
			if approver.ID == authUserId {
				authIsApprover = true
				break
			}
		}
	}

	return approvers, authIsApprover, nil
}
