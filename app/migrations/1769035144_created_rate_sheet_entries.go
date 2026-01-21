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
					"cascadeDelete": false,
					"collectionId": "pbc_3637380980",
					"hidden": false,
					"id": "relation1466534506",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "role",
					"presentable": true,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"cascadeDelete": false,
					"collectionId": "pbc_126575313",
					"hidden": false,
					"id": "relation394037441",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "rate_sheet",
					"presentable": true,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"hidden": false,
					"id": "number3756801849",
					"max": null,
					"min": 1,
					"name": "rate",
					"onlyInt": true,
					"presentable": false,
					"required": true,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "number2867273880",
					"max": null,
					"min": 1,
					"name": "overtime_rate",
					"onlyInt": true,
					"presentable": false,
					"required": true,
					"system": false,
					"type": "number"
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
			"id": "pbc_2420693830",
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_MLXiy2bT4z` + "`" + ` ON ` + "`" + `rate_sheet_entries` + "`" + ` (\n  ` + "`" + `role` + "`" + `,\n  ` + "`" + `rate_sheet` + "`" + `\n)"
			],
			"listRule": "@request.auth.id != \"\"",
			"name": "rate_sheet_entries",
			"system": false,
			"type": "base",
			"updateRule": null,
			"viewRule": "@request.auth.id != \"\""
		}`

		collection := &core.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_2420693830")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
