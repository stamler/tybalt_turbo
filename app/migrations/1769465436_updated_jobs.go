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
		if err := collection.Fields.AddMarshaledJSONAt(27, []byte(`{
			"cascadeDelete": false,
			"collectionId": "pbc_126575313",
			"hidden": false,
			"id": "relation394037441",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "rate_sheet",
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
		collection.Fields.RemoveById("relation394037441")

		return app.Save(collection)
	})
}
