package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		jsonData := `{
			"id": "3v7wxidd2f9yhf9",
			"created": "2024-10-08 20:13:27.681Z",
			"updated": "2024-10-08 20:13:27.681Z",
			"name": "contacts",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "isgvpgue",
					"name": "surname",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"pattern": ""
					}
				},
				{
					"system": false,
					"id": "sdagw2zd",
					"name": "given_name",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"pattern": ""
					}
				},
				{
					"system": false,
					"id": "hfcua49b",
					"name": "email",
					"type": "email",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"exceptDomains": null,
						"onlyDomains": null
					}
				},
				{
					"system": false,
					"id": "w4csqqjx",
					"name": "client",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "1v6i9rrpniuatcx",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				}
			],
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_KxKk01Y` + "`" + ` ON ` + "`" + `contacts` + "`" + ` (\n  ` + "`" + `surname` + "`" + `,\n  ` + "`" + `given_name` + "`" + `\n)"
			],
			"listRule": null,
			"viewRule": null,
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {}
		}`

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("3v7wxidd2f9yhf9")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
