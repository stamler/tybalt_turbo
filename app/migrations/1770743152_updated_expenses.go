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
		if err := collection.Fields.AddMarshaledJSONAt(30, []byte(`{
			"cascadeDelete": false,
			"collectionId": "pbc_675944091",
			"hidden": false,
			"id": "relation1002749145",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "kind",
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
		collection.Fields.RemoveById("relation1002749145")

		return app.Save(collection)
	})
}
