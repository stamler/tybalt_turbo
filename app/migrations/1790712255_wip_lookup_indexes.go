package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		expenses, err := app.FindCollectionByNameOrId("expenses")
		if err != nil {
			return err
		}
		// WIP selects expenses by job or PO. The PO/date index already exists.
		expenses.AddIndex("idx_expenses_job_date", false, "`job`, `date`", "")
		if err := app.Save(expenses); err != nil {
			return err
		}
		entries, err := app.FindCollectionByNameOrId("time_entries")
		if err != nil {
			return err
		}
		// Division WIP tests for positive work hours in each allocated job.
		entries.AddIndex("idx_time_entries_job_division_date_work", false, "`job`, `division`, `date`", "`hours` > 0")
		return app.Save(entries)
	}, func(app core.App) error {
		expenses, err := app.FindCollectionByNameOrId("expenses")
		if err != nil {
			return err
		}
		expenses.RemoveIndex("idx_expenses_job_date")
		if err := app.Save(expenses); err != nil {
			return err
		}
		entries, err := app.FindCollectionByNameOrId("time_entries")
		if err != nil {
			return err
		}
		entries.RemoveIndex("idx_time_entries_job_division_date_work")
		return app.Save(entries)
	})
}
