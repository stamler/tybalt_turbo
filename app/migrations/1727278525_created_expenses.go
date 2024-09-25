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
			"id": "o1vpz1mm7qsfoyy",
			"created": "2024-09-25 15:35:25.447Z",
			"updated": "2024-09-25 15:35:25.447Z",
			"name": "expenses",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "1pjwom6l",
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
					"id": "8suftgyi",
					"name": "date",
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
					"id": "cggnkeqm",
					"name": "division",
					"type": "relation",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "3esdddggow6dykr",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "spdshefk",
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
				},
				{
					"system": false,
					"id": "st2japdo",
					"name": "total",
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
					"id": "puynywev",
					"name": "payment_type",
					"type": "select",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"maxSelect": 1,
						"values": [
							"OnAccount",
							"Expense",
							"CorporateCreditCard",
							"Allowance",
							"FuelCard",
							"Mileage",
							"PersonalReimbursement"
						]
					}
				},
				{
					"system": false,
					"id": "oplty6th",
					"name": "vendor_name",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 2,
						"max": null,
						"pattern": ""
					}
				},
				{
					"system": false,
					"id": "edbixzlo",
					"name": "attachment",
					"type": "file",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"mimeTypes": [
							"application/pdf",
							"image/jpeg",
							"image/png",
							"image/heic"
						],
						"thumbs": [],
						"maxSelect": 1,
						"maxSize": 5242880,
						"protected": false
					}
				},
				{
					"system": false,
					"id": "wjdoqxuu",
					"name": "rejector",
					"type": "relation",
					"required": false,
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
					"id": "yy4wgwrx",
					"name": "rejected",
					"type": "date",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": "",
						"max": ""
					}
				},
				{
					"system": false,
					"id": "fpshyvya",
					"name": "rejection_reason",
					"type": "text",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": 5,
						"max": null,
						"pattern": ""
					}
				},
				{
					"system": false,
					"id": "uoh8s8ea",
					"name": "approver",
					"type": "relation",
					"required": false,
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
					"id": "p19lerrm",
					"name": "approved",
					"type": "date",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": "",
						"max": ""
					}
				},
				{
					"system": false,
					"id": "3f4rryq3",
					"name": "job",
					"type": "relation",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "yovqzrnnomp0lkx",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "gszhhxl6",
					"name": "category",
					"type": "relation",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "nrwhbwowokwu6cr",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
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

		collection, err := dao.FindCollectionByNameOrId("o1vpz1mm7qsfoyy")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
