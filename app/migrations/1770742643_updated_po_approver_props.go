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
		if err := collection.Fields.AddMarshaledJSONAt(4, []byte(`{
			"hidden": false,
			"id": "number252573802",
			"max": null,
			"min": 0,
			"name": "project_max",
			"onlyInt": true,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(5, []byte(`{
			"hidden": false,
			"id": "number3874684565",
			"max": null,
			"min": 0,
			"name": "sponsorship_max",
			"onlyInt": true,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(6, []byte(`{
			"hidden": false,
			"id": "number127518376",
			"max": null,
			"min": 0,
			"name": "staff_and_social_max",
			"onlyInt": true,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(7, []byte(`{
			"hidden": false,
			"id": "number1001852773",
			"max": null,
			"min": 0,
			"name": "media_and_event_max",
			"onlyInt": true,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(8, []byte(`{
			"hidden": false,
			"id": "number3848991897",
			"max": null,
			"min": 0,
			"name": "computer_max",
			"onlyInt": true,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
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
		collection.Fields.RemoveById("number252573802")

		// remove field
		collection.Fields.RemoveById("number3874684565")

		// remove field
		collection.Fields.RemoveById("number127518376")

		// remove field
		collection.Fields.RemoveById("number1001852773")

		// remove field
		collection.Fields.RemoveById("number3848991897")

		return app.Save(collection)
	})
}
