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
		if err := collection.Fields.AddMarshaledJSONAt(14, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text178857617",
			"max": 0,
			"min": 0,
			"name": "mobile_phone",
			"pattern": "^\\+1 \\(\\d{3}\\) \\d{3}-\\d{4}$",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(15, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text711640347",
			"max": 0,
			"min": 0,
			"name": "job_title",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(16, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "text1768657410",
			"max": 0,
			"min": 0,
			"name": "personal_vehicle_insurance_expiry",
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
		collection, err := app.FindCollectionByNameOrId("zc850lb2wclrr87")
		if err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("text178857617")

		// remove field
		collection.Fields.RemoveById("text711640347")

		// remove field
		collection.Fields.RemoveById("text1768657410")

		return app.Save(collection)
	})
}
