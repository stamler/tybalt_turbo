package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("zc850lb2wclrr87")
		if err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(21, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text2354878778",
			"max": 0,
			"min": 0,
			"name": "legacy_uid",
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
		collection, err := app.FindCollectionByNameOrId("zc850lb2wclrr87")
		if err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("text2354878778")

		return app.Save(collection)
	})
}
