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
				"CREATE INDEX ` + "`" + `idx_slBmqtw6SZ` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `date` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_3TRP1AbuJv` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (\n  ` + "`" + `branch` + "`" + `,\n  ` + "`" + `job` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_rpwIgf4976` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (\n  ` + "`" + `job` + "`" + `,\n  ` + "`" + `date DESC` + "`" + `\n) WHERE committed != ''",
				"CREATE INDEX ` + "`" + `idx_VDN9yGQZiE` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `pay_period_ending` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_wYkhyQttHZ` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `committed_week_ending` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_pSzcfIjcSK` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (\n  ` + "`" + `uid` + "`" + `,\n  ` + "`" + `date` + "`" + `\n) WHERE payment_type = 'Mileage' AND committed != ''",
				"CREATE INDEX ` + "`" + `idx_oYwhXgKmaw` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `purchase_order` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_aA6a2CAKTf` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (\n  ` + "`" + `pay_period_ending` + "`" + `,\n  ` + "`" + `submitted` + "`" + `\n)"
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
				"CREATE INDEX ` + "`" + `idx_8LRpecUoxd` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (\n  ` + "`" + `purchase_order` + "`" + `,\n  ` + "`" + `committed` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_slBmqtw6SZ` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `date` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_3TRP1AbuJv` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (\n  ` + "`" + `branch` + "`" + `,\n  ` + "`" + `job` + "`" + `\n)",
				"CREATE INDEX ` + "`" + `idx_rpwIgf4976` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (\n  ` + "`" + `job` + "`" + `,\n  ` + "`" + `date DESC` + "`" + `\n) WHERE committed != ''",
				"CREATE INDEX ` + "`" + `idx_VDN9yGQZiE` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `pay_period_ending` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_wYkhyQttHZ` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `committed_week_ending` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_pSzcfIjcSK` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (\n  ` + "`" + `uid` + "`" + `,\n  ` + "`" + `date` + "`" + `\n) WHERE payment_type = 'Mileage' AND committed != ''",
				"CREATE INDEX ` + "`" + `idx_oYwhXgKmaw` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `purchase_order` + "`" + `)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
