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
			"id": "1v6i9rrpniuatcx",
			"created": "2024-10-08 20:08:54.731Z",
			"updated": "2024-10-08 20:08:54.731Z",
			"name": "clients",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "hpftesxg",
					"name": "name",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 2,
						"max": null,
						"pattern": ""
					}
				}
			],
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_aXJh3FO` + "`" + ` ON ` + "`" + `clients` + "`" + ` (` + "`" + `name` + "`" + `)"
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

		collection, err := dao.FindCollectionByNameOrId("1v6i9rrpniuatcx")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
