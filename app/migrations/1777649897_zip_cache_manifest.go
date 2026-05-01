package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_3202283038")
		if err != nil {
			return err
		}

		records, err := app.FindRecordsByFilter("zip_cache", "1=1", "", 0, 0)
		if err != nil {
			return err
		}
		for _, record := range records {
			if err := app.Delete(record); err != nil {
				return err
			}
		}

		collection.Fields.RemoveById("json536828135")
		collection.Fields.RemoveById("json652890421")
		if collection.Fields.GetByName("manifest") == nil {
			if err := collection.Fields.AddMarshaledJSONAt(4, []byte(`{
				"hidden": false,
				"id": "json1777649897",
				"maxSize": 0,
				"name": "manifest",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "json"
			}`)); err != nil {
				return err
			}
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_3202283038")
		if err != nil {
			return err
		}

		records, err := app.FindRecordsByFilter("zip_cache", "1=1", "", 0, 0)
		if err != nil {
			return err
		}
		for _, record := range records {
			if err := app.Delete(record); err != nil {
				return err
			}
		}

		collection.Fields.RemoveById("json1777649897")
		if collection.Fields.GetByName("hashes") == nil {
			if err := collection.Fields.AddMarshaledJSONAt(4, []byte(`{
				"hidden": false,
				"id": "json536828135",
				"maxSize": 0,
				"name": "hashes",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "json"
			}`)); err != nil {
				return err
			}
		}
		if collection.Fields.GetByName("filenames") == nil {
			if err := collection.Fields.AddMarshaledJSONAt(5, []byte(`{
				"hidden": false,
				"id": "json652890421",
				"maxSize": 0,
				"name": "filenames",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "json"
			}`)); err != nil {
				return err
			}
		}
		return app.Save(collection)
	})
}
