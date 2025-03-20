package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1501628665")
		if err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(3, []byte(`{
			"cascadeDelete": false,
			"collectionId": "pmxhrqhngh60icm",
			"hidden": false,
			"id": "relation1168844159",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "user_claim",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1501628665")
		if err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("relation1168844159")

		return app.Save(collection)
	})
}
