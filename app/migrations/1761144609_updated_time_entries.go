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
				"CREATE INDEX ` + "`" + `idx_7JBLPOOySg` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (` + "`" + `tsid` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_0Htw2Bikbl` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (` + "`" + `week_ending` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_9QORseSHUO` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (\n  ` + "`" + `job` + "`" + `,\n  ` + "`" + `date DESC` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_9fjDrGa7M7` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (` + "`" + `division` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_ZJij6mtUY4` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (` + "`" + `time_type` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_EXtVt4doe6` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (` + "`" + `branch` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_oUAVmzP0Gf` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (` + "`" + `category` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_a7yXF8hMf0` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (\n  ` + "`" + `tsid` + "`" + `,\n  ` + "`" + `time_type` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_ZX8ABROs07` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (\n  ` + "`" + `uid` + "`" + `,\n  ` + "`" + `week_ending` + "`" + `\n)"
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
				"CREATE INDEX ` + "`" + `idx_ljFGUYCrIB` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (\n  ` + "`" + `branch` + "`" + `,\n  ` + "`" + `job` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_7JBLPOOySg` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (` + "`" + `tsid` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_0Htw2Bikbl` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (` + "`" + `week_ending` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_9QORseSHUO` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (\n  ` + "`" + `job` + "`" + `,\n  ` + "`" + `date DESC` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_9fjDrGa7M7` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (` + "`" + `division` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_ZJij6mtUY4` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (` + "`" + `time_type` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_EXtVt4doe6` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (` + "`" + `branch` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_oUAVmzP0Gf` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (` + "`" + `category` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_a7yXF8hMf0` + "`" + ` ON ` + "`" + `time_entries` + "`" + ` (\n  ` + "`" + `tsid` + "`" + `,\n  ` + "`" + `time_type` + "`" + `\n)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
