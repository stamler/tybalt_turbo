package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("o1vpz1mm7qsfoyy")
		if err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(29, []byte(`{
			"cascadeDelete": false,
			"collectionId": "pbc_2536409462",
			"hidden": false,
			"id": "relation3146128159",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "branch",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("o1vpz1mm7qsfoyy")
		if err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("relation3146128159")

		return app.Save(collection)
	})
}
