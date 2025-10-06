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
			"viewQuery": "SELECT ap.id,\n  ap.uid,\n  ap.work_week_hours,\n  ap.salary,\n  ap.default_charge_out_rate,\n  ap.off_rotation_permitted,\n  ap.skip_min_time_check,\n  ap.opening_date,\n  ap.opening_op,\n  ap.opening_ov,\n  ap.payroll_id,\n  ap.untracked_time_off,\n  ap.time_sheet_expected,\n  ap.allow_personal_reimbursement,\n  ap.mobile_phone,\n  ap.job_title,\n  ap.personal_vehicle_insurance_expiry,\n  ap.default_branch,\n  p.given_name,\n  p.surname,\n  po.po_approver_max_amount,\n  COALESCE(po.po_approver_divisions, '[]') AS po_approver_divisions\nFROM admin_profiles ap\nLEFT JOIN users u ON u.id = ap.uid\nLEFT JOIN profiles p ON u.id = p.uid\nLEFT JOIN (\n  SELECT\n    uc.uid,\n    pap.max_amount AS po_approver_max_amount,\n    pap.divisions AS po_approver_divisions\n  FROM user_claims uc\n  INNER JOIN claims c ON c.id = uc.cid AND c.name = 'po_approver'\n  LEFT JOIN po_approver_props pap ON pap.user_claim = uc.id\n) po ON po.uid = ap.uid;"
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_edi7")

		// remove field
		collection.Fields.RemoveById("_clone_rA64")

		// remove field
		collection.Fields.RemoveById("_clone_oOiG")

		// remove field
		collection.Fields.RemoveById("_clone_6MKC")

		// remove field
		collection.Fields.RemoveById("_clone_FGr4")

		// remove field
		collection.Fields.RemoveById("_clone_f95k")

		// remove field
		collection.Fields.RemoveById("_clone_IvC3")

		// remove field
		collection.Fields.RemoveById("_clone_ucNv")

		// remove field
		collection.Fields.RemoveById("_clone_1iOt")

		// remove field
		collection.Fields.RemoveById("_clone_Lbx0")

		// remove field
		collection.Fields.RemoveById("_clone_9SDc")

		// remove field
		collection.Fields.RemoveById("_clone_atft")

		// remove field
		collection.Fields.RemoveById("_clone_QI5s")

		// remove field
		collection.Fields.RemoveById("_clone_uTSf")

		// remove field
		collection.Fields.RemoveById("_clone_GA6a")

		// remove field
		collection.Fields.RemoveById("_clone_OlUx")

		// remove field
		collection.Fields.RemoveById("_clone_Gg24")

		// remove field
		collection.Fields.RemoveById("_clone_Dz05")

		// remove field
		collection.Fields.RemoveById("_clone_5pJ8")

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

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(20, []byte(`{
			"hidden": false,
			"id": "json1113389409",
			"maxSize": 1,
			"name": "po_approver_max_amount",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "json"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(21, []byte(`{
			"hidden": false,
			"id": "json1126415246",
			"maxSize": 1,
			"name": "po_approver_divisions",
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
			"viewQuery": "SELECT ap.id,\n ap.uid,\n ap.work_week_hours,\n ap.salary,\n ap.default_charge_out_rate,\n ap.off_rotation_permitted,\n ap.skip_min_time_check,\n ap.opening_date,\n ap.opening_op,\n ap.opening_ov,\n ap.payroll_id,\n ap.untracked_time_off,\n ap.time_sheet_expected,\n ap.allow_personal_reimbursement,\n ap.mobile_phone,\n ap.job_title,\n ap.personal_vehicle_insurance_expiry,\n ap.default_branch,\n p.given_name, \n p.surname\nFROM admin_profiles ap\nLEFT JOIN users u ON u.id = ap.uid\nLEFT JOIN profiles p ON u.id = p.uid;"
		}`), &collection); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_edi7",
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
			"id": "_clone_rA64",
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
			"id": "_clone_oOiG",
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
			"id": "_clone_6MKC",
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
			"id": "_clone_FGr4",
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
			"id": "_clone_f95k",
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
			"id": "_clone_IvC3",
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
			"id": "_clone_ucNv",
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
			"id": "_clone_1iOt",
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
			"id": "_clone_Lbx0",
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
			"id": "_clone_9SDc",
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
			"id": "_clone_atft",
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
			"id": "_clone_QI5s",
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
			"id": "_clone_uTSf",
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
			"id": "_clone_GA6a",
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
			"id": "_clone_OlUx",
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
			"id": "_clone_Gg24",
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
			"id": "_clone_Dz05",
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
			"id": "_clone_5pJ8",
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

		// remove field
		collection.Fields.RemoveById("json1113389409")

		// remove field
		collection.Fields.RemoveById("json1126415246")

		return app.Save(collection)
	})
}
