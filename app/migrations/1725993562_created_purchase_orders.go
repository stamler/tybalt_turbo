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
			"id": "m19q72syy0e3lvm",
			"created": "2024-09-10 18:39:22.442Z",
			"updated": "2024-09-10 18:39:22.442Z",
			"name": "purchase_orders",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "tjcbf5e3",
					"name": "po_number",
					"type": "text",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"pattern": "^(20[2-9]\\d)-(\\d{4})$"
					}
				},
				{
					"system": false,
					"id": "od79ozm1",
					"name": "status",
					"type": "select",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"maxSelect": 1,
						"values": [
							"Unapproved",
							"Active",
							"Cancelled"
						]
					}
				},
				{
					"system": false,
					"id": "l0bykiha",
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
					"id": "wwwtd51w",
					"name": "type",
					"type": "select",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"maxSelect": 1,
						"values": [
							"Normal",
							"Cumulative",
							"Recurring"
						]
					}
				},
				{
					"system": false,
					"id": "4c4auzt9",
					"name": "date",
					"type": "text",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"pattern": "^\\\\d{4}-\\\\d{2}-\\\\d{2}$"
					}
				},
				{
					"system": false,
					"id": "hqtvqmtx",
					"name": "end_date",
					"type": "text",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"pattern": "^\\\\d{4}-\\\\d{2}-\\\\d{2}$"
					}
				},
				{
					"system": false,
					"id": "65m4tbko",
					"name": "frequency",
					"type": "select",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"maxSelect": 1,
						"values": [
							"Weekly",
							"Biweekly",
							"Monthly"
						]
					}
				},
				{
					"system": false,
					"id": "nfuhmtlf",
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
					"id": "6uz2s2c6",
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
					"id": "azgktu8n",
					"name": "total",
					"type": "number",
					"required": true,
					"presentable": false,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"noDecimal": false
					}
				},
				{
					"system": false,
					"id": "qakahtme",
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
							"CorporateCreditCard"
						]
					}
				},
				{
					"system": false,
					"id": "s2yffwz9",
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
					"id": "0clolnui",
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
						"protected": true
					}
				},
				{
					"system": false,
					"id": "5rekg0iz",
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
					"id": "qj3tjhw6",
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
					"id": "war1qt5e",
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
					"id": "xiadfk0k",
					"name": "approver",
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
					"id": "kmdaym5e",
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
					"id": "elntbwwz",
					"name": "second_approver_claim",
					"type": "relation",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"collectionId": "l0tpyvfnr1inncv",
						"cascadeDelete": false,
						"minSelect": null,
						"maxSelect": 1,
						"displayFields": null
					}
				},
				{
					"system": false,
					"id": "wwnnme9m",
					"name": "second_approver",
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
					"id": "j3v3g8vs",
					"name": "second_approval",
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
					"id": "4tjxswnx",
					"name": "canceller",
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
					"id": "lm1hbt7h",
					"name": "cancelled",
					"type": "date",
					"required": false,
					"presentable": false,
					"unique": false,
					"options": {
						"min": "",
						"max": ""
					}
				}
			],
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_6Ao8pCT` + "`" + ` ON ` + "`" + `purchase_orders` + "`" + ` (` + "`" + `po_number` + "`" + `)"
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

		collection, err := dao.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
