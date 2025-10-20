package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("yovqzrnnomp0lkx")
		if err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(22, []byte(`{
			"cascadeDelete": false,
			"collectionId": "yovqzrnnomp0lkx",
			"hidden": false,
			"id": "relation1032740943",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "parent",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("yovqzrnnomp0lkx")
		if err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("relation1032740943")

		return app.Save(collection)
	})
}
