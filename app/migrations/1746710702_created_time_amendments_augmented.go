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
					"collectionId": "3esdddggow6dykr",
					"hidden": false,
					"id": "_clone_RIZn",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "division",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "relation"
				},
				{
					"cascadeDelete": false,
					"collectionId": "_pb_users_auth_",
					"hidden": false,
					"id": "_clone_9MBG",
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
					"id": "_clone_twCp",
					"max": 18,
					"min": -18,
					"name": "hours",
					"onlyInt": false,
					"presentable": false,
					"required": true,
					"system": false,
					"type": "number"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_c5Yb",
					"max": 0,
					"min": 0,
					"name": "description",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"cascadeDelete": false,
					"collectionId": "cnqv0wm8hly7r3n",
					"hidden": false,
					"id": "_clone_XZ9E",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "time_type",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"hidden": false,
					"id": "_clone_WAu2",
					"max": 3,
					"min": 0,
					"name": "meals_hours",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"cascadeDelete": false,
					"collectionId": "yovqzrnnomp0lkx",
					"hidden": false,
					"id": "_clone_x7xd",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "job",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "relation"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_VgLu",
					"max": 0,
					"min": 0,
					"name": "work_record",
					"pattern": "^[FKQ][0-9]{2}-[0-9]{3,}(-[0-9]+)?$",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "_clone_Pr0W",
					"max": null,
					"min": null,
					"name": "payout_request_amount",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_Gsy4",
					"max": 0,
					"min": 0,
					"name": "date",
					"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
					"presentable": true,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_NkNO",
					"max": 0,
					"min": 0,
					"name": "week_ending",
					"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
					"presentable": true,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"cascadeDelete": false,
					"collectionId": "fpri53nrr2xgoov",
					"hidden": false,
					"id": "_clone_vDHD",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "tsid",
					"presentable": true,
					"required": false,
					"system": false,
					"type": "relation"
				},
				{
					"cascadeDelete": false,
					"collectionId": "nrwhbwowokwu6cr",
					"hidden": false,
					"id": "_clone_Wl6i",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "category",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "relation"
				},
				{
					"cascadeDelete": false,
					"collectionId": "_pb_users_auth_",
					"hidden": false,
					"id": "_clone_Uh7P",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "creator",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"hidden": false,
					"id": "_clone_5KFD",
					"max": "",
					"min": "",
					"name": "committed",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "date"
				},
				{
					"cascadeDelete": false,
					"collectionId": "_pb_users_auth_",
					"hidden": false,
					"id": "_clone_ItpT",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "committer",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "relation"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_bx59",
					"max": 0,
					"min": 0,
					"name": "committed_week_ending",
					"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
					"presentable": true,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "_clone_XHYY",
					"name": "skip_tsid_check",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "bool"
				},
				{
					"hidden": false,
					"id": "json3388866023",
					"maxSize": 1,
					"name": "uid_name",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json682311642",
					"maxSize": 1,
					"name": "creator_name",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json3621682694",
					"maxSize": 1,
					"name": "committer_name",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_8T6p",
					"max": 0,
					"min": 1,
					"name": "time_type_code",
					"pattern": "",
					"presentable": true,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_sv4T",
					"max": 0,
					"min": 2,
					"name": "time_type_name",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_UXsb",
					"max": 0,
					"min": 0,
					"name": "job_number",
					"pattern": "^(P)?[0-9]{2}-[0-9]{3,4}(-[0-9]{1,2})?(-[0-9])?$",
					"presentable": true,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_jQRA",
					"max": 0,
					"min": 3,
					"name": "job_description",
					"pattern": "",
					"presentable": true,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_wVe0",
					"max": 0,
					"min": 3,
					"name": "category_name",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_fI5B",
					"max": 0,
					"min": 1,
					"name": "division_code",
					"pattern": "",
					"presentable": true,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_pNlv",
					"max": 0,
					"min": 2,
					"name": "division_name",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				}
			],
			"id": "pbc_3637308620",
			"indexes": [],
			"listRule": "// copy the listRule and viewRule from the time_amendments collection api rules\n@request.auth.user_claims_via_uid.cid.name ?= 'tame' ||\n@request.auth.user_claims_via_uid.cid.name ?= 'report'",
			"name": "time_amendments_augmented",
			"system": false,
			"type": "view",
			"updateRule": null,
			"viewQuery": "SELECT \n  ta.id,\n  ta.division,\n  ta.uid,\n  ta.hours,\n  ta.description,\n  ta.time_type,\n  ta.meals_hours,\n  ta.job,\n  ta.work_record,\n  ta.payout_request_amount,\n  ta.date,\n  ta.week_ending,\n  ta.tsid,\n  ta.category,\n  ta.creator,\n  ta.committed,\n  ta.committer,\n  ta.committed_week_ending,\n  ta.skip_tsid_check,\n  (p0.given_name || ' ' || p0.surname) as uid_name,\n  (p1.given_name || ' ' || p1.surname) as creator_name,\n  (p2.given_name || ' ' || p2.surname) as committer_name,\n  tt.code as time_type_code,\n  tt.name as time_type_name,\n  j.number as job_number,\n  j.description as job_description,\n  c.name as category_name,\n  d.code as division_code,\n  d.name as division_name\nFROM time_amendments ta\nLEFT JOIN time_types tt ON ta.time_type = tt.id\nLEFT JOIN jobs j ON ta.job = j.id\nLEFT JOIN categories c ON ta.category = c.id\nLEFT JOIN divisions d ON ta.division = d.id\nLEFT JOIN profiles p0 ON ta.uid = p0.uid\nLEFT JOIN profiles p1 ON ta.creator = p1.uid\nLEFT JOIN profiles p2 ON ta.committer = p2.uid\nORDER BY ta.date DESC;",
			"viewRule": "// copy the listRule and viewRule from the time_amendments collection api rules\n@request.auth.user_claims_via_uid.cid.name ?= 'tame' ||\n@request.auth.user_claims_via_uid.cid.name ?= 'report'"
		}`

		collection := &core.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_3637308620")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
