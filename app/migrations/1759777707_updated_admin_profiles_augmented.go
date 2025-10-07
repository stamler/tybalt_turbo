package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_697077494")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"viewQuery": "SELECT ap.id,\n  ap.uid,\n  ap.work_week_hours,\n  ap.salary,\n  ap.default_charge_out_rate,\n  ap.off_rotation_permitted,\n  ap.skip_min_time_check,\n  ap.opening_date,\n  ap.opening_op,\n  ap.opening_ov,\n  ap.payroll_id,\n  ap.untracked_time_off,\n  ap.time_sheet_expected,\n  ap.allow_personal_reimbursement,\n  ap.mobile_phone,\n  ap.job_title,\n  ap.personal_vehicle_insurance_expiry,\n  ap.default_branch,\n  p.given_name,\n  p.surname,\n  po.po_approver_props_id,\n  po.po_approver_max_amount,\n  COALESCE(po.po_approver_divisions, '[]') AS po_approver_divisions\nFROM admin_profiles ap\nLEFT JOIN users u ON u.id = ap.uid\nLEFT JOIN profiles p ON u.id = p.uid\nLEFT JOIN (\n  SELECT\n    uc.uid,\n    pap.id AS po_approver_props_id,\n    pap.max_amount AS po_approver_max_amount,\n    pap.divisions AS po_approver_divisions\n  FROM user_claims uc\n  INNER JOIN claims c ON c.id = uc.cid AND c.name = 'po_approver'\n  LEFT JOIN po_approver_props pap ON pap.user_claim = uc.id\n) po ON po.uid = ap.uid;"
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_HWWB")

		// remove field
		collection.Fields.RemoveById("_clone_XSwc")

		// remove field
		collection.Fields.RemoveById("_clone_4EJe")

		// remove field
		collection.Fields.RemoveById("_clone_BzQp")

		// remove field
		collection.Fields.RemoveById("_clone_AJnS")

		// remove field
		collection.Fields.RemoveById("_clone_KbtE")

		// remove field
		collection.Fields.RemoveById("_clone_ba6Y")

		// remove field
		collection.Fields.RemoveById("_clone_ypQx")

		// remove field
		collection.Fields.RemoveById("_clone_mH1f")

		// remove field
		collection.Fields.RemoveById("_clone_VRrt")

		// remove field
		collection.Fields.RemoveById("_clone_nVxK")

		// remove field
		collection.Fields.RemoveById("_clone_zyxF")

		// remove field
		collection.Fields.RemoveById("_clone_NHZw")

		// remove field
		collection.Fields.RemoveById("_clone_xSGl")

		// remove field
		collection.Fields.RemoveById("_clone_KJgf")

		// remove field
		collection.Fields.RemoveById("_clone_3ZMB")

		// remove field
		collection.Fields.RemoveById("_clone_Jwc3")

		// remove field
		collection.Fields.RemoveById("_clone_gP1P")

		// remove field
		collection.Fields.RemoveById("_clone_q05M")

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_bEiY",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "uid",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(2, []byte(`{
			"hidden": false,
			"id": "_clone_qdPa",
			"max": 40,
			"min": 8,
			"name": "work_week_hours",
			"onlyInt": false,
			"presentable": false,
			"required": true,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(3, []byte(`{
			"hidden": false,
			"id": "_clone_JQZF",
			"name": "salary",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(4, []byte(`{
			"hidden": false,
			"id": "_clone_04wv",
			"max": 1000,
			"min": 50,
			"name": "default_charge_out_rate",
			"onlyInt": false,
			"presentable": false,
			"required": true,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(5, []byte(`{
			"hidden": false,
			"id": "_clone_pJn9",
			"name": "off_rotation_permitted",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(6, []byte(`{
			"hidden": false,
			"id": "_clone_S5WE",
			"maxSelect": 1,
			"name": "skip_min_time_check",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "select",
			"values": [
				"no",
				"on_next_bundle",
				"yes"
			]
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(7, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_YUDg",
			"max": 0,
			"min": 0,
			"name": "opening_date",
			"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(8, []byte(`{
			"hidden": false,
			"id": "_clone_Gnks",
			"max": 332,
			"min": 0,
			"name": "opening_op",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(9, []byte(`{
			"hidden": false,
			"id": "_clone_ezmh",
			"max": 200,
			"min": 0,
			"name": "opening_ov",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(10, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_eYoG",
			"max": 0,
			"min": 0,
			"name": "payroll_id",
			"pattern": "^(?:[1-9]\\d*|CMS[0-9]{1,2})$",
			"presentable": false,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(11, []byte(`{
			"hidden": false,
			"id": "_clone_MaXD",
			"name": "untracked_time_off",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(12, []byte(`{
			"hidden": false,
			"id": "_clone_kUJw",
			"name": "time_sheet_expected",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(13, []byte(`{
			"hidden": false,
			"id": "_clone_FpoS",
			"name": "allow_personal_reimbursement",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(14, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_Gp0j",
			"max": 0,
			"min": 0,
			"name": "mobile_phone",
			"pattern": "^\\+1 \\(\\d{3}\\) \\d{3}-\\d{4}$",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(15, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_V4E8",
			"max": 0,
			"min": 0,
			"name": "job_title",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(16, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_ZH3B",
			"max": 0,
			"min": 0,
			"name": "personal_vehicle_insurance_expiry",
			"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(17, []byte(`{
			"cascadeDelete": false,
			"collectionId": "pbc_2536409462",
			"hidden": false,
			"id": "_clone_u2PN",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "default_branch",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(18, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_ayTa",
			"max": 48,
			"min": 2,
			"name": "given_name",
			"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$",
			"presentable": false,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(19, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_gZOk",
			"max": 48,
			"min": 2,
			"name": "surname",
			"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$",
			"presentable": false,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(20, []byte(`{
			"hidden": false,
			"id": "json122688194",
			"maxSize": 1,
			"name": "po_approver_props_id",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "json"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_697077494")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"viewQuery": "SELECT ap.id,\n  ap.uid,\n  ap.work_week_hours,\n  ap.salary,\n  ap.default_charge_out_rate,\n  ap.off_rotation_permitted,\n  ap.skip_min_time_check,\n  ap.opening_date,\n  ap.opening_op,\n  ap.opening_ov,\n  ap.payroll_id,\n  ap.untracked_time_off,\n  ap.time_sheet_expected,\n  ap.allow_personal_reimbursement,\n  ap.mobile_phone,\n  ap.job_title,\n  ap.personal_vehicle_insurance_expiry,\n  ap.default_branch,\n  p.given_name,\n  p.surname,\n  po.po_approver_max_amount,\n  COALESCE(po.po_approver_divisions, '[]') AS po_approver_divisions\nFROM admin_profiles ap\nLEFT JOIN users u ON u.id = ap.uid\nLEFT JOIN profiles p ON u.id = p.uid\nLEFT JOIN (\n  SELECT\n    uc.uid,\n    pap.max_amount AS po_approver_max_amount,\n    pap.divisions AS po_approver_divisions\n  FROM user_claims uc\n  INNER JOIN claims c ON c.id = uc.cid AND c.name = 'po_approver'\n  LEFT JOIN po_approver_props pap ON pap.user_claim = uc.id\n) po ON po.uid = ap.uid;"
		}`), &collection); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_HWWB",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "uid",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(2, []byte(`{
			"hidden": false,
			"id": "_clone_XSwc",
			"max": 40,
			"min": 8,
			"name": "work_week_hours",
			"onlyInt": false,
			"presentable": false,
			"required": true,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(3, []byte(`{
			"hidden": false,
			"id": "_clone_4EJe",
			"name": "salary",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(4, []byte(`{
			"hidden": false,
			"id": "_clone_BzQp",
			"max": 1000,
			"min": 50,
			"name": "default_charge_out_rate",
			"onlyInt": false,
			"presentable": false,
			"required": true,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(5, []byte(`{
			"hidden": false,
			"id": "_clone_AJnS",
			"name": "off_rotation_permitted",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(6, []byte(`{
			"hidden": false,
			"id": "_clone_KbtE",
			"maxSelect": 1,
			"name": "skip_min_time_check",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "select",
			"values": [
				"no",
				"on_next_bundle",
				"yes"
			]
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(7, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_ba6Y",
			"max": 0,
			"min": 0,
			"name": "opening_date",
			"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(8, []byte(`{
			"hidden": false,
			"id": "_clone_ypQx",
			"max": 332,
			"min": 0,
			"name": "opening_op",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(9, []byte(`{
			"hidden": false,
			"id": "_clone_mH1f",
			"max": 200,
			"min": 0,
			"name": "opening_ov",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(10, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_VRrt",
			"max": 0,
			"min": 0,
			"name": "payroll_id",
			"pattern": "^(?:[1-9]\\d*|CMS[0-9]{1,2})$",
			"presentable": false,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(11, []byte(`{
			"hidden": false,
			"id": "_clone_nVxK",
			"name": "untracked_time_off",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(12, []byte(`{
			"hidden": false,
			"id": "_clone_zyxF",
			"name": "time_sheet_expected",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(13, []byte(`{
			"hidden": false,
			"id": "_clone_NHZw",
			"name": "allow_personal_reimbursement",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(14, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_xSGl",
			"max": 0,
			"min": 0,
			"name": "mobile_phone",
			"pattern": "^\\+1 \\(\\d{3}\\) \\d{3}-\\d{4}$",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(15, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_KJgf",
			"max": 0,
			"min": 0,
			"name": "job_title",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(16, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_3ZMB",
			"max": 0,
			"min": 0,
			"name": "personal_vehicle_insurance_expiry",
			"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(17, []byte(`{
			"cascadeDelete": false,
			"collectionId": "pbc_2536409462",
			"hidden": false,
			"id": "_clone_Jwc3",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "default_branch",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(18, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_gP1P",
			"max": 48,
			"min": 2,
			"name": "given_name",
			"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$",
			"presentable": false,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(19, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_q05M",
			"max": 48,
			"min": 2,
			"name": "surname",
			"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$",
			"presentable": false,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_bEiY")

		// remove field
		collection.Fields.RemoveById("_clone_qdPa")

		// remove field
		collection.Fields.RemoveById("_clone_JQZF")

		// remove field
		collection.Fields.RemoveById("_clone_04wv")

		// remove field
		collection.Fields.RemoveById("_clone_pJn9")

		// remove field
		collection.Fields.RemoveById("_clone_S5WE")

		// remove field
		collection.Fields.RemoveById("_clone_YUDg")

		// remove field
		collection.Fields.RemoveById("_clone_Gnks")

		// remove field
		collection.Fields.RemoveById("_clone_ezmh")

		// remove field
		collection.Fields.RemoveById("_clone_eYoG")

		// remove field
		collection.Fields.RemoveById("_clone_MaXD")

		// remove field
		collection.Fields.RemoveById("_clone_kUJw")

		// remove field
		collection.Fields.RemoveById("_clone_FpoS")

		// remove field
		collection.Fields.RemoveById("_clone_Gp0j")

		// remove field
		collection.Fields.RemoveById("_clone_V4E8")

		// remove field
		collection.Fields.RemoveById("_clone_ZH3B")

		// remove field
		collection.Fields.RemoveById("_clone_u2PN")

		// remove field
		collection.Fields.RemoveById("_clone_ayTa")

		// remove field
		collection.Fields.RemoveById("_clone_gZOk")

		// remove field
		collection.Fields.RemoveById("json122688194")

		return app.Save(collection)
	})
}
