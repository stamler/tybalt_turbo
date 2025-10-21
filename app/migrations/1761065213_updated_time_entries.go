package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("ranctx5xgih6n3a")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_jgvQezNmMn` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (` + "`" + `uid` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_ljFGUYCrIB` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (\n  ` + "`" + `branch` + "`" + `,\n  ` + "`" + `job` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_7JBLPOOySg` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (` + "`" + `tsid` + "`" + `)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("ranctx5xgih6n3a")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_jgvQezNmMn` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (` + "`" + `uid` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_ljFGUYCrIB` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (\n  ` + "`" + `branch` + "`" + `,\n  ` + "`" + `job` + "`" + `\n)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
