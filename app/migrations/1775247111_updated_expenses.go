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
		if err := collection.Fields.AddMarshaledJSONAt(31, []byte(`{
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
		if err := collection.Fields.AddMarshaledJSONAt(32, []byte(`{
			"hidden": false,
			"id": "number2912047547",
			"max": null,
			"min": null,
			"name": "settled_total",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(33, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "relation395724627",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "settler",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(34, []byte(`{
			"hidden": false,
			"id": "date3812815362",
			"max": "",
			"min": "",
			"name": "settled",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "date"
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
		collection.Fields.RemoveById("relation1767278655")

		// remove field
		collection.Fields.RemoveById("number2912047547")

		// remove field
		collection.Fields.RemoveById("relation395724627")

		// remove field
		collection.Fields.RemoveById("date3812815362")

		return app.Save(collection)
	})
}
