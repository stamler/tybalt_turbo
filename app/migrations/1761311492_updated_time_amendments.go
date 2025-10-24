package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("5z24r2v5jgh8qft")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_Yq0DnuvO5A` + "`" + ` ON ` + "`" + `time_amendments` + "`" + ` (\n  ` + "`" + `uid` + "`" + `,\n  ` + "`" + `committed_week_ending` + "`" + `\n) WHERE committed != ''",
				"CREATE INDEX ` + "`" + `idx_c43L5uNmeG` + "`" + ` ON ` + "`" + `time_amendments` + "`" + ` (` + "`" + `committed` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_C3meukVA1I` + "`" + ` ON ` + "`" + `time_amendments` + "`" + ` (` + "`" + `date` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_5wbwGHsXcD` + "`" + ` ON ` + "`" + `time_amendments` + "`" + ` (` + "`" + `committed_week_ending` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_qypvG1ia1o` + "`" + ` ON ` + "`" + `time_amendments` + "`" + ` (\n  ` + "`" + `committed_week_ending` + "`" + `,\n  ` + "`" + `date` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_lYZL0KvBo9` + "`" + ` ON ` + "`" + `time_amendments` + "`" + ` (` + "`" + `uid` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_q5RlAM51lh` + "`" + ` ON ` + "`" + `time_amendments` + "`" + ` (` + "`" + `creator` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_eHrmGBgoXR` + "`" + ` ON ` + "`" + `time_amendments` + "`" + ` (` + "`" + `committer` + "`" + `)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("5z24r2v5jgh8qft")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_Yq0DnuvO5A` + "`" + ` ON ` + "`" + `time_amendments` + "`" + ` (\n  ` + "`" + `uid` + "`" + `,\n  ` + "`" + `committed_week_ending` + "`" + `\n) WHERE committed != ''",
				"CREATE INDEX ` + "`" + `idx_c43L5uNmeG` + "`" + ` ON ` + "`" + `time_amendments` + "`" + ` (` + "`" + `committed` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_C3meukVA1I` + "`" + ` ON ` + "`" + `time_amendments` + "`" + ` (` + "`" + `date` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_5wbwGHsXcD` + "`" + ` ON ` + "`" + `time_amendments` + "`" + ` (` + "`" + `committed_week_ending` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_qypvG1ia1o` + "`" + ` ON ` + "`" + `time_amendments` + "`" + ` (\n  ` + "`" + `committed_week_ending` + "`" + `,\n  ` + "`" + `date` + "`" + `\n)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
