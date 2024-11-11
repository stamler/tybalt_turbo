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
			"id": "y0xvnesailac971",
			"created": "2024-11-11 18:19:35.595Z",
			"updated": "2024-11-11 18:19:35.595Z",
			"name": "vendors",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "so6nx9uo",
					"name": "name",
					"type": "text",
					"required": true,
					"presentable": true,
					"unique": false,
					"options": {
						"min": 3,
						"max": null,
						"pattern": ""
					}
				},
				{
					"system": false,
					"id": "sxfocdv1",
					"name": "alias",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 3,
						"max": null,
						"pattern": ""
					}
				},
				{
					"system": false,
					"id": "7lzhalcf",
					"name": "status",
					"type": "select",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"maxSelect": 1,
						"values": [
							"Active",
							"Inactive"
						]
					}
				}
			],
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_GCZxhiM` + "`" + ` ON ` + "`" + `vendors` + "`" + ` (` + "`" + `name` + "`" + `)",
				"CREATE UNIQUE INDEX ` + "`" + `idx_c8OTvkU` + "`" + ` ON ` + "`" + `vendors` + "`" + ` (` + "`" + `alias` + "`" + `)"
			],
			"listRule": "@request.auth.id != \"\"",
			"viewRule": "@request.auth.id != \"\"",
			"createRule": "@request.auth.user_claims_via_uid.cid.name ?= 'payables_admin'",
			"updateRule": "@request.auth.user_claims_via_uid.cid.name ?= 'payables_admin'",
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

		collection, err := dao.FindCollectionByNameOrId("y0xvnesailac971")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
