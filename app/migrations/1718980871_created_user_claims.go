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
			"id": "pmxhrqhngh60icm",
			"created": "2024-06-21 14:41:11.871Z",
			"updated": "2024-06-21 14:41:11.871Z",
			"name": "user_claims",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "pkwnhskh",
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
					"id": "1xyqocjd",
					"name": "cid",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "l0tpyvfnr1inncv",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				}
			],
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_6dSZCrb` + "`" + ` ON ` + "`" + `user_claims` + "`" + ` (\n  ` + "`" + `uid` + "`" + `,\n  ` + "`" + `cid` + "`" + `\n)"
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

		collection, err := dao.FindCollectionByNameOrId("pmxhrqhngh60icm")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
