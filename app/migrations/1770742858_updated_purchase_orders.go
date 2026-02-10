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

		// remove field
		collection.Fields.RemoveById("select1002749145")

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(36, []byte(`{
			"hidden": false,
			"id": "select1002749145",
			"maxSelect": 1,
			"name": "kind",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "select",
			"values": [
				"standard",
				"sponsorship",
				"staff_and_social",
				"media_and_event",
				"computer"
			]
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
