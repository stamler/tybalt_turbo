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
					"id": "text1997877400",
					"max": 0,
					"min": 0,
					"name": "code",
					"pattern": "^[A-Z]{3}$",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text3972544249",
					"max": 1,
					"min": 0,
					"name": "symbol",
					"pattern": "^(\\p{Sc}|[A-Z]{3})$",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "file1704208859",
					"maxSelect": 1,
					"maxSize": 0,
					"mimeTypes": [
						"image/svg+xml"
					],
					"name": "icon",
					"presentable": false,
					"protected": false,
					"required": true,
					"system": false,
					"thumbs": [],
					"type": "file"
				},
				{
					"hidden": false,
					"id": "number3756801849",
					"max": null,
					"min": null,
					"name": "rate",
					"onlyInt": false,
					"presentable": false,
					"required": true,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "date1436190069",
					"max": "",
					"min": "",
					"name": "rate_date",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "date"
				},
				{
					"hidden": false,
					"id": "number3336833860",
					"max": null,
					"min": null,
					"name": "ui_sort",
					"onlyInt": false,
					"presentable": false,
					"required": false,
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
			"id": "pbc_3379852803",
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_xVaAmGcXs3` + "`" + ` ON ` + "`" + `currencies` + "`" + ` (` + "`" + `code` + "`" + `)"
			],
			"listRule": null,
			"name": "currencies",
			"system": false,
			"type": "base",
			"updateRule": null,
			"viewRule": null
		}`

		collection := &core.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_3379852803")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
