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
			"listRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'admin'",
			"viewRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'admin'"
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_QuOo")

		// remove field
		collection.Fields.RemoveById("_clone_AqYr")

		// remove field
		collection.Fields.RemoveById("_clone_jFGG")

		// remove field
		collection.Fields.RemoveById("_clone_i51a")

		// remove field
		collection.Fields.RemoveById("_clone_aBiM")

		// remove field
		collection.Fields.RemoveById("_clone_5nyu")

		// remove field
		collection.Fields.RemoveById("_clone_Hggv")

		// remove field
		collection.Fields.RemoveById("_clone_c0q4")

		// remove field
		collection.Fields.RemoveById("_clone_aedC")

		// remove field
		collection.Fields.RemoveById("_clone_UvWw")

		// remove field
		collection.Fields.RemoveById("_clone_x0Tf")

		// remove field
		collection.Fields.RemoveById("_clone_zN5I")

		// remove field
		collection.Fields.RemoveById("_clone_UDzs")

		// remove field
		collection.Fields.RemoveById("_clone_wD0o")

		// remove field
		collection.Fields.RemoveById("_clone_38W6")

		// remove field
		collection.Fields.RemoveById("_clone_fXC1")

		// remove field
		collection.Fields.RemoveById("_clone_htel")

		// remove field
		collection.Fields.RemoveById("_clone_3Wcy")

		// remove field
		collection.Fields.RemoveById("_clone_iCat")

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_xP5M",
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
			"id": "_clone_qwnu",
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
			"id": "_clone_SIcz",
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
			"id": "_clone_hXUz",
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
			"id": "_clone_gbQK",
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
			"id": "_clone_a9vy",
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
			"id": "_clone_tqNj",
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
			"id": "_clone_6oGY",
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
			"id": "_clone_9xAb",
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
			"id": "_clone_e3fo",
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
			"id": "_clone_4KEk",
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
			"id": "_clone_MCKg",
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
			"id": "_clone_PQLP",
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
			"id": "_clone_JGWZ",
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
			"id": "_clone_I4LM",
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
			"id": "_clone_VWEC",
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
			"id": "_clone_zB0i",
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
			"id": "_clone_Uy1C",
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
			"id": "_clone_KACz",
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

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_697077494")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"listRule": null,
			"viewRule": null
		}`), &collection); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_QuOo",
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
			"id": "_clone_AqYr",
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
			"id": "_clone_jFGG",
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
			"id": "_clone_i51a",
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
			"id": "_clone_aBiM",
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
			"id": "_clone_5nyu",
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
			"id": "_clone_Hggv",
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
			"id": "_clone_c0q4",
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
			"id": "_clone_aedC",
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
			"id": "_clone_UvWw",
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
			"id": "_clone_x0Tf",
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
			"id": "_clone_zN5I",
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
			"id": "_clone_UDzs",
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
			"id": "_clone_wD0o",
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
			"id": "_clone_38W6",
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
			"id": "_clone_fXC1",
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
			"id": "_clone_htel",
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
			"id": "_clone_3Wcy",
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
			"id": "_clone_iCat",
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
		collection.Fields.RemoveById("_clone_xP5M")

		// remove field
		collection.Fields.RemoveById("_clone_qwnu")

		// remove field
		collection.Fields.RemoveById("_clone_SIcz")

		// remove field
		collection.Fields.RemoveById("_clone_hXUz")

		// remove field
		collection.Fields.RemoveById("_clone_gbQK")

		// remove field
		collection.Fields.RemoveById("_clone_a9vy")

		// remove field
		collection.Fields.RemoveById("_clone_tqNj")

		// remove field
		collection.Fields.RemoveById("_clone_6oGY")

		// remove field
		collection.Fields.RemoveById("_clone_9xAb")

		// remove field
		collection.Fields.RemoveById("_clone_e3fo")

		// remove field
		collection.Fields.RemoveById("_clone_4KEk")

		// remove field
		collection.Fields.RemoveById("_clone_MCKg")

		// remove field
		collection.Fields.RemoveById("_clone_PQLP")

		// remove field
		collection.Fields.RemoveById("_clone_JGWZ")

		// remove field
		collection.Fields.RemoveById("_clone_I4LM")

		// remove field
		collection.Fields.RemoveById("_clone_VWEC")

		// remove field
		collection.Fields.RemoveById("_clone_zB0i")

		// remove field
		collection.Fields.RemoveById("_clone_Uy1C")

		// remove field
		collection.Fields.RemoveById("_clone_KACz")

		return app.Save(collection)
	})
}
