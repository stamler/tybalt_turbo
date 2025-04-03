package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("glmf9xpnwgpwudm")
		if err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(7, []byte(`{
			"hidden": false,
			"id": "select2676255945",
			"maxSelect": 1,
			"name": "notification_type",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "select",
			"values": [
				"email_text",
				"email_html"
			]
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("glmf9xpnwgpwudm")
		if err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(7, []byte(`{
			"hidden": false,
			"id": "select2676255945",
			"maxSelect": 2,
			"name": "notification_types",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "select",
			"values": [
				"email_text",
				"email_html"
			]
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
