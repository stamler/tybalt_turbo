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
			"id": "kn6f5sfmzjogw63",
			"created": "2024-10-22 18:48:13.306Z",
			"updated": "2024-10-22 18:48:13.306Z",
			"name": "po_approvers",
			"type": "view",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "pqu2ncyf",
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
					"id": "0onwdwi1",
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
			"listRule": null,
			"viewRule": null,
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {
				"query": "SELECT p.uid AS id, p.surname AS surname, p.given_name AS given_name \nFROM profiles p\nINNER JOIN user_claims u ON p.uid = u.uid\nINNER JOIN claims c ON u.cid = c.id\nWHERE c.name = 'po_approver'"
			}
		}`

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("kn6f5sfmzjogw63")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
