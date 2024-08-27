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
			"id": "g3surmbkacieshv",
			"created": "2024-08-27 14:15:33.594Z",
			"updated": "2024-08-27 14:15:33.594Z",
			"name": "time_sheet_reviewers",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "6i9fbu28",
					"name": "time_sheet",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "fpri53nrr2xgoov",
						"cascadeDelete": true,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "lelfbeex",
					"name": "reviewer",
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
					"id": "d5utnnkq",
					"name": "reviewed",
					"type": "date",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": "",
						"max": ""
					}
				}
			],
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_MVTW8sD` + "`" + ` ON ` + "`" + `time_sheet_reviewers` + "`" + ` (\n  ` + "`" + `time_sheet` + "`" + `,\n  ` + "`" + `reviewer` + "`" + `\n)"
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

		collection, err := dao.FindCollectionByNameOrId("g3surmbkacieshv")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
