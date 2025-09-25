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
		if err := collection.Fields.AddMarshaledJSONAt(20, []byte(`{
			"hidden": false,
			"id": "number486793109",
			"max": null,
			"min": null,
			"name": "outstanding_balance",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(21, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text1832948875",
			"max": 0,
			"min": 0,
			"name": "outstanding_balance_date",
			"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
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
		collection.Fields.RemoveById("number486793109")

		// remove field
		collection.Fields.RemoveById("text1832948875")

		return app.Save(collection)
	})
}
