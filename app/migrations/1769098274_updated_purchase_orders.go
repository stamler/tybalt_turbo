package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_6Ao8pCT` + "`" + ` ON ` + "`" + `purchase_orders` + "`" + ` (` + "`" + `po_number` + "`" + `) WHERE ` + "`" + `po_number` + "`" + ` != ''",
				"CREATE INDEX ` + "`" + `idx_lVCg50dCG9` + "`" + ` ON ` + "`" + `purchase_orders` + "`" + ` (\n  ` + "`" + `job` + "`" + `,\n  ` + "`" + `date DESC` + "`" + `\n) WHERE status = 'Active'",
				"CREATE UNIQUE INDEX ` + "`" + `idx_Ml6Pmg44QP` + "`" + ` ON ` + "`" + `purchase_orders` + "`" + ` (` + "`" + `attachment_hash` + "`" + `) WHERE ` + "`" + `attachment_hash` + "`" + ` != ''"
			]
		}`), &collection); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(35, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text701200007",
			"max": 0,
			"min": 0,
			"name": "attachment_hash",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_6Ao8pCT` + "`" + ` ON ` + "`" + `purchase_orders` + "`" + ` (` + "`" + `po_number` + "`" + `) WHERE ` + "`" + `po_number` + "`" + ` != ''",
				"CREATE INDEX ` + "`" + `idx_lVCg50dCG9` + "`" + ` ON ` + "`" + `purchase_orders` + "`" + ` (\n  ` + "`" + `job` + "`" + `,\n  ` + "`" + `date DESC` + "`" + `\n) WHERE status = 'Active'"
			]
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("text701200007")

		return app.Save(collection)
	})
}
