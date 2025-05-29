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

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "tjcbf5e3",
			"max": 0,
			"min": 0,
			"name": "po_number",
			"pattern": "^([1-9]\\d{3})-(\\d{4})(?:-(0[1-9]|[1-9]\\d))?$",
			"presentable": true,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "tjcbf5e3",
			"max": 0,
			"min": 0,
			"name": "po_number",
			"pattern": "^([1-9]\\d{3})-(0{3}[1-9]|0{2}[1-9]\\d|0[1-9]\\d{2}|[1-3]\\d{3}|4[0-8]\\d{2}|49[0-9]{2})(?:-(0[1-9]|[1-9]\\d))?$",
			"presentable": true,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
