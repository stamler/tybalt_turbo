package migrations

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

type purchaseOrderUnrecordedClosureRow struct {
	ID       string `db:"id"`
	PONumber string `db:"po_number"`
}

// This migration records the automatic closures that Turbo performed before
// commit-time auto-close set closed and closed_by_system.
//
// Historical context:
//   - Committing an expense against a One-Time, Recurring or Cumulative PO
//     can close the PO automatically. Until this change, that code set only
//     status = 'Closed', leaving closed, closer and closed_by_system blank.
//   - A manual close always sets closer and closed, so a locally-owned PO
//     that is Closed with neither field set was closed by a commit.
//
// Scope:
//   - Only purchase orders with _imported = 0 are touched. Imported legacy POs
//     already carry closed_by_system = 1 from the import tool.
//
// Backfill rules:
//   - closed_by_system is set to 1.
//   - closed is set to the latest committed timestamp among the PO's
//     expenses. Auto-close happens in the same transaction as the commit that
//     triggered it, so that commit is the closure time.
//   - closer stays blank.
//   - POs with no committed expenses have no closure time to use. They are
//     left unchanged and logged so they can be reviewed by hand.
//
// Rollback note:
//   - The down migration is intentionally a no-op. Once backfilled, these rows
//     match POs closed by the current commit code and cannot be told apart.
func init() {
	m.Register(func(app core.App) error {
		return app.RunInTransaction(func(txApp core.App) error {
			if _, err := txApp.DB().NewQuery(`
				UPDATE purchase_orders
				SET closed_by_system = 1,
				    closed = (
				      SELECT MAX(e.committed)
				      FROM expenses e
				      WHERE e.purchase_order = purchase_orders.id
				        AND COALESCE(e.committed, '') != ''
				    )
				WHERE status = 'Closed'
				  AND COALESCE(closed, '') = ''
				  AND COALESCE(closer, '') = ''
				  AND _imported = 0
				  AND EXISTS (
				    SELECT 1
				    FROM expenses e
				    WHERE e.purchase_order = purchase_orders.id
				      AND COALESCE(e.committed, '') != ''
				  )
			`).Execute(); err != nil {
				return err
			}

			var skipped []purchaseOrderUnrecordedClosureRow
			if err := txApp.DB().NewQuery(`
				SELECT id, po_number
				FROM purchase_orders
				WHERE status = 'Closed'
				  AND COALESCE(closed, '') = ''
				  AND COALESCE(closer, '') = ''
				  AND _imported = 0
				ORDER BY po_number
			`).All(&skipped); err != nil {
				return err
			}
			if len(skipped) > 0 {
				labels := make([]string, 0, len(skipped))
				for _, row := range skipped {
					labels = append(labels, row.PONumber+" ("+row.ID+")")
				}
				txApp.Logger().Warn(
					"closed purchase orders without closer, closed timestamp or committed expenses were not backfilled",
					"count", len(skipped),
					"purchase_orders", strings.Join(labels, ", "),
				)
			}

			return nil
		})
	}, func(app core.App) error {
		return nil
	})
}
