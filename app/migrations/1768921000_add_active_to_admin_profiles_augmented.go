package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_697077494")
		if err != nil {
			return err
		}

		// Update the view query to include the active field
		collection.ViewQuery = `SELECT ap.id,
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

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_697077494")
		if err != nil {
			return err
		}

		// Restore the original view query without active field
		collection.ViewQuery = `SELECT ap.id,
  ap.uid,
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

		return app.Save(collection)
	})
}
