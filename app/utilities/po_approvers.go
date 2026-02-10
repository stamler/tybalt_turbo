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

// GetPOApprovers fetches first- or second-approval candidates for a PO context.
// For second approval, the qualifying po_approver_props limit column depends on
// kind + job-presence.
func GetPOApprovers(
	app core.App,
	auth *core.Record,
	division string,
	amount float64,
	kindID string,
	hasJob bool,
	forSecondApproval bool,
) ([]Approver, bool, error) {
	if division == "" {
		return nil, false, fmt.Errorf("division is required")
	}

	thresholds, err := GetPOApprovalThresholds(app)
	if err != nil {
		return nil, false, fmt.Errorf("error fetching po approval thresholds: %v", err)
	}
	if len(thresholds) == 0 {
		return nil, false, fmt.Errorf("po approval thresholds are not configured")
	}

	if forSecondApproval && amount <= thresholds[0] {
		return []Approver{}, false, nil
	}

	limitColumn := ""
	if forSecondApproval {
		resolvedColumn, resolveErr := ResolvePOApproverLimitColumn(kindID, hasJob)
		if resolveErr != nil {
			return nil, false, resolveErr
		}
		limitColumn = resolvedColumn
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
			INNER JOIN po_approver_props pap ON pap.user_claim = u.id
			WHERE u.uid = {:userId} AND u.cid = {:claimId}
			AND (
				JSON_ARRAY_LENGTH(pap.divisions) = 0
				OR EXISTS (
					SELECT 1
					FROM JSON_EACH(pap.divisions)
					WHERE value = {:division}
				)
			)
		`

		if forSecondApproval {
			query := fmt.Sprintf("%s AND pap.%s >= {:amount}", hasClaimQueryString, limitColumn)
			err = app.DB().NewQuery(query).Bind(dbx.Params{
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
	// the po_approver_props record's max_amount property.

	var approvers []Approver
	// SQL query to find approvers
	approversQueryString := `
		SELECT p.uid AS id, p.given_name, p.surname
		FROM profiles p
		INNER JOIN user_claims u ON p.uid = u.uid
		INNER JOIN po_approver_props pap ON pap.user_claim = u.id
		WHERE u.cid = {:claimId}
		AND (
			JSON_ARRAY_LENGTH(pap.divisions) = 0
			OR EXISTS (
				SELECT 1
				FROM JSON_EACH(pap.divisions)
				WHERE value = {:division}
			)
		)`

	if forSecondApproval {
		query := fmt.Sprintf(`
			%s
			AND pap.%s >= {:amount}
			AND pap.%s <= {:ceiling}
			ORDER BY p.surname, p.given_name
		`, approversQueryString, limitColumn, limitColumn)
		err = app.DB().NewQuery(query).Bind(dbx.Params{
			"claimId":  constants.PO_APPROVER_CLAIM_ID,
			"division": division,
			"amount":   amount,
			"ceiling":  ceiling,
		}).All(&approvers)
	} else {
		err = app.DB().NewQuery(approversQueryString + `
			ORDER BY p.surname, p.given_name
		`).Bind(dbx.Params{
			"claimId":  constants.PO_APPROVER_CLAIM_ID,
			"division": division,
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
