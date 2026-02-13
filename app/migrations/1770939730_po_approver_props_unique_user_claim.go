package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		if _, err := app.DB().NewQuery(`
			DELETE FROM po_approver_props
			WHERE id IN (
				SELECT id
				FROM (
					SELECT
						id,
						ROW_NUMBER() OVER (
							PARTITION BY user_claim
							ORDER BY datetime(updated) DESC, datetime(created) DESC, id DESC
						) AS rn
					FROM po_approver_props
					WHERE user_claim IS NOT NULL AND user_claim != ''
				) ranked
				WHERE rn > 1
			)
		`).Execute(); err != nil {
			return err
		}

		if _, err := app.DB().NewQuery(`
			CREATE UNIQUE INDEX IF NOT EXISTS idx_po_approver_props_user_claim
			ON po_approver_props (user_claim)
		`).Execute(); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		if _, err := app.DB().NewQuery(`
			DROP INDEX IF EXISTS idx_po_approver_props_user_claim
		`).Execute(); err != nil {
			return err
		}

		return nil
	})
}

