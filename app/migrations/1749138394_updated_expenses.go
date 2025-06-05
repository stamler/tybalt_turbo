package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("o1vpz1mm7qsfoyy")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_KqwTULTh3p` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `attachment_hash` + "`" + `) WHERE ` + "`" + `attachment_hash` + "`" + ` != ''",
				"CREATE INDEX ` + "`" + `idx_8LRpecUoxd` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (\n  ` + "`" + `purchase_order` + "`" + `,\n  ` + "`" + `committed` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_slBmqtw6SZ` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `date` + "`" + `)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("o1vpz1mm7qsfoyy")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_KqwTULTh3p` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `attachment_hash` + "`" + `) WHERE ` + "`" + `attachment_hash` + "`" + ` != ''",
				"CREATE INDEX ` + "`" + `idx_8LRpecUoxd` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (\n  ` + "`" + `purchase_order` + "`" + `,\n  ` + "`" + `committed` + "`" + `\n)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
