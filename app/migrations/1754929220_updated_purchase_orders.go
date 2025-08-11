package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(24, []byte(`{
			"cascadeDelete": false,
			"collectionId": "y0xvnesailac971",
			"hidden": false,
			"id": "kbqsgaiq",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "vendor",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(24, []byte(`{
			"cascadeDelete": false,
			"collectionId": "y0xvnesailac971",
			"hidden": false,
			"id": "kbqsgaiq",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "vendor",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
