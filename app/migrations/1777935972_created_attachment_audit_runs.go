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
					"autogeneratePattern": "[a-z0-9]{15}",
					"hidden": false,
					"id": "text3208210256",
					"max": 15,
					"min": 15,
					"name": "id",
					"pattern": "^[a-z0-9]+$",
					"presentable": false,
					"primaryKey": true,
					"required": true,
					"system": true,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text1777935972a",
					"max": 0,
					"min": 0,
					"name": "target_key",
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
					"id": "text1777935972b",
					"max": 0,
					"min": 0,
					"name": "label",
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
					"id": "text1777935972c",
					"max": 0,
					"min": 0,
					"name": "collection_name",
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
					"id": "text1777935972d",
					"max": 0,
					"min": 0,
					"name": "field_name",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "select1777935972",
					"maxSelect": 1,
					"name": "status",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "select",
					"values": ["running", "completed", "failed"]
				},
				{
					"cascadeDelete": false,
					"collectionId": "_pb_users_auth_",
					"hidden": false,
					"id": "relation1777935972",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "requested_by",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "relation"
				},
				{
					"hidden": false,
					"id": "date1777935972a",
					"max": "",
					"min": "",
					"name": "started_at",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "date"
				},
				{
					"hidden": false,
					"id": "date1777935972b",
					"max": "",
					"min": "",
					"name": "finished_at",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "date"
				},
				{
					"hidden": false,
					"id": "number1777935972a",
					"max": null,
					"min": 0,
					"name": "total_records",
					"onlyInt": true,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "number1777935972b",
					"max": null,
					"min": 0,
					"name": "referenced_records",
					"onlyInt": true,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "number1777935972c",
					"max": null,
					"min": 0,
					"name": "matching_records",
					"onlyInt": true,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "number1777935972d",
					"max": null,
					"min": 0,
					"name": "missing_records",
					"onlyInt": true,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "number1777935972e",
					"max": null,
					"min": 0,
					"name": "orphaned_files",
					"onlyInt": true,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text1777935972e",
					"max": 0,
					"min": 0,
					"name": "error",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "file1777935972a",
					"maxSelect": 1,
					"maxSize": 536870912,
					"mimeTypes": [],
					"name": "missing_report",
					"presentable": false,
					"protected": false,
					"required": false,
					"system": false,
					"thumbs": [],
					"type": "file"
				},
				{
					"hidden": false,
					"id": "file1777935972b",
					"maxSelect": 1,
					"maxSize": 536870912,
					"mimeTypes": [],
					"name": "orphaned_report",
					"presentable": false,
					"protected": false,
					"required": false,
					"system": false,
					"thumbs": [],
					"type": "file"
				},
				{
					"hidden": false,
					"id": "autodate2990389176",
					"name": "created",
					"onCreate": true,
					"onUpdate": false,
					"presentable": false,
					"system": false,
					"type": "autodate"
				},
				{
					"hidden": false,
					"id": "autodate3332085495",
					"name": "updated",
					"onCreate": true,
					"onUpdate": true,
					"presentable": false,
					"system": false,
					"type": "autodate"
				}
			],
			"id": "pbc_1777935972",
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_attachment_audit_runs_target_key` + "`" + ` ON ` + "`" + `attachment_audit_runs` + "`" + ` (` + "`" + `target_key` + "`" + `)"
			],
			"listRule": "@request.auth.user_claims_via_uid.cid.name ?= 'admin'",
			"name": "attachment_audit_runs",
			"system": false,
			"type": "base",
			"updateRule": null,
			"viewRule": "@request.auth.user_claims_via_uid.cid.name ?= 'admin'"
		}`

		collection := &core.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1777935972")
		if err != nil {
			return err
		}
		return app.Delete(collection)
	})
}
