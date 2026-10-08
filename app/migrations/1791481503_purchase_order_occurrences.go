package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Store the permitted payment count of recurring purchase orders. Existing rows
// keep the count from the former 7/14/30-day formula, which is the count their
// approval_total and committed expense limit were based on.
func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("purchase_orders")
		if err != nil {
			return err
		}
		collection.Fields.Add(&core.NumberField{
			Id:      "number1791481503",
			Name:    "occurrences",
			OnlyInt: true,
		})
		if err := app.Save(collection); err != nil {
			return err
		}
		_, err = app.DB().NewQuery(`
			UPDATE purchase_orders
			SET occurrences = CASE frequency
				WHEN 'Weekly' THEN CAST((julianday(end_date) - julianday(date)) / 7 AS INTEGER)
				WHEN 'Biweekly' THEN CAST((julianday(end_date) - julianday(date)) / 14 AS INTEGER)
				WHEN 'Monthly' THEN CAST((julianday(end_date) - julianday(date)) / 30 AS INTEGER)
				ELSE 0
			END
			WHERE type = 'Recurring' AND end_date != '' AND frequency != ''
		`).Execute()
		return err
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("purchase_orders")
		if err != nil {
			return err
		}
		collection.Fields.RemoveByName("occurrences")
		return app.Save(collection)
	})
}
