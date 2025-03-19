package utilities

import (
	"fmt"
	"tybalt/constants"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Approver represents a user who can approve purchase orders
type Approver struct {
	ID        string `db:"id" json:"id"`
	GivenName string `db:"given_name" json:"given_name"`
	Surname   string `db:"surname" json:"surname"`
}

// GetPOApprovers fetches a list of users who can approve a purchase order based
// on specified parameters.
//
// This function encapsulates the logic for finding both first-level approvers
// and second-level approvers for purchase orders, based on the payload of a
// user's po_approver claim and provided division.
//
// How it works:
// First the function checks if the authenticated user has the right to approve
// the purchase order based on the claim payload's max_amount property at the
// requested approval level (first or second). If the authenticated user has the
// right to approve, the function returns an empty list, indicating that the
// user is an approver. It also returns true in the second return value to
// indicate that the user is an approver.
//
// If forSecondApproval is true, it returns users who have the
// constants.PO_APPROVER_CLAIM_ID with a payload that has a max_amount greater
// than or equal to the purchase_orders records' approval_total AND less than or
// equal to the ceiling, AND the claim payload's division property is missing,
// or is a list that contains the provided division. If the amount is less than
// or equal to the lowest threshold returned by GetPOApprovalThresholds, it
// returns an empty list.
//
// If forSecondApproval is false, it returns users who have the
// constants.PO_APPROVER_CLAIM_ID claim with a payload that is either missing
// max_amount property or has a max_amount less than or equal to the lowest
// threshold returned by GetPOApprovalThresholds AND the claim payload's
// division property is missing, or is a list that contains the provided
// division.
//
// Parameters:
//   - app: the application context used to access the database and
//     other core services
//   - auth: the authenticated user record (optional, nil is
//     valid) - when provided, checks if this user is among eligible approvers
//   - division: the division ID for which approval is needed
//   - amount: the purchase order amount used to determine the required approval tier
//   - forSecondApproval: boolean flag indicating whether we're looking for second
//     approvers (true) or first approvers (false)
//
// Returns:
//   - []Approver: list of eligible approvers with their basic
//     information
//   - bool: whether the current user (auth) is among the eligible
//     approvers (always false if auth is nil)
//   - error: any error that occurred during the operation
func GetPOApprovers(
	app core.App,
	auth *core.Record,
	division string,
	amount float64,
	forSecondApproval bool,
) ([]Approver, bool, error) {
	if division == "" {
		return nil, false, fmt.Errorf("division is required")
	}

	thresholds, err := GetPOApprovalThresholds(app)
	if err != nil {
		return nil, false, fmt.Errorf("error fetching po approval thresholds: %v", err)
	}

	// Find the value of the lowest threshold that is greater than or equal to
	// the amount and store it in the ceiling variable. If no such threshold
	// exists, set the ceiling to constants.MAX_APPROVAL_TOTAL.
	ceiling := constants.MAX_APPROVAL_TOTAL
	for _, threshold := range thresholds {
		if threshold >= amount {
			ceiling = threshold
			break
		}
	}

	// Check if the authenticated user has approval permission. If they do, return
	// empty list (UI will auto-set to self)
	if auth != nil {
		// Check if the auth user has the required claim
		type ClaimResult struct {
			HasClaim bool `db:"has_claim"`
		}
		var result ClaimResult

		hasClaimQueryString := `
			SELECT COUNT(*) > 0 AS has_claim
			FROM user_claims u
			WHERE u.uid = {:userId} AND u.cid = {:claimId}
			AND (
				JSON_EXTRACT(u.payload, '$.divisions') IS NULL
				OR EXISTS (
					SELECT 1
					FROM JSON_EACH(JSON_EXTRACT(u.payload, '$.divisions'))
					WHERE value = {:division}
				)
			)
		`

		if forSecondApproval {
			err = app.DB().NewQuery(hasClaimQueryString + `
				AND JSON_EXTRACT(u.payload, '$.max_amount') >= {:amount}
			`).Bind(dbx.Params{
				"userId":   auth.Id,
				"claimId":  constants.PO_APPROVER_CLAIM_ID,
				"division": division,
				"amount":   amount,
			}).One(&result)
		} else {
			err = app.DB().NewQuery(hasClaimQueryString).Bind(dbx.Params{
				"userId":   auth.Id,
				"claimId":  constants.PO_APPROVER_CLAIM_ID,
				"division": division,
				"amount":   thresholds[0],
			}).One(&result)
		}

		if err != nil {
			return nil, false, fmt.Errorf("error checking user claims: %v", err)
		}

		if result.HasClaim {
			// User has the required claim, return empty list and indicate they are an approver
			return []Approver{}, true, nil
		}
	}

	// The authenticated user is not an approver, or the function is being called
	// without an authenticated user (e.g. from the UI when the user is not logged
	// in). In this case, we need to find the approvers based on the amount and
	// the claim payload's max_amount property.

	var approvers []Approver
	// SQL query to find approvers
	approversQueryString := `
		SELECT p.uid AS id, p.given_name, p.surname
		FROM profiles p
		INNER JOIN user_claims u ON p.uid = u.uid
		WHERE u.cid = {:claimId}
		AND (
			JSON_EXTRACT(u.payload, '$.divisions') IS NULL
			OR EXISTS (
				SELECT 1
				FROM JSON_EACH(JSON_EXTRACT(u.payload, '$.divisions'))
				WHERE value = {:division}
			)
		)`

	if forSecondApproval {
		// return an empty list if the amount is less than or equal to the lowest
		// threshold returned by GetPOApprovalThresholds.
		if amount <= thresholds[0] {
			return []Approver{}, false, nil
		}

		err = app.DB().NewQuery(approversQueryString + `
			AND JSON_EXTRACT(u.payload, '$.max_amount') >= {:amount}
			AND JSON_EXTRACT(u.payload, '$.max_amount') <= {:ceiling}
			ORDER BY p.surname, p.given_name
		`).Bind(dbx.Params{
			"claimId":  constants.PO_APPROVER_CLAIM_ID,
			"division": division,
			"amount":   amount,
			"ceiling":  ceiling,
		}).All(&approvers)
	} else {
		err = app.DB().NewQuery(approversQueryString + `
			AND (
				JSON_EXTRACT(u.payload, '$.max_amount') <= {:amount}
				OR JSON_EXTRACT(u.payload, '$.max_amount') IS NULL
			)
			ORDER BY p.surname, p.given_name
		`).Bind(dbx.Params{
			"claimId":  constants.PO_APPROVER_CLAIM_ID,
			"division": division,
			"amount":   thresholds[0],
			"ceiling":  ceiling,
		}).All(&approvers)
	}

	if err != nil {
		return nil, false, fmt.Errorf("error finding approvers: %v", err)
	}

	return approvers, false, nil
}

// GetPOApprovalThresholds fetches an ordered (ascending) slice of all
// thresholds from the po_approval_thresholds table.
func GetPOApprovalThresholds(app core.App) ([]float64, error) {
	thresholds := []struct {
		Threshold float64 `db:"threshold"`
	}{}
	err := app.DB().NewQuery("SELECT threshold FROM po_approval_thresholds ORDER BY threshold ASC").All(&thresholds)
	if err != nil {
		return nil, fmt.Errorf("error fetching po_approval_thresholds: %v", err)
	}
	threshold_floats := make([]float64, len(thresholds))
	for i, threshold := range thresholds {
		threshold_floats[i] = threshold.Threshold
	}
	return threshold_floats, nil
}
