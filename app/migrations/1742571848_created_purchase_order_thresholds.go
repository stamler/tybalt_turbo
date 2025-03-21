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
					"hidden": false,
					"id": "_clone_CHHG",
					"max": null,
					"min": null,
					"name": "approval_total",
					"onlyInt": false,
					"presentable": false,
					"required": true,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "json2651176244",
					"maxSize": 1,
					"name": "lower_threshold",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json3779115990",
					"maxSize": 1,
					"name": "upper_threshold",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				}
			],
			"id": "pbc_1245168108",
			"indexes": [],
			"listRule": null,
			"name": "purchase_order_thresholds",
			"system": false,
			"type": "view",
			"updateRule": null,
			"viewQuery": "SELECT \n    po.id, po.approval_total,\n    COALESCE((SELECT MAX(threshold) \n     FROM po_approval_thresholds \n     WHERE threshold < po.approval_total), 0) AS lower_threshold,\n    COALESCE((SELECT MIN(threshold) \n     FROM po_approval_thresholds \n     WHERE threshold >= po.approval_total),1000000) AS upper_threshold\nFROM purchase_orders AS po;",
			"viewRule": null
		}`

		collection := &core.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1245168108")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
