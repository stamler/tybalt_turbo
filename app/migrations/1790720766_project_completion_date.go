package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("jobs")
		if err != nil {
			return err
		}
		collection.Fields.Add(&core.TextField{
			Id:      "text1790720766",
			Name:    "project_completion_date",
			Pattern: `^\d{4}-\d{2}-\d{2}$`,
		})
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("jobs")
		if err != nil {
			return err
		}
		collection.Fields.RemoveByName("project_completion_date")
		return app.Save(collection)
	})
}
