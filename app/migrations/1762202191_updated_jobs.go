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
		if err := collection.Fields.AddMarshaledJSONAt(23, []byte(`{
			"hidden": false,
			"id": "select2069160921",
			"maxSelect": 1,
			"name": "authorizing_document",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "select",
			"values": [
				"Unauthorized",
				"PO",
				"PA"
			]
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(24, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text356063665",
			"max": 64,
			"min": 0,
			"name": "client_po",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(25, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text2939693995",
			"max": 0,
			"min": 0,
			"name": "client_reference_number",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
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
		collection.Fields.RemoveById("select2069160921")

		// remove field
		collection.Fields.RemoveById("text356063665")

		// remove field
		collection.Fields.RemoveById("text2939693995")

		return app.Save(collection)
	})
}
