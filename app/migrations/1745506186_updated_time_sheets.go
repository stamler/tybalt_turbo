package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("fpri53nrr2xgoov")
		if err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(13, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text3684909290",
			"max": 0,
			"min": 0,
			"name": "payroll_id",
			"pattern": "^(?:[1-9]\\d*|CMS[0-9]{1,2})$",
			"presentable": false,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("fpri53nrr2xgoov")
		if err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("text3684909290")

		return app.Save(collection)
	})
}
