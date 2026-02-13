package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	adminProfilesAugmentedCollectionID = "pbc_697077494"
)

var (
	adminProfilesAugmentedBaseQuery = `SELECT ap.id,
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
  COALESCE(po.po_approver_divisions, '[]') AS po_approver_divisions
FROM admin_profiles ap
LEFT JOIN users u ON u.id = ap.uid
LEFT JOIN profiles p ON u.id = p.uid
LEFT JOIN (
  SELECT
    uc.uid,
    pap.id AS po_approver_props_id,
    pap.max_amount AS po_approver_max_amount,
    pap.divisions AS po_approver_divisions
  FROM user_claims uc
  INNER JOIN claims c ON c.id = uc.cid AND c.name = 'po_approver'
  LEFT JOIN po_approver_props pap ON pap.user_claim = uc.id
) po ON po.uid = ap.uid;`

	adminProfilesAugmentedExpandedQuery = `SELECT ap.id,
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
)

type jsonFieldDef struct {
	ID   string
	Name string
}

var poApproverExpandedFields = []jsonFieldDef{
	{ID: "json1770948800", Name: "po_approver_project_max"},
	{ID: "json1770948801", Name: "po_approver_sponsorship_max"},
	{ID: "json1770948802", Name: "po_approver_staff_and_social_max"},
	{ID: "json1770948803", Name: "po_approver_media_and_event_max"},
	{ID: "json1770948804", Name: "po_approver_computer_max"},
}

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(adminProfilesAugmentedCollectionID)
		if err != nil {
			return err
		}

		collection.ViewQuery = adminProfilesAugmentedExpandedQuery

		for _, fieldDef := range poApproverExpandedFields {
			if hasFieldByName(collection, fieldDef.Name) {
				continue
			}
			fieldJSON := fmt.Sprintf(`{
				"hidden": false,
				"id": "%s",
				"maxSize": 1,
				"name": "%s",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "json"
			}`, fieldDef.ID, fieldDef.Name)
			if err := collection.Fields.AddMarshaledJSONAt(len(collection.Fields), []byte(fieldJSON)); err != nil {
				return err
			}
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(adminProfilesAugmentedCollectionID)
		if err != nil {
			return err
		}

		collection.ViewQuery = adminProfilesAugmentedBaseQuery

		for _, fieldDef := range poApproverExpandedFields {
			collection.Fields.RemoveById(fieldDef.ID)
		}

		return app.Save(collection)
	})
}

func hasFieldByName(collection *core.Collection, fieldName string) bool {
	for _, field := range collection.Fields {
		if field.GetName() == fieldName {
			return true
		}
	}
	return false
}
