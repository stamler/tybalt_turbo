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
		collection.Fields.RemoveById("s2yffwz9")

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(32, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "relation1897617465",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "priority_second_approver",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(29, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "s2yffwz9",
			"max": 0,
			"min": 0,
			"name": "vendor_name",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("relation1897617465")

		return app.Save(collection)
	})
}
