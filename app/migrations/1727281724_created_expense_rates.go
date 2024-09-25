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
			"id": "kbohbd4ww45zf23",
			"created": "2024-09-25 16:28:44.348Z",
			"updated": "2024-09-25 16:28:44.348Z",
			"name": "expense_rates",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "27qaxv2u",
					"name": "effective_date",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"pattern": "^\\d{4}-\\d{2}-\\d{2}$"
					}
				},
				{
					"system": false,
					"id": "nwllwvdz",
					"name": "breakfast",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 0,
						"max": null,
						"noDecimal": false
					}
				},
				{
					"system": false,
					"id": "iz3crqwa",
					"name": "lunch",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 0,
						"max": null,
						"noDecimal": false
					}
				},
				{
					"system": false,
					"id": "uzmiw2za",
					"name": "dinner",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 0,
						"max": null,
						"noDecimal": false
					}
				},
				{
					"system": false,
					"id": "drfzivwc",
					"name": "lodging",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 0,
						"max": null,
						"noDecimal": false
					}
				},
				{
					"system": false,
					"id": "uf3fazuz",
					"name": "mileage",
					"type": "json",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"maxSize": 2000000
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

		collection, err := dao.FindCollectionByNameOrId("kbohbd4ww45zf23")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
