package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_4126047805")
		if err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(6, []byte(`{
			"hidden": false,
			"id": "select4255467882",
			"maxSelect": 1,
			"name": "job_status_changed_to",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "select",
			"values": [
				"Cancelled",
				"No Bid"
			]
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_4126047805")
		if err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("select4255467882")

		return app.Save(collection)
	})
}
