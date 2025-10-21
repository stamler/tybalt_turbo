package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("yovqzrnnomp0lkx")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_V1RKd7H` + "`" + ` ON ` + "`" + `jobs` + "`" + ` (` + "`" + `number` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_d1R7JSCuuJ` + "`" + ` ON ` + "`" + `jobs` + "`" + ` (` + "`" + `proposal` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_SCUCvwyln3` + "`" + ` ON ` + "`" + `jobs` + "`" + ` (` + "`" + `parent` + "`" + `)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("yovqzrnnomp0lkx")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_V1RKd7H` + "`" + ` ON ` + "`" + `jobs` + "`" + ` (` + "`" + `number` + "`" + `)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
