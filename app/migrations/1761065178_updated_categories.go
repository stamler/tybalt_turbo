package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("nrwhbwowokwu6cr")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_SF6A76x` + "`" + ` ON ` + "`" + `categories` + "`" + ` (\n  ` + "`" + `job` + "`" + `,\n  ` + "`" + `name` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_XMSeZCfYU0` + "`" + ` ON ` + "`" + `categories` + "`" + ` (` + "`" + `job` + "`" + `)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("nrwhbwowokwu6cr")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_SF6A76x` + "`" + ` ON ` + "`" + `categories` + "`" + ` (\n  ` + "`" + `job` + "`" + `,\n  ` + "`" + `name` + "`" + `\n)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
