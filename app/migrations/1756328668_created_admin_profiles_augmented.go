package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		jsonData := `{
			"createRule": null,
			"deleteRule": null,
			"fields": [
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text3208210256",
					"max": 0,
					"min": 0,
					"name": "id",
					"pattern": "^[a-z0-9]+$",
					"presentable": false,
					"primaryKey": true,
					"required": true,
					"system": true,
					"type": "text"
				},
				{
					"cascadeDelete": false,
					"collectionId": "_pb_users_auth_",
					"hidden": false,
					"id": "_clone_SwLz",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "uid",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"hidden": false,
					"id": "_clone_ig1h",
					"max": 40,
					"min": 8,
					"name": "work_week_hours",
					"onlyInt": false,
					"presentable": false,
					"required": true,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "_clone_IOis",
					"name": "salary",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "bool"
				},
				{
					"hidden": false,
					"id": "_clone_BNOL",
					"max": 1000,
					"min": 50,
					"name": "default_charge_out_rate",
					"onlyInt": false,
					"presentable": false,
					"required": true,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "_clone_M8I7",
					"name": "off_rotation_permitted",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "bool"
				},
				{
					"hidden": false,
					"id": "_clone_AjpM",
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
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_ggLX",
					"max": 0,
					"min": 0,
					"name": "opening_date",
					"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "_clone_bI9K",
					"max": 332,
					"min": 0,
					"name": "opening_op",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "_clone_YvMC",
					"max": 200,
					"min": 0,
					"name": "opening_ov",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_eL7h",
					"max": 0,
					"min": 0,
					"name": "payroll_id",
					"pattern": "^(?:[1-9]\\d*|CMS[0-9]{1,2})$",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "_clone_1vRF",
					"name": "untracked_time_off",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "bool"
				},
				{
					"hidden": false,
					"id": "_clone_eY4S",
					"name": "time_sheet_expected",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "bool"
				},
				{
					"hidden": false,
					"id": "_clone_0Gb3",
					"name": "allow_personal_reimbursement",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "bool"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_nvIh",
					"max": 0,
					"min": 0,
					"name": "mobile_phone",
					"pattern": "^\\+1 \\(\\d{3}\\) \\d{3}-\\d{4}$",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_nQXj",
					"max": 0,
					"min": 0,
					"name": "job_title",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_ZG7L",
					"max": 0,
					"min": 0,
					"name": "personal_vehicle_insurance_expiry",
					"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"cascadeDelete": false,
					"collectionId": "pbc_2536409462",
					"hidden": false,
					"id": "_clone_TItn",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "default_branch",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_2zuU",
					"max": 48,
					"min": 2,
					"name": "given_name",
					"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_dkfy",
					"max": 48,
					"min": 2,
					"name": "surname",
					"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				}
			],
			"id": "pbc_697077494",
			"indexes": [],
			"listRule": null,
			"name": "admin_profiles_augmented",
			"system": false,
			"type": "view",
			"updateRule": null,
			"viewQuery": "SELECT ap.id,\n ap.uid,\n ap.work_week_hours,\n ap.salary,\n ap.default_charge_out_rate,\n ap.off_rotation_permitted,\n ap.skip_min_time_check,\n ap.opening_date,\n ap.opening_op,\n ap.opening_ov,\n ap.payroll_id,\n ap.untracked_time_off,\n ap.time_sheet_expected,\n ap.allow_personal_reimbursement,\n ap.mobile_phone,\n ap.job_title,\n ap.personal_vehicle_insurance_expiry,\n ap.default_branch,\n p.given_name, \n p.surname\nFROM admin_profiles ap\nLEFT JOIN users u ON u.id = ap.uid\nLEFT JOIN profiles p ON u.id = p.uid;",
			"viewRule": null
		}`

		collection := &core.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_697077494")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
