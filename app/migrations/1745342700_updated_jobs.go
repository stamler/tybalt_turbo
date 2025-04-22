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

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(13, []byte(`{
			"cascadeDelete": false,
			"collectionId": "3esdddggow6dykr",
			"hidden": false,
			"id": "relation1542800728",
			"maxSelect": 999,
			"minSelect": 0,
			"name": "divisions",
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

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(13, []byte(`{
			"cascadeDelete": false,
			"collectionId": "3esdddggow6dykr",
			"hidden": false,
			"id": "relation1542800728",
			"maxSelect": 999,
			"minSelect": 0,
			"name": "field",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
