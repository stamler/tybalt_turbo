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
					"id": "_clone_m0OD",
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
					"hidden": false,
					"id": "json2170425494",
					"maxSize": 1,
					"name": "committed_count",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json35465996",
					"maxSize": 1,
					"name": "approved_count",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json1456061467",
					"maxSize": 1,
					"name": "submitted_count",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				}
			],
			"id": "pbc_331583008",
			"indexes": [],
			"listRule": null,
			"name": "time_tracking",
			"system": false,
			"type": "view",
			"updateRule": null,
			"viewQuery": "SELECT\n  MIN(id) id,\n  week_ending,\n  SUM(CASE WHEN committed != '' THEN 1 ELSE 0 END) committed_count,\n  SUM(CASE WHEN approved != '' AND committed = '' AND rejected = '' THEN 1 ELSE 0 END) approved_count,\n  SUM(CASE WHEN submitted = 1 AND approved = '' AND committed = '' AND rejected = '' THEN 1 ELSE 0 END) submitted_count\nFROM time_sheets\nGROUP BY week_ending\nORDER BY week_ending DESC; ",
			"viewRule": null
		}`

		collection := &core.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_331583008")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
