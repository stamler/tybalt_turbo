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
				"CREATE INDEX ` + "`" + `idx_expenses_uid_date` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `uid` + "`" + `, ` + "`" + `date` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_expenses_approver_submitted_date` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `approver` + "`" + `, ` + "`" + `submitted` + "`" + `, ` + "`" + `date` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_expenses_po_date` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `purchase_order` + "`" + `, ` + "`" + `date` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_expenses_approved_nonempty` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `approved` + "`" + `) WHERE ` + "`" + `approved` + "`" + ` != ''",
				"CREATE INDEX ` + "`" + `idx_expenses_committed_nonempty` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `committed` + "`" + `) WHERE ` + "`" + `committed` + "`" + ` != ''",
				"CREATE INDEX ` + "`" + `idx_Y3uLpJvqvc` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `committed_week_ending` + "`" + `)"
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
				"CREATE INDEX ` + "`" + `idx_expenses_uid_date` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `uid` + "`" + `, ` + "`" + `date` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_expenses_approver_submitted_date` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `approver` + "`" + `, ` + "`" + `submitted` + "`" + `, ` + "`" + `date` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_expenses_po_date` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `purchase_order` + "`" + `, ` + "`" + `date` + "`" + `)",
				"CREATE INDEX ` + "`" + `idx_expenses_approved_nonempty` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `approved` + "`" + `) WHERE ` + "`" + `approved` + "`" + ` != ''",
				"CREATE INDEX ` + "`" + `idx_expenses_committed_nonempty` + "`" + ` ON ` + "`" + `expenses` + "`" + ` (` + "`" + `committed` + "`" + `) WHERE ` + "`" + `committed` + "`" + ` != ''"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
