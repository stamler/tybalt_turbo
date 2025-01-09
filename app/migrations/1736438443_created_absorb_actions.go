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
			"id": "yw3bni1ad22grdo",
			"created": "2025-01-09 16:00:43.838Z",
			"updated": "2025-01-09 16:00:43.838Z",
			"name": "absorb_actions",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "mm9oylkv",
					"name": "collection_name",
					"type": "text",
					"required": true,
					"presentable": true,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"pattern": ""
					}
				},
				{
					"system": false,
					"id": "vjvkevat",
					"name": "target_id",
					"type": "text",
					"required": true,
					"presentable": true,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"pattern": ""
					}
				},
				{
					"system": false,
					"id": "zt83vc63",
					"name": "absorbed_records",
					"type": "json",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"maxSize": 2000000
					}
				},
				{
					"system": false,
					"id": "d80tdp67",
					"name": "updated_references",
					"type": "json",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"maxSize": 2000000
					}
				}
			],
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_T0t8iRR` + "`" + ` ON ` + "`" + `absorb_actions` + "`" + ` (` + "`" + `collection_name` + "`" + `)"
			],
			"listRule": "@request.auth.user_claims_via_uid.cid.name ?= 'absorb'",
			"viewRule": "@request.auth.user_claims_via_uid.cid.name ?= 'absorb'",
			"createRule": null,
			"updateRule": null,
			"deleteRule": "@request.auth.user_claims_via_uid.cid.name ?= 'absorb'",
			"options": {}
		}`

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("yw3bni1ad22grdo")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
