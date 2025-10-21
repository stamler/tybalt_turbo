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
				"CREATE INDEX ` + "`" + `idx_lVCg50dCG9` + "`" + ` ON ` + "`" + `purchase_orders` + "`" + ` (\n  ` + "`" + `job` + "`" + `,\n  ` + "`" + `date DESC` + "`" + `\n) WHERE status = 'Active'"
			]
		}`), &collection); err != nil {
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
				"CREATE UNIQUE INDEX ` + "`" + `idx_6Ao8pCT` + "`" + ` ON ` + "`" + `purchase_orders` + "`" + ` (` + "`" + `po_number` + "`" + `) WHERE ` + "`" + `po_number` + "`" + ` != ''"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
