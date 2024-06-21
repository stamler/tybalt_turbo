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
			"id": "l0tpyvfnr1inncv",
			"created": "2024-06-21 14:19:34.603Z",
			"updated": "2024-06-21 14:19:34.603Z",
			"name": "claims",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "xcillp3i",
					"name": "name",
					"type": "text",
					"required": true,
					"presentable": true,
					"unique": false,
					"options": {
						"min": 2,
						"max": null,
						"pattern": ""
					}
				},
				{
					"system": false,
					"id": "7zmxmcdq",
					"name": "description",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 5,
						"max": null,
						"pattern": ""
					}
				}
			],
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_3KEX8wA` + "`" + ` ON ` + "`" + `claims` + "`" + ` (` + "`" + `name` + "`" + `)"
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

		collection, err := dao.FindCollectionByNameOrId("l0tpyvfnr1inncv")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
