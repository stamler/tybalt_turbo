package routes

import (
	"encoding/json"
	"net/http"
	"tybalt/constants"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type poApprovalLimitsRow struct {
	ID                string   `db:"id" json:"id"`
	GivenName         string   `db:"given_name" json:"given_name"`
	Surname           string   `db:"surname" json:"surname"`
	Configured        bool     `db:"configured" json:"configured"`
	MaxAmount         float64  `db:"max_amount" json:"max_amount"`
	ProjectMax        float64  `db:"project_max" json:"project_max"`
	SponsorshipMax    float64  `db:"sponsorship_max" json:"sponsorship_max"`
	StaffAndSocialMax float64  `db:"staff_and_social_max" json:"staff_and_social_max"`
	MediaAndEventMax  float64  `db:"media_and_event_max" json:"media_and_event_max"`
	ComputerMax       float64  `db:"computer_max" json:"computer_max"`
	DivisionJSON      string   `db:"division_json" json:"-"`
	Divisions         []string `json:"divisions"`
}

type poApprovalLimitCategory struct {
	ID                      string  `db:"id" json:"id"`
	Label                   string  `db:"label" json:"label"`
	LimitField              string  `json:"limit_field"`
	SecondApprovalThreshold float64 `db:"second_approval_threshold" json:"second_approval_threshold"`
}

type poApprovalLimitDivision struct {
	ID     string `db:"id" json:"id"`
	Code   string `db:"code" json:"code"`
	Name   string `db:"name" json:"name"`
	Active bool   `db:"active" json:"active"`
}

type poApprovalLimitsResponse struct {
	Items      []poApprovalLimitsRow     `json:"items"`
	Categories []poApprovalLimitCategory `json:"categories"`
	Divisions  []poApprovalLimitDivision `json:"divisions"`
}

func createGetPOApprovalLimitsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		allowed, err := utilities.HasClaim(app, e.Auth, "report")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to check approval-limit access", err)
		}
		if !allowed {
			return e.ForbiddenError("you do not have permission to view PO approval limits", nil)
		}

		result := poApprovalLimitsResponse{
			Items: []poApprovalLimitsRow{}, Categories: []poApprovalLimitCategory{}, Divisions: []poApprovalLimitDivision{},
		}
		// Read only the fields needed by this report. Do not expose admin profiles
		// or grant access to the editable approval properties collection.
		err = app.DB().NewQuery(`
			SELECT uc.id, p.given_name, p.surname,
				pap.id IS NOT NULL AS configured,
				COALESCE(pap.max_amount, 0) AS max_amount,
				COALESCE(pap.project_max, 0) AS project_max,
				COALESCE(pap.sponsorship_max, 0) AS sponsorship_max,
				COALESCE(pap.staff_and_social_max, 0) AS staff_and_social_max,
				COALESCE(pap.media_and_event_max, 0) AS media_and_event_max,
				COALESCE(pap.computer_max, 0) AS computer_max,
				COALESCE(pap.divisions, '[]') AS division_json
			FROM user_claims uc
			JOIN users u ON u.id = uc.uid
			JOIN profiles p ON p.uid = u.id
			JOIN admin_profiles ap ON ap.uid = u.id AND ap.active = 1
			LEFT JOIN po_approver_props pap ON pap.user_claim = uc.id
			WHERE uc.cid = {:claim}
			ORDER BY p.surname COLLATE NOCASE, p.given_name COLLATE NOCASE, uc.id
		`).Bind(dbx.Params{"claim": constants.PO_APPROVER_CLAIM_ID}).All(&result.Items)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load PO approval limits", err)
		}
		for i := range result.Items {
			row := &result.Items[i]
			row.Divisions = []string{}
			if err := json.Unmarshal([]byte(row.DivisionJSON), &row.Divisions); err != nil {
				return e.Error(http.StatusInternalServerError, "failed to read PO approval divisions", err)
			}
			if row.Divisions == nil {
				row.Divisions = []string{}
			}
		}

		if err := app.DB().NewQuery(`
			SELECT id, en_ui_label AS label, second_approval_threshold
			FROM expenditure_kinds ORDER BY ui_order, id
		`).All(&result.Categories); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load PO approval categories", err)
		}
		for i := range result.Categories {
			category := &result.Categories[i]
			field, err := utilities.ResolvePOApproverLimitColumn(category.ID, false)
			if err != nil {
				return e.Error(http.StatusInternalServerError, "failed to resolve PO approval category", err)
			}
			category.LimitField = field
		}
		if err := app.DB().NewQuery(`SELECT id, code, name, active FROM divisions ORDER BY code, name`).All(&result.Divisions); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load PO approval divisions", err)
		}
		return e.JSON(http.StatusOK, result)
	}
}
