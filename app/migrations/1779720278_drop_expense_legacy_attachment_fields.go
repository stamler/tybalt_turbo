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

		collection.RemoveIndex("idx_KqwTULTh3p")
		collection.Fields.RemoveById("edbixzlo")
		collection.Fields.RemoveById("text701200007")

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("expenses")
		if err != nil {
			return err
		}

		if collection.Fields.GetByName("attachment") == nil {
			if err := collection.Fields.AddMarshaledJSONAt(7, []byte(`{
				"hidden": false,
				"id": "edbixzlo",
				"maxSelect": 1,
				"maxSize": 5242880,
				"mimeTypes": [
					"application/pdf",
					"image/jpeg",
					"image/png",
					"image/heic"
				],
				"name": "attachment",
				"presentable": false,
				"protected": false,
				"required": false,
				"system": false,
				"thumbs": null,
				"type": "file"
			}`)); err != nil {
				return err
			}
		}

		if collection.Fields.GetByName("attachment_hash") == nil {
			if err := collection.Fields.AddMarshaledJSONAt(28, []byte(`{
				"autogeneratePattern": "",
				"hidden": false,
				"id": "text701200007",
				"max": 0,
				"min": 0,
				"name": "attachment_hash",
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

		collection.AddIndex("idx_KqwTULTh3p", true, "`attachment_hash`", "`attachment_hash` != ''")

		return app.Save(collection)
	})
}
