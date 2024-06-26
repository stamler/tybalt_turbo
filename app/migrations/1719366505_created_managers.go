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
			"id": "phpak4pjznt98yu",
			"created": "2024-06-26 01:48:25.926Z",
			"updated": "2024-06-26 01:48:25.926Z",
			"name": "managers",
			"type": "view",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "1c0gdbiw",
					"name": "surname",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 2,
						"max": 48,
						"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$"
					}
				},
				{
					"system": false,
					"id": "2obso2rr",
					"name": "given_name",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 2,
						"max": 48,
						"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$"
					}
				}
			],
			"indexes": [],
			"listRule": "@request.auth.id != \"\"",
			"viewRule": "@request.auth.id != \"\"",
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {
				"query": "SELECT p.uid AS id, p.surname AS surname, p.given_name AS given_name \nFROM profiles p\nINNER JOIN user_claims u ON p.uid = u.uid\nINNER JOIN claims c ON u.cid = c.id\nWHERE c.name = 'tapr'"
			}
		}`

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("phpak4pjznt98yu")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
