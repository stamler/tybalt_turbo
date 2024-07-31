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
			"id": "fpri53nrr2xgoov",
			"created": "2024-07-30 14:46:21.293Z",
			"updated": "2024-07-30 14:46:21.293Z",
			"name": "time_sheets",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "1hsureno",
					"name": "uid",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "_pb_users_auth_",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "9rbt8bsn",
					"name": "manager_id",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "_pb_users_auth_",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "wdwbzxxl",
					"name": "work_week_hours",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 8,
						"max": 40,
						"noDecimal": false
					}
				},
				{
					"system": false,
					"id": "toak4dg5",
					"name": "salary",
					"type": "bool",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {}
				},
				{
					"system": false,
					"id": "xoebt068",
					"name": "week_ending",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"pattern": "^\\\\d{4}-\\\\d{2}-\\\\d{2}$"
					}
				}
			],
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_NSP4DAc` + "`" + ` ON ` + "`" + `time_sheets` + "`" + ` (\n  ` + "`" + `uid` + "`" + `,\n  ` + "`" + `week_ending` + "`" + `\n)"
			],
			"listRule": "@request.auth.id != \"\"",
			"viewRule": "@request.auth.id != \"\"",
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

		collection, err := dao.FindCollectionByNameOrId("fpri53nrr2xgoov")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
