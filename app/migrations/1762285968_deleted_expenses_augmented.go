package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_4039657122")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	}, func(app core.App) error {
		jsonData := `{
			"createRule": null,
			"deleteRule": null,
			"fields": [
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text3208210256",
					"max": 0,
					"min": 0,
					"name": "id",
					"pattern": "^[a-z0-9]+$",
					"presentable": false,
					"primaryKey": true,
					"required": true,
					"system": true,
					"type": "text"
				},
				{
					"cascadeDelete": false,
					"collectionId": "_pb_users_auth_",
					"hidden": false,
					"id": "_clone_0JHt",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "uid",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_kK9Q",
					"max": 0,
					"min": 0,
					"name": "date",
					"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"cascadeDelete": false,
					"collectionId": "3esdddggow6dykr",
					"hidden": false,
					"id": "_clone_zvHQ",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "division",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_phxn",
					"max": 0,
					"min": 0,
					"name": "description",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "_clone_3EgP",
					"max": null,
					"min": 0,
					"name": "total",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "_clone_iCOq",
					"maxSelect": 1,
					"name": "payment_type",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "select",
					"values": [
						"OnAccount",
						"Expense",
						"CorporateCreditCard",
						"Allowance",
						"FuelCard",
						"Mileage",
						"PersonalReimbursement"
					]
				},
				{
					"hidden": false,
					"id": "_clone_k3gH",
					"maxSelect": 1,
					"maxSize": 5242880,
					"mimeTypes": [
						"application/pdf",
						"image/jpeg",
						"image/png",
						"image/heic"
					],
					"name": "attachment",
					"presentable": false,
					"protected": false,
					"required": false,
					"system": false,
					"thumbs": null,
					"type": "file"
				},
				{
					"cascadeDelete": false,
					"collectionId": "_pb_users_auth_",
					"hidden": false,
					"id": "_clone_PMOm",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "rejector",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "relation"
				},
				{
					"hidden": false,
					"id": "_clone_Qqis",
					"max": "",
					"min": "",
					"name": "rejected",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "date"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_Ud4j",
					"max": 0,
					"min": 5,
					"name": "rejection_reason",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"cascadeDelete": false,
					"collectionId": "_pb_users_auth_",
					"hidden": false,
					"id": "_clone_c49a",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "approver",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "relation"
				},
				{
					"hidden": false,
					"id": "_clone_3vqd",
					"max": "",
					"min": "",
					"name": "approved",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "date"
				},
				{
					"cascadeDelete": false,
					"collectionId": "yovqzrnnomp0lkx",
					"hidden": false,
					"id": "_clone_w9ED",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "job",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "relation"
				},
				{
					"cascadeDelete": false,
					"collectionId": "nrwhbwowokwu6cr",
					"hidden": false,
					"id": "_clone_8j5p",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "category",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "relation"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_aMCE",
					"max": 0,
					"min": 0,
					"name": "pay_period_ending",
					"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "_clone_Wr7N",
					"maxSelect": 4,
					"name": "allowance_types",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "select",
					"values": [
						"Lodging",
						"Breakfast",
						"Lunch",
						"Dinner"
					]
				},
				{
					"hidden": false,
					"id": "_clone_p9NL",
					"name": "submitted",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "bool"
				},
				{
					"cascadeDelete": false,
					"collectionId": "_pb_users_auth_",
					"hidden": false,
					"id": "_clone_immQ",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "committer",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "relation"
				},
				{
					"hidden": false,
					"id": "_clone_EqUA",
					"max": "",
					"min": "",
					"name": "committed",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "date"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_fA3q",
					"max": 0,
					"min": 0,
					"name": "committed_week_ending",
					"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "_clone_T24r",
					"max": null,
					"min": 0,
					"name": "distance",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_oSgo",
					"max": 0,
					"min": 0,
					"name": "cc_last_4_digits",
					"pattern": "^\\d{4}$",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"cascadeDelete": false,
					"collectionId": "m19q72syy0e3lvm",
					"hidden": false,
					"id": "_clone_BrDm",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "purchase_order",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "relation"
				},
				{
					"cascadeDelete": false,
					"collectionId": "y0xvnesailac971",
					"hidden": false,
					"id": "_clone_IrEn",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "vendor",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "relation"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_gzKs",
					"max": 0,
					"min": 0,
					"name": "purchase_order_number",
					"pattern": "^([1-9]\\d{3})-(\\d{4})(?:-(0[1-9]|[1-9]\\d))?$",
					"presentable": true,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_uJNs",
					"max": 0,
					"min": 2,
					"name": "client_name",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_L8FM",
					"max": 0,
					"min": 3,
					"name": "category_name",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_mEeL",
					"max": 0,
					"min": 0,
					"name": "job_number",
					"pattern": "^(P)?[0-9]{2}-[0-9]{3,4}(-[0-9]{1,2})?(-[0-9])?$",
					"presentable": true,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_szWV",
					"max": 0,
					"min": 3,
					"name": "job_description",
					"pattern": "",
					"presentable": true,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_HLUs",
					"max": 0,
					"min": 2,
					"name": "division_name",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_2qZr",
					"max": 0,
					"min": 1,
					"name": "division_code",
					"pattern": "",
					"presentable": true,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_1HFX",
					"max": 0,
					"min": 3,
					"name": "vendor_name",
					"pattern": "",
					"presentable": true,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_NapK",
					"max": 0,
					"min": 3,
					"name": "vendor_alias",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "json3388866023",
					"maxSize": 1,
					"name": "uid_name",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json1197482769",
					"maxSize": 1,
					"name": "approver_name",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json1398595088",
					"maxSize": 1,
					"name": "rejector_name",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				}
			],
			"id": "pbc_4039657122",
			"indexes": [],
			"listRule": "uid = @request.auth.id ||\n(approver = @request.auth.id && submitted = true) ||\n(approved != \"\" && @request.auth.user_claims_via_uid.cid.name ?= 'commit') ||\n(committed != \"\" && @request.auth.user_claims_via_uid.cid.name ?= 'report')",
			"name": "expenses_augmented",
			"system": false,
			"type": "view",
			"updateRule": null,
			"viewQuery": "-- The augmented expenses query is to create a view that includes all the required\n-- fields for the expenses list page UI.\n\n-- copy the existing expenses listRule / viewRule\n-- uid = @request.auth.id ||\n-- (approver = @request.auth.id && submitted = true) ||\n-- (approved != \"\" && @request.auth.user_claims_via_uid.cid.name ?= 'commit') ||\n-- (committed != \"\" && @request.auth.user_claims_via_uid.cid.name ?= 'report')\nSELECT e.id,\n  e.uid,\n  e.date,\n  e.division,\n  e.description,\n  e.total,\n  e.payment_type,\n  e.attachment,\n  e.rejector,\n  e.rejected,\n  e.rejection_reason,\n  e.approver,\n  e.approved,\n  e.job,\n  e.category,\n  e.pay_period_ending,\n  e.allowance_types,\n  e.submitted,\n  e.committer,\n  e.committed,\n  e.committed_week_ending,\n  e.distance,\n  e.cc_last_4_digits,\n  e.purchase_order,\n  e.vendor,\n  po.po_number as purchase_order_number,\n  cl.name as client_name,\n  ca.name as category_name,\n  j.number as job_number,\n  j.description as job_description,\n  d.name as division_name,\n  d.code as division_code,\n  v.name as vendor_name,\n  v.alias as vendor_alias,\n  (p0.given_name || ' ' || p0.surname) as uid_name,\n  (p1.given_name || ' ' || p1.surname) as approver_name,\n  (p2.given_name || ' ' || p2.surname) as rejector_name\nFROM expenses e\nLEFT JOIN jobs j ON e.job = j.id\nLEFT JOIN clients cl ON j.client = cl.id\nLEFT JOIN vendors v ON e.vendor = v.id\nLEFT JOIN divisions d ON e.division = d.id\nLEFT JOIN categories ca ON e.category = ca.id\nLEFT JOIN profiles p0 ON e.uid = p0.uid\nLEFT JOIN profiles p1 ON e.approver = p1.uid\nLEFT JOIN profiles p2 ON e.rejector = p2.uid\nLEFT JOIN purchase_orders po ON e.purchase_order = po.id;",
			"viewRule": "uid = @request.auth.id ||\n(approver = @request.auth.id && submitted = true) ||\n(approved != \"\" && @request.auth.user_claims_via_uid.cid.name ?= 'commit') ||\n(committed != \"\" && @request.auth.user_claims_via_uid.cid.name ?= 'report')"
		}`

		collection := &core.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
