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
			"id": "7sy46ddznlst4av",
			"created": "2024-06-05 18:41:16.050Z",
			"updated": "2024-06-05 18:41:16.050Z",
			"name": "files",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "ukutbh22",
					"name": "file",
					"type": "file",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"mimeTypes": [],
						"thumbs": [],
						"maxSelect": 1,
						"maxSize": 5242880,
						"protected": false
					}
				},
				{
					"system": false,
					"id": "rzvw5qch",
					"name": "name",
					"type": "text",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"pattern": ""
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

		collection, err := dao.FindCollectionByNameOrId("7sy46ddznlst4av")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
