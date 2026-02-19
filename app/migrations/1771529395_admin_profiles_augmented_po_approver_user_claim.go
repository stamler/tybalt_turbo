package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	adminProfilesAugmentedWithPOApproverUserClaimIDQuery = `SELECT ap.id,
  ap.uid,
  ap.active,
  ap.work_week_hours,
  ap.salary,
  ap.default_charge_out_rate,
  ap.off_rotation_permitted,
  ap.skip_min_time_check,
  ap.opening_date,
  ap.opening_op,
  ap.opening_ov,
  ap.payroll_id,
  ap.untracked_time_off,
  ap.time_sheet_expected,
  ap.allow_personal_reimbursement,
  ap.mobile_phone,
  ap.job_title,
  ap.personal_vehicle_insurance_expiry,
  ap.default_branch,
  p.given_name,
  p.surname,
  po.po_approver_props_id,
  po.po_approver_user_claim_id,
  po.po_approver_max_amount,
  po.po_approver_project_max,
  po.po_approver_sponsorship_max,
  po.po_approver_staff_and_social_max,
  po.po_approver_media_and_event_max,
  po.po_approver_computer_max,
  COALESCE(po.po_approver_divisions, '[]') AS po_approver_divisions
FROM admin_profiles ap
LEFT JOIN users u ON u.id = ap.uid
LEFT JOIN profiles p ON u.id = p.uid
LEFT JOIN (
  SELECT
    uc.uid,
    uc.id AS po_approver_user_claim_id,
    pap.id AS po_approver_props_id,
    pap.max_amount AS po_approver_max_amount,
    pap.project_max AS po_approver_project_max,
    pap.sponsorship_max AS po_approver_sponsorship_max,
    pap.staff_and_social_max AS po_approver_staff_and_social_max,
    pap.media_and_event_max AS po_approver_media_and_event_max,
    pap.computer_max AS po_approver_computer_max,
    pap.divisions AS po_approver_divisions
  FROM user_claims uc
  INNER JOIN claims c ON c.id = uc.cid AND c.name = 'po_approver'
  LEFT JOIN po_approver_props pap ON pap.user_claim = uc.id
) po ON po.uid = ap.uid;`

	adminProfilesAugmentedWithoutPOApproverUserClaimIDQuery = `SELECT ap.id,
  ap.uid,
  ap.active,
  ap.work_week_hours,
  ap.salary,
  ap.default_charge_out_rate,
  ap.off_rotation_permitted,
  ap.skip_min_time_check,
  ap.opening_date,
  ap.opening_op,
  ap.opening_ov,
  ap.payroll_id,
  ap.untracked_time_off,
  ap.time_sheet_expected,
  ap.allow_personal_reimbursement,
  ap.mobile_phone,
  ap.job_title,
  ap.personal_vehicle_insurance_expiry,
  ap.default_branch,
  p.given_name,
  p.surname,
  po.po_approver_props_id,
  po.po_approver_max_amount,
  po.po_approver_project_max,
  po.po_approver_sponsorship_max,
  po.po_approver_staff_and_social_max,
  po.po_approver_media_and_event_max,
  po.po_approver_computer_max,
  COALESCE(po.po_approver_divisions, '[]') AS po_approver_divisions
FROM admin_profiles ap
LEFT JOIN users u ON u.id = ap.uid
LEFT JOIN profiles p ON u.id = p.uid
LEFT JOIN (
  SELECT
    uc.uid,
    pap.id AS po_approver_props_id,
    pap.max_amount AS po_approver_max_amount,
    pap.project_max AS po_approver_project_max,
    pap.sponsorship_max AS po_approver_sponsorship_max,
    pap.staff_and_social_max AS po_approver_staff_and_social_max,
    pap.media_and_event_max AS po_approver_media_and_event_max,
    pap.computer_max AS po_approver_computer_max,
    pap.divisions AS po_approver_divisions
  FROM user_claims uc
  INNER JOIN claims c ON c.id = uc.cid AND c.name = 'po_approver'
  LEFT JOIN po_approver_props pap ON pap.user_claim = uc.id
) po ON po.uid = ap.uid;`

	poApproverUserClaimIDFieldID   = "json1771529395"
	poApproverUserClaimIDFieldName = "po_approver_user_claim_id"
	poApproverUserClaimIDFieldJSON = `{
		"hidden": false,
		"id": "json1771529395",
		"maxSize": 1,
		"name": "po_approver_user_claim_id",
		"presentable": false,
		"required": false,
		"system": false,
		"type": "json"
	}`
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_697077494")
		if err != nil {
			return err
		}

		collection.ViewQuery = adminProfilesAugmentedWithPOApproverUserClaimIDQuery

		if !hasFieldByNameForPOApproverUserClaimID(collection, poApproverUserClaimIDFieldName) {
			if err := collection.Fields.AddMarshaledJSONAt(
				len(collection.Fields),
				[]byte(poApproverUserClaimIDFieldJSON),
			); err != nil {
				return err
			}
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_697077494")
		if err != nil {
			return err
		}

		collection.ViewQuery = adminProfilesAugmentedWithoutPOApproverUserClaimIDQuery
		collection.Fields.RemoveById(poApproverUserClaimIDFieldID)

		return app.Save(collection)
	})
}

func hasFieldByNameForPOApproverUserClaimID(collection *core.Collection, fieldName string) bool {
	for _, field := range collection.Fields {
		if field.GetName() == fieldName {
			return true
		}
	}
	return false
}
