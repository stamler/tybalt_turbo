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
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_JXVx",
					"max": 0,
					"min": 0,
					"name": "week_ending",
					"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
					"presentable": true,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				}
			],
			"id": "pbc_1013075334",
			"indexes": [],
			"listRule": "@request.auth.id != \"\"",
			"name": "payroll_report_week_endings",
			"system": false,
			"type": "view",
			"updateRule": null,
			"viewQuery": "SELECT\n  MIN(id) as id,\n  week_ending\nFROM time_sheets\nWHERE committed != ''\n  AND (CAST(JULIANDAY(week_ending) - JULIANDAY('2025-03-01') AS INTEGER)) % 14 = 0\nGROUP BY week_ending\nORDER BY week_ending DESC;",
			"viewRule": "@request.auth.id != \"\""
		}`

		collection := &core.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1013075334")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
