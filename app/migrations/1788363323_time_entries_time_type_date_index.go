package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const employeeBranchHoursTimeEntryIndex = "idx_time_entries_time_type_date"

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("time_entries")
		if err != nil {
			return err
		}
		collection.AddIndex(
			employeeBranchHoursTimeEntryIndex,
			false,
			"`time_type`, `date`",
			"",
		)
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("time_entries")
		if err != nil {
			return err
		}
		collection.RemoveIndex(employeeBranchHoursTimeEntryIndex)
		return app.Save(collection)
	})
}
