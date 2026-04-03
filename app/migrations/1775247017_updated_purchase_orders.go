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

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(38, []byte(`{
			"cascadeDelete": false,
			"collectionId": "pbc_3379852803",
			"hidden": false,
			"id": "relation1767278655",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "currency",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(39, []byte(`{
			"hidden": false,
			"id": "number1851301164",
			"max": null,
			"min": null,
			"name": "approval_total_home",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("relation1767278655")

		// remove field
		collection.Fields.RemoveById("number1851301164")

		return app.Save(collection)
	})
}
