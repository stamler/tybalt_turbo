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
			"id": "zc850lb2wclrr87",
			"created": "2024-07-30 18:12:19.576Z",
			"updated": "2024-07-30 18:12:19.576Z",
			"name": "admin_profiles",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "4hsjcwtw",
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
					"id": "6of5hjva",
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
					"id": "pgwqbaui",
					"name": "salary",
					"type": "bool",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {}
				},
				{
					"system": false,
					"id": "nd2tweu3",
					"name": "default_charge_out_rate",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 50,
						"max": 1000,
						"noDecimal": false
					}
				}
			],
			"indexes": [],
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

		collection, err := dao.FindCollectionByNameOrId("zc850lb2wclrr87")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
