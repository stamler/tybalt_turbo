package utilities

import (
	"errors"
	"fmt"
	"strings"
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

type approverWithLimit struct {
	Approver
	LimitValue float64 `db:"limit_value"`
}

var ErrUnknownExpenditureKind = errors.New("unknown expenditure kind")

// POApproverPolicy captures eligibility and stage pools for a concrete PO
// context (division + amount + kind + hasJob).
type POApproverPolicy struct {
	SecondApprovalThreshold float64
	SecondApprovalRequired  bool
	LimitColumn             string
	FirstStageApprovers     []Approver
	SecondStageApprovers    []Approver
	firstStageLimits        map[string]float64
	secondStageLimits       map[string]float64
}

func (p POApproverPolicy) IsFirstStageApprover(userID string) bool {
	_, ok := p.firstStageLimits[userID]
	return ok
}

func (p POApproverPolicy) IsSecondStageApprover(userID string) bool {
	_, ok := p.secondStageLimits[userID]
	return ok
}

func (p POApproverPolicy) LimitForSecondStageApprover(userID string) (float64, bool) {
	v, ok := p.secondStageLimits[userID]
	return v, ok
}

func (p POApproverPolicy) HasSufficientFinalLimit(userID string, amount float64) bool {
	limit, ok := p.LimitForSecondStageApprover(userID)
	return ok && limit >= amount
}

func IsSecondApprovalRequired(secondApprovalThreshold float64, amount float64) bool {
	return secondApprovalThreshold > 0 && amount > secondApprovalThreshold
}

func GetKindSecondApprovalThreshold(app core.App, kindID string) (float64, error) {
	kindID = strings.TrimSpace(kindID)
	if kindID == "" {
		return 0, fmt.Errorf("%w: blank", ErrUnknownExpenditureKind)
	}
	type result struct {
		SecondApprovalThreshold float64 `db:"second_approval_threshold"`
	}
	rows := []result{}
	if err := app.DB().NewQuery(`
		SELECT COALESCE(second_approval_threshold, 0) AS second_approval_threshold
		FROM expenditure_kinds
		WHERE id = {:kindId}
	`).Bind(dbx.Params{
		"kindId": kindID,
	}).All(&rows); err != nil {
		return 0, fmt.Errorf("error fetching expenditure kind second approval threshold: %w", err)
	}
	if len(rows) == 0 {
		return 0, fmt.Errorf("%w: %s", ErrUnknownExpenditureKind, kindID)
	}
	return rows[0].SecondApprovalThreshold, nil
}

func GetPOApproverPolicy(
	app core.App,
	division string,
	amount float64,
	kindID string,
	hasJob bool,
) (POApproverPolicy, error) {
	if strings.TrimSpace(division) == "" {
		return POApproverPolicy{}, fmt.Errorf("division is required")
	}
	kindID = strings.TrimSpace(kindID)
	if kindID == "" {
		return POApproverPolicy{}, fmt.Errorf("%w: blank", ErrUnknownExpenditureKind)
	}

	limitColumn, err := ResolvePOApproverLimitColumn(kindID, hasJob)
	if err != nil {
		return POApproverPolicy{}, err
	}

	secondApprovalThreshold, err := GetKindSecondApprovalThreshold(app, kindID)
	if err != nil {
		return POApproverPolicy{}, err
	}
	secondApprovalRequired := IsSecondApprovalRequired(secondApprovalThreshold, amount)

	query := fmt.Sprintf(`
		SELECT
			p.uid AS id,
			p.given_name,
			p.surname,
			COALESCE(pap.%s, 0) AS limit_value
		FROM profiles p
		INNER JOIN users u ON p.uid = u.id
		INNER JOIN admin_profiles ap ON ap.uid = u.id
		INNER JOIN user_claims uc ON p.uid = uc.uid
		INNER JOIN po_approver_props pap ON pap.user_claim = uc.id
		WHERE
			uc.cid = {:claimId}
			AND ap.active = 1
			AND pap.%s IS NOT NULL
			AND (
				JSON_ARRAY_LENGTH(pap.divisions) = 0
				OR EXISTS (
					SELECT 1
					FROM JSON_EACH(pap.divisions)
					WHERE value = {:division}
				)
			)
		ORDER BY p.surname, p.given_name
	`, limitColumn, limitColumn)

	var eligible []approverWithLimit
	if err := app.DB().NewQuery(query).Bind(dbx.Params{
		"claimId":  constants.PO_APPROVER_CLAIM_ID,
		"division": division,
	}).All(&eligible); err != nil {
		return POApproverPolicy{}, fmt.Errorf("error finding approvers: %w", err)
	}

	policy := POApproverPolicy{
		SecondApprovalThreshold: secondApprovalThreshold,
		SecondApprovalRequired:  secondApprovalRequired,
		LimitColumn:             limitColumn,
		FirstStageApprovers:     make([]Approver, 0, len(eligible)),
		SecondStageApprovers:    make([]Approver, 0, len(eligible)),
		firstStageLimits:        map[string]float64{},
		secondStageLimits:       map[string]float64{},
	}

	for _, candidate := range eligible {
		if !secondApprovalRequired {
			// Single-stage approvals are final approvals, so the selected approver
			// must be able to fully approve the amount.
			if candidate.LimitValue < amount {
				continue
			}
			policy.FirstStageApprovers = append(policy.FirstStageApprovers, candidate.Approver)
			policy.firstStageLimits[candidate.ID] = candidate.LimitValue
			continue
		}

		// Dual-stage pools are intentionally disjoint:
		// first-stage limits are <= threshold, while second-stage limits must be
		// > threshold and also >= amount. Candidates between threshold and amount
		// are excluded from both pools because they can neither start nor finish.
		if candidate.LimitValue <= secondApprovalThreshold {
			policy.FirstStageApprovers = append(policy.FirstStageApprovers, candidate.Approver)
			policy.firstStageLimits[candidate.ID] = candidate.LimitValue
			continue
		}

		// Second-stage pool must be able to fully approve this amount.
		if candidate.LimitValue < amount {
			continue
		}

		policy.SecondStageApprovers = append(policy.SecondStageApprovers, candidate.Approver)
		policy.secondStageLimits[candidate.ID] = candidate.LimitValue
	}

	return policy, nil
}
