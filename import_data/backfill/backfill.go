package backfill

import (
	"database/sql"
	"fmt"
)

// BackfillBranches executes SQL statements to populate branch fields on imported records
func BackfillBranches(dbPath string) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	statements := []string{
		// -- Backfill the branch column in the time_entries table to the default branch from the corresponding user's admin_profiles record
		// -- TODO: use historical branch membership data to backfill branches for time entries at earlier dates
		`UPDATE time_entries AS te
SET branch = ap.default_branch
FROM admin_profiles AS ap
WHERE ap.uid = te.uid
  AND te.branch = '';`,

		// -- Backfill the branch column in the expenses table to the default branch from the corresponding user's admin_profiles record
		// -- This only updates expenses with job = '' (and branch empty).
		// -- TODO: use historical branch membership data to backfill branches for expenses at earlier dates
		`UPDATE expenses AS e
SET branch = ap.default_branch
FROM admin_profiles AS ap
WHERE ap.uid = e.uid
  AND e.job = ''
  AND e.branch = '';`,

		// -- Backfill the branch column in the expenses table to the branch from the corresponding jobs record
		// -- This only updates expenses with job <> '' (and branch empty).
		// -- TODO: use historical branch membership data to backfill branches for expenses at earlier dates
		`UPDATE expenses AS e
SET branch = j.branch
FROM jobs AS j
WHERE e.job <> ''
  AND e.branch = ''
  AND j.id = e.job;`,

		// -- Backfill the branch column in the purchase_orders table to the default branch from the corresponding user's admin_profiles record
		// -- This only updates purchase_orders with job = '' (and branch empty).
		// -- TODO: use historical branch membership data to backfill branches for purchase_orders at earlier dates
		`UPDATE purchase_orders AS po
SET branch = ap.default_branch
FROM admin_profiles AS ap
WHERE ap.uid = po.uid
  AND po.job = ''
  AND po.branch = '';`,

		// -- Backfill the branch column in the purchase_orders table to the branch from the corresponding jobs record
		// -- This only updates purchase_orders with job <> '' (and branch empty).
		// -- TODO: use historical branch membership data to backfill branches for purchase_orders at earlier dates
		`UPDATE purchase_orders AS po
SET branch = j.branch
FROM jobs AS j
WHERE po.job <> ''
  AND po.branch = ''
  AND j.id = po.job;`,
	}

	for _, stmt := range statements {
		if _, err := tx.Exec(stmt); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("backfill branches: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit backfill: %w", err)
	}

	return nil
}
