package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1245168108")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"name": "purchase_orders_augmented",
			"viewQuery": "SELECT \n    po.id, po.approval_total,\n    COALESCE((SELECT MAX(threshold) \n     FROM po_approval_thresholds \n     WHERE threshold < po.approval_total), 0) AS lower_threshold,\n    COALESCE((SELECT MIN(threshold) \n     FROM po_approval_thresholds \n     WHERE threshold >= po.approval_total),1000000) AS upper_threshold,\n    (SELECT COUNT(*) \n     FROM expenses \n     WHERE expenses.purchase_order = po.id) AS expenses_count\nFROM purchase_orders AS po;"
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_CHHG")

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"hidden": false,
			"id": "_clone_0b9j",
			"max": null,
			"min": null,
			"name": "approval_total",
			"onlyInt": false,
			"presentable": false,
			"required": true,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(4, []byte(`{
			"hidden": false,
			"id": "json3243452931",
			"maxSize": 1,
			"name": "expenses_count",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "json"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1245168108")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"name": "purchase_order_thresholds",
			"viewQuery": "SELECT \n    po.id, po.approval_total,\n    COALESCE((SELECT MAX(threshold) \n     FROM po_approval_thresholds \n     WHERE threshold < po.approval_total), 0) AS lower_threshold,\n    COALESCE((SELECT MIN(threshold) \n     FROM po_approval_thresholds \n     WHERE threshold >= po.approval_total),1000000) AS upper_threshold\nFROM purchase_orders AS po;"
		}`), &collection); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
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
		}`)); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_0b9j")

		// remove field
		collection.Fields.RemoveById("json3243452931")

		return app.Save(collection)
	})
}
