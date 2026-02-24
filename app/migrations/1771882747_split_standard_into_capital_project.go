package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// 1. Resolve the current "standard" / "capital" record.
		type kindRow struct {
			ID                      string  `db:"id"`
			SecondApprovalThreshold float64 `db:"second_approval_threshold"`
		}

		var capitalKindID string
		rows := []kindRow{}
		if err := app.DB().NewQuery(`
			SELECT id, COALESCE(second_approval_threshold, 0) AS second_approval_threshold
			FROM expenditure_kinds
			WHERE name = 'standard'
			LIMIT 1
		`).All(&rows); err != nil {
			return fmt.Errorf("error querying standard kind: %w", err)
		}
		if len(rows) > 0 {
			capitalKindID = rows[0].ID
		} else {
			// Already migrated: look for capital.
			capitalRows := []kindRow{}
			if err := app.DB().NewQuery(`
				SELECT id, COALESCE(second_approval_threshold, 0) AS second_approval_threshold
				FROM expenditure_kinds
				WHERE name = 'capital'
				LIMIT 1
			`).All(&capitalRows); err != nil {
				return fmt.Errorf("error querying capital kind: %w", err)
			}
			if len(capitalRows) == 0 {
				return fmt.Errorf("neither 'standard' nor 'capital' expenditure kind found")
			}
			capitalKindID = capitalRows[0].ID
			rows = capitalRows
		}

		// 2. Rename standard → capital.
		threshold := rows[0].SecondApprovalThreshold
		if threshold <= 0 {
			threshold = 500
		}
		if _, err := app.DB().NewQuery(`
			UPDATE expenditure_kinds
			SET name = 'capital',
			    en_ui_label = 'Capital',
			    description = 'expenses not linked to a job',
			    allow_job = 0,
			    second_approval_threshold = {:threshold}
			WHERE id = {:id}
		`).Bind(map[string]any{
			"id":        capitalKindID,
			"threshold": threshold,
		}).Execute(); err != nil {
			return fmt.Errorf("error renaming standard to capital: %w", err)
		}

		// 3. Insert project kind if missing.
		existingProject := []kindRow{}
		if err := app.DB().NewQuery(`
			SELECT id, COALESCE(second_approval_threshold, 0) AS second_approval_threshold
			FROM expenditure_kinds
			WHERE name = 'project'
			LIMIT 1
		`).All(&existingProject); err != nil {
			return fmt.Errorf("error checking for existing project kind: %w", err)
		}

		var projectKindID string
		if len(existingProject) > 0 {
			projectKindID = existingProject[0].ID
		} else {
			// Use a deterministic 15-char alphanumeric ID matching PocketBase conventions.
			const newProjectKindID = "prj0kind0000001"
			if _, err := app.DB().NewQuery(`
				INSERT INTO expenditure_kinds (id, name, en_ui_label, description, allow_job, second_approval_threshold)
				VALUES ({:id}, 'project', 'Project', 'expenses linked to a job', 1, 5000)
			`).Bind(map[string]any{
				"id": newProjectKindID,
			}).Execute(); err != nil {
				return fmt.Errorf("error inserting project kind: %w", err)
			}
			projectKindID = newProjectKindID
		}

		// 4. Backfill purchase_orders.kind.
		// Reclassify old "standard" POs with a job to project.
		if _, err := app.DB().NewQuery(`
			UPDATE purchase_orders
			SET kind = {:projectKindID}
			WHERE kind = {:capitalKindID}
			  AND TRIM(job) != ''
		`).Bind(map[string]any{
			"capitalKindID": capitalKindID,
			"projectKindID": projectKindID,
		}).Execute(); err != nil {
			return fmt.Errorf("error reclassifying POs with job to project: %w", err)
		}

		// Handle legacy blank/NULL PO kinds.
		if _, err := app.DB().NewQuery(`
			UPDATE purchase_orders
			SET kind = CASE
				WHEN TRIM(job) != '' THEN {:projectKindID}
				ELSE {:capitalKindID}
			END
			WHERE TRIM(COALESCE(kind, '')) = ''
		`).Bind(map[string]any{
			"capitalKindID": capitalKindID,
			"projectKindID": projectKindID,
		}).Execute(); err != nil {
			return fmt.Errorf("error backfilling blank PO kinds: %w", err)
		}

		// 5. Backfill expenses.kind.
		// PO-linked expenses inherit from their PO.
		if _, err := app.DB().NewQuery(`
			UPDATE expenses
			SET kind = (
				SELECT po.kind
				FROM purchase_orders po
				WHERE po.id = expenses.purchase_order
			)
			WHERE TRIM(COALESCE(purchase_order, '')) != ''
			  AND EXISTS (
				SELECT 1 FROM purchase_orders po WHERE po.id = expenses.purchase_order
			  )
		`).Execute(); err != nil {
			return fmt.Errorf("error backfilling PO-linked expense kinds: %w", err)
		}

		// Non-PO expenses: default by job presence.
		if _, err := app.DB().NewQuery(`
			UPDATE expenses
			SET kind = CASE
				WHEN TRIM(job) != '' THEN {:projectKindID}
				ELSE {:capitalKindID}
			END
			WHERE TRIM(COALESCE(purchase_order, '')) = ''
		`).Bind(map[string]any{
			"capitalKindID": capitalKindID,
			"projectKindID": projectKindID,
		}).Execute(); err != nil {
			return fmt.Errorf("error backfilling non-PO expense kinds: %w", err)
		}

		// Catch any remaining blank/NULL expense kinds.
		if _, err := app.DB().NewQuery(`
			UPDATE expenses
			SET kind = CASE
				WHEN TRIM(job) != '' THEN {:projectKindID}
				ELSE {:capitalKindID}
			END
			WHERE TRIM(COALESCE(kind, '')) = ''
		`).Bind(map[string]any{
			"capitalKindID": capitalKindID,
			"projectKindID": projectKindID,
		}).Execute(); err != nil {
			return fmt.Errorf("error backfilling remaining blank expense kinds: %w", err)
		}

		return nil
	}, func(app core.App) error {
		// Down migration: reverse the split.
		type kindRow struct {
			ID string `db:"id"`
		}

		// Find capital and project IDs.
		capitalRows := []kindRow{}
		if err := app.DB().NewQuery(`
			SELECT id FROM expenditure_kinds WHERE name = 'capital' LIMIT 1
		`).All(&capitalRows); err != nil || len(capitalRows) == 0 {
			return fmt.Errorf("capital kind not found for down migration")
		}
		capitalKindID := capitalRows[0].ID

		projectRows := []kindRow{}
		if err := app.DB().NewQuery(`
			SELECT id FROM expenditure_kinds WHERE name = 'project' LIMIT 1
		`).All(&projectRows); err != nil || len(projectRows) == 0 {
			return fmt.Errorf("project kind not found for down migration")
		}
		projectKindID := projectRows[0].ID

		// Move project POs/expenses back to capital (will become standard).
		if _, err := app.DB().NewQuery(`
			UPDATE purchase_orders SET kind = {:capitalKindID} WHERE kind = {:projectKindID}
		`).Bind(map[string]any{
			"capitalKindID": capitalKindID,
			"projectKindID": projectKindID,
		}).Execute(); err != nil {
			return fmt.Errorf("error reverting PO kinds: %w", err)
		}
		if _, err := app.DB().NewQuery(`
			UPDATE expenses SET kind = {:capitalKindID} WHERE kind = {:projectKindID}
		`).Bind(map[string]any{
			"capitalKindID": capitalKindID,
			"projectKindID": projectKindID,
		}).Execute(); err != nil {
			return fmt.Errorf("error reverting expense kinds: %w", err)
		}

		// Rename capital back to standard.
		if _, err := app.DB().NewQuery(`
			UPDATE expenditure_kinds
			SET name = 'standard', allow_job = 1, en_ui_label = 'Capital/Project'
			WHERE id = {:capitalKindID}
		`).Bind(map[string]any{
			"capitalKindID": capitalKindID,
		}).Execute(); err != nil {
			return fmt.Errorf("error renaming capital to standard: %w", err)
		}

		// Delete project kind.
		if _, err := app.DB().NewQuery(`
			DELETE FROM expenditure_kinds WHERE id = {:projectKindID}
		`).Bind(map[string]any{
			"projectKindID": projectKindID,
		}).Execute(); err != nil {
			return fmt.Errorf("error deleting project kind: %w", err)
		}

		return nil
	})
}
