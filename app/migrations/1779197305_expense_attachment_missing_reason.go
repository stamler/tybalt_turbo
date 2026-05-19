package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("expenses")
		if err != nil {
			return err
		}

		if collection.Fields.GetByName("attachment_missing_reason") == nil {
			if err := collection.Fields.AddMarshaledJSON([]byte(`{
				"autogeneratePattern": "",
				"hidden": false,
				"id": "text1779197305",
				"max": 0,
				"min": 0,
				"name": "attachment_missing_reason",
				"pattern": "",
				"presentable": false,
				"primaryKey": false,
				"required": false,
				"system": false,
				"type": "text"
			}`)); err != nil {
				return err
			}
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("expenses")
		if err != nil {
			return err
		}

		collection.Fields.RemoveById("text1779197305")
		return app.Save(collection)
	})
}
