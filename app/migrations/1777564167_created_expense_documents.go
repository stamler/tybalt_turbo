package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		jsonData := `{
			"createRule": null,
			"deleteRule": null,
			"fields": [
				{
					"autogeneratePattern": "[a-z0-9]{15}",
					"hidden": false,
					"id": "text3208210256",
					"max": 15,
					"min": 15,
					"name": "id",
					"pattern": "^[a-z0-9]+$",
					"presentable": false,
					"primaryKey": true,
					"required": true,
					"system": true,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "file1777564167",
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
					"required": true,
					"system": false,
					"thumbs": null,
					"type": "file"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text1777564167",
					"max": 64,
					"min": 64,
					"name": "attachment_hash",
					"pattern": "^[a-fA-F0-9]{64}$",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"cascadeDelete": false,
					"collectionId": "_pb_users_auth_",
					"hidden": false,
					"id": "relation1777564167",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "uploaded_by",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"hidden": false,
					"id": "autodate2990389176",
					"name": "created",
					"onCreate": true,
					"onUpdate": false,
					"presentable": false,
					"system": false,
					"type": "autodate"
				},
				{
					"hidden": false,
					"id": "autodate3332085495",
					"name": "updated",
					"onCreate": true,
					"onUpdate": true,
					"presentable": false,
					"system": false,
					"type": "autodate"
				}
			],
			"id": "pbc_2089657321",
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_expense_documents_attachment_hash` + "`" + ` ON ` + "`" + `expense_documents` + "`" + ` (` + "`" + `attachment_hash` + "`" + `) WHERE ` + "`" + `attachment_hash` + "`" + ` != ''"
			],
			"listRule": null,
			"name": "expense_documents",
			"system": false,
			"type": "base",
			"updateRule": null,
			"viewRule": null
		}`

		collection := &core.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}
		if err := app.Save(collection); err != nil {
			return err
		}

		expenses, err := app.FindCollectionByNameOrId("expenses")
		if err != nil {
			return err
		}
		if expenses.Fields.GetByName("attachment_document") == nil {
			if err := expenses.Fields.AddMarshaledJSON([]byte(`{
				"cascadeDelete": false,
				"collectionId": "pbc_2089657321",
				"hidden": false,
				"id": "relation1777564167",
				"maxSelect": 1,
				"minSelect": 0,
				"name": "attachment_document",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "relation"
			}`)); err != nil {
				return err
			}
		}

		return app.Save(expenses)
	}, func(app core.App) error {
		expenses, err := app.FindCollectionByNameOrId("expenses")
		if err == nil {
			expenses.Fields.RemoveById("relation1777564167")
			if err := app.Save(expenses); err != nil {
				return err
			}
		}

		collection, err := app.FindCollectionByNameOrId("pbc_2089657321")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
