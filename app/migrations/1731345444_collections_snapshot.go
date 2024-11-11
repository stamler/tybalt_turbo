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
		jsonData := `[
			{
				"id": "_pb_users_auth_",
				"created": "2024-03-22 12:59:17.612Z",
				"updated": "2024-10-31 15:55:15.805Z",
				"name": "users",
				"type": "auth",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "users_name",
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
				"listRule": "id = @request.auth.id",
				"viewRule": "// The caller is logged in and either\n@request.auth.id != \"\" &&\n(\n  // 1. The caller id is equal to the record id or\n  @request.auth.id = id ||\n  // 2. The record has the po_approver claim or\n  user_claims_via_uid.cid.name ?= 'po_approver' ||\n  // 3. The record has the tapr claim or\n  user_claims_via_uid.cid.name ?= 'tapr' ||\n  // 4. The caller has the tapr claim\n  @request.auth.user_claims_via_uid.cid.name ?= 'tapr'\n)",
				"createRule": "",
				"updateRule": "id = @request.auth.id",
				"deleteRule": "id = @request.auth.id",
				"options": {
					"allowEmailAuth": false,
					"allowOAuth2Auth": true,
					"allowUsernameAuth": false,
					"exceptEmailDomains": null,
					"manageRule": null,
					"minPasswordLength": 8,
					"onlyEmailDomains": null,
					"onlyVerified": false,
					"requireEmail": false
				}
			},
			{
				"id": "yovqzrnnomp0lkx",
				"created": "2024-03-24 14:50:35.856Z",
				"updated": "2024-10-10 14:49:20.303Z",
				"name": "jobs",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "zloyds7s",
						"name": "number",
						"type": "text",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": "^(P)?[0-9]{2}-[0-9]{3,4}(-[0-9]{1,2})?(-[0-9])?$"
						}
					},
					{
						"system": false,
						"id": "seuuugpd",
						"name": "description",
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
						"id": "efj2t5lj",
						"name": "client",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "1v6i9rrpniuatcx",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "k65clvxw",
						"name": "contact",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "3v7wxidd2f9yhf9",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "erlnpgrl",
						"name": "manager",
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
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_V1RKd7H` + "`" + ` ON ` + "`" + `jobs` + "`" + ` (` + "`" + `number` + "`" + `)"
				],
				"listRule": "@request.auth.id != \"\"",
				"viewRule": "@request.auth.id != \"\"",
				"createRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job' &&\n\n// the contact belongs to the client\n@request.data.contact.client = @request.data.client",
				"updateRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job' &&\n\n// the contact belongs to the client\n@request.data.contact.client = @request.data.client",
				"deleteRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'admin' &&\n\n// prevent deletion of jobs if there are referencing time_entries\n@collection.time_entries.job != id &&\n\n// prevent deletion of jobs if there are referencing purchase orders\n@collection.purchase_orders.job != id &&\n\n// prevent deletion of jobs if there are referencing expenses\n@collection.expenses.job != id",
				"options": {}
			},
			{
				"id": "glmf9xpnwgpwudm",
				"created": "2024-04-03 18:24:43.543Z",
				"updated": "2024-10-31 15:56:13.974Z",
				"name": "profiles",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "fxlkxvsy",
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
						"id": "e7uz2a2n",
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
					},
					{
						"system": false,
						"id": "gudkt7qq",
						"name": "manager",
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
						"id": "rwknt5er",
						"name": "alternate_manager",
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
						"id": "naf0546m",
						"name": "default_division",
						"type": "relation",
						"required": false,
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
						"id": "e8mbl3rh",
						"name": "uid",
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
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_dvV9kj4` + "`" + ` ON ` + "`" + `profiles` + "`" + ` (` + "`" + `uid` + "`" + `)"
				],
				"listRule": "@request.auth.id != \"\" &&\n(uid = @request.auth.id || manager = @request.auth.id)",
				"viewRule": "// The caller is logged in and either\n@request.auth.id != \"\" &&\n(\n  // 1. The caller id matches the record uid or\n  @request.auth.id = uid ||\n  // 2. The record has the tapr claim\n  uid.user_claims_via_uid.cid.name ?= 'tapr' ||\n  // 3. The record has the po_approver claim\n  uid.user_claims_via_uid.cid.name ?= 'po_approver'\n)",
				"createRule": "@request.auth.id != \"\" &&\nuid = @request.auth.id",
				"updateRule": "@request.auth.id != \"\" &&\nuid = @request.auth.id",
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "cnqv0wm8hly7r3n",
				"created": "2024-06-03 16:51:22.959Z",
				"updated": "2024-09-06 20:07:36.263Z",
				"name": "time_types",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "eoitnxlx",
						"name": "code",
						"type": "text",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"min": 1,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "rwphtkdf",
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
					},
					{
						"system": false,
						"id": "q4ppqv3i",
						"name": "description",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "wfkvnoh0",
						"name": "allowed_fields",
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
						"id": "onuwxebx",
						"name": "required_fields",
						"type": "json",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"maxSize": 2000000
						}
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_fQtszvd` + "`" + ` ON ` + "`" + `time_types` + "`" + ` (` + "`" + `code` + "`" + `)"
				],
				"listRule": "@request.auth.id != \"\"",
				"viewRule": "@request.auth.id != \"\"",
				"createRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'tt'",
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "3esdddggow6dykr",
				"created": "2024-06-03 16:59:48.189Z",
				"updated": "2024-09-06 20:07:36.263Z",
				"name": "divisions",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "cmlhnbq8",
						"name": "code",
						"type": "text",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"min": 1,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "xc9wslmg",
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
					"CREATE UNIQUE INDEX ` + "`" + `idx_rbNPJNF` + "`" + ` ON ` + "`" + `divisions` + "`" + ` (` + "`" + `code` + "`" + `)"
				],
				"listRule": "@request.auth.id != \"\"",
				"viewRule": "@request.auth.id != \"\"",
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "ranctx5xgih6n3a",
				"created": "2024-06-04 13:35:40.992Z",
				"updated": "2024-09-24 21:24:55.661Z",
				"name": "time_entries",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "jlqkb6jb",
						"name": "division",
						"type": "relation",
						"required": false,
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
						"id": "rjasv0rb",
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
						"id": "amfas3ce",
						"name": "hours",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 0,
							"max": 18,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "4eu16q2p",
						"name": "description",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "xkbfo3ev",
						"name": "time_type",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "cnqv0wm8hly7r3n",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "r18fowxw",
						"name": "meals_hours",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 0,
							"max": 3,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "jcncwdjc",
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
						"id": "fjcrzqdc",
						"name": "work_record",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": "^[FKQ][0-9]{2}-[0-9]{3,}(-[0-9]+)?$"
						}
					},
					{
						"system": false,
						"id": "n8ys3o83",
						"name": "payout_request_amount",
						"type": "number",
						"required": false,
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
						"id": "qjavwq6p",
						"name": "date",
						"type": "text",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": "^\\d{4}-\\d{2}-\\d{2}$"
						}
					},
					{
						"system": false,
						"id": "kbl2eccm",
						"name": "week_ending",
						"type": "text",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": "^\\d{4}-\\d{2}-\\d{2}$"
						}
					},
					{
						"system": false,
						"id": "gih0hrty",
						"name": "tsid",
						"type": "relation",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "fpri53nrr2xgoov",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "l5mlhdph",
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
				"listRule": "@request.auth.id = uid ||\n@request.auth.id = tsid.approver ||\n@request.auth.user_claims_via_uid.cid.name ?= 'report'",
				"viewRule": "@request.auth.id = uid ||\n@request.auth.id = tsid.approver ||\n@request.auth.user_claims_via_uid.cid.name ?= 'report'",
				"createRule": "@request.auth.id != \"\" &&\n// if present, the category belongs to the job, otherwise is blank\n(\n  // compare the new category to the new job\n  ( @request.data.job:isset = true && @request.data.category.job = @request.data.job ) ||\n  @request.data.category = \"\"\n)",
				"updateRule": "// the creating user can edit if the entry is not yet part of a timesheet\nuid = @request.auth.id && tsid = \"\" &&\n// if present, the category belongs to the job, otherwise is blank\n(\n  // the job is unchanged, compare the new category to job\n  ( @request.data.job:isset = false && @request.data.category.job = job ) ||\n  // the job has changed, compare the new category to the new job\n  ( @request.data.job:isset = true && @request.data.category.job = @request.data.job ) ||\n  @request.data.category = \"\"\n)",
				"deleteRule": "// request is from the creator and entry is not part of timesheet\n@request.auth.id = uid && tsid = \"\"",
				"options": {}
			},
			{
				"id": "l0tpyvfnr1inncv",
				"created": "2024-06-21 14:19:34.603Z",
				"updated": "2024-10-22 18:09:00.948Z",
				"name": "claims",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "xcillp3i",
						"name": "name",
						"type": "text",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"min": 2,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "7zmxmcdq",
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
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_3KEX8wA` + "`" + ` ON ` + "`" + `claims` + "`" + ` (` + "`" + `name` + "`" + `)"
				],
				"listRule": null,
				"viewRule": "@request.auth.id != \"\"",
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "pmxhrqhngh60icm",
				"created": "2024-06-21 14:41:11.871Z",
				"updated": "2024-10-22 18:20:17.285Z",
				"name": "user_claims",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "pkwnhskh",
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
						"id": "1xyqocjd",
						"name": "cid",
						"type": "relation",
						"required": true,
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
						"id": "gfyrln8y",
						"name": "payload",
						"type": "json",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"maxSize": 2000000
						}
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_6dSZCrb` + "`" + ` ON ` + "`" + `user_claims` + "`" + ` (\n  ` + "`" + `uid` + "`" + `,\n  ` + "`" + `cid` + "`" + `\n)"
				],
				"listRule": null,
				"viewRule": null,
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "phpak4pjznt98yu",
				"created": "2024-06-26 01:48:25.926Z",
				"updated": "2024-09-06 20:07:36.272Z",
				"name": "managers",
				"type": "view",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "komw2gef",
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
						"id": "awpifkph",
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
				"listRule": "@request.auth.id != \"\"",
				"viewRule": "@request.auth.id != \"\"",
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {
					"query": "SELECT p.uid AS id, p.surname AS surname, p.given_name AS given_name \nFROM profiles p\nINNER JOIN user_claims u ON p.uid = u.uid\nINNER JOIN claims c ON u.cid = c.id\nWHERE c.name = 'tapr'"
				}
			},
			{
				"id": "fpri53nrr2xgoov",
				"created": "2024-07-30 14:46:21.293Z",
				"updated": "2024-10-01 15:30:02.764Z",
				"name": "time_sheets",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "1hsureno",
						"name": "uid",
						"type": "relation",
						"required": true,
						"presentable": true,
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
						"id": "wdwbzxxl",
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
						"id": "toak4dg5",
						"name": "salary",
						"type": "bool",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {}
					},
					{
						"system": false,
						"id": "xoebt068",
						"name": "week_ending",
						"type": "text",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": "^\\d{4}-\\d{2}-\\d{2}$"
						}
					},
					{
						"system": false,
						"id": "32m2ceei",
						"name": "submitted",
						"type": "bool",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {}
					},
					{
						"system": false,
						"id": "pfwfhk8a",
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
						"id": "lwzae5gf",
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
						"id": "8wtvhwar",
						"name": "rejection_reason",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "yzugnurw",
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
						"id": "vue3mlk0",
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
						"id": "fjjylizi",
						"name": "committed",
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
						"id": "8sig1vra",
						"name": "committer",
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
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_NSP4DAc` + "`" + ` ON ` + "`" + `time_sheets` + "`" + ` (\n  ` + "`" + `uid` + "`" + `,\n  ` + "`" + `week_ending` + "`" + `\n)"
				],
				"listRule": "@request.auth.id != \"\" && (\n  uid = @request.auth.id\n)",
				"viewRule": "@request.auth.id != \"\" && (\n  uid = @request.auth.id\n)",
				"createRule": null,
				"updateRule": "(rejected = true && rejector != \"\" && rejection_reason != \"\") || (rejected = false)",
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "zc850lb2wclrr87",
				"created": "2024-07-30 18:12:19.576Z",
				"updated": "2024-09-06 20:07:36.264Z",
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
					},
					{
						"system": false,
						"id": "6yqnu4zu",
						"name": "off_rotation_permitted",
						"type": "bool",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {}
					},
					{
						"system": false,
						"id": "fmuapxvl",
						"name": "skip_min_time_check",
						"type": "select",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"maxSelect": 1,
							"values": [
								"no",
								"on_next_bundle",
								"yes"
							]
						}
					},
					{
						"system": false,
						"id": "jtq5elga",
						"name": "opening_date",
						"type": "text",
						"required": false,
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
						"id": "gnwvxtyk",
						"name": "opening_op",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 0,
							"max": 500,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "4pjevdlg",
						"name": "opening_ov",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 0,
							"max": 500,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "d6fgkrwy",
						"name": "payroll_id",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": "^(?:[1-9]\\d*|CMS[0-9]{1,2})$"
						}
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_UpEVC7E` + "`" + ` ON ` + "`" + `admin_profiles` + "`" + ` (` + "`" + `uid` + "`" + `)",
					"CREATE UNIQUE INDEX ` + "`" + `idx_XnQ4v11` + "`" + ` ON ` + "`" + `admin_profiles` + "`" + ` (` + "`" + `payroll_id` + "`" + `)"
				],
				"listRule": null,
				"viewRule": null,
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "g3surmbkacieshv",
				"created": "2024-08-27 14:15:33.594Z",
				"updated": "2024-09-06 20:07:36.264Z",
				"name": "time_sheet_reviewers",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "6i9fbu28",
						"name": "time_sheet",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "fpri53nrr2xgoov",
							"cascadeDelete": true,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "lelfbeex",
						"name": "reviewer",
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
						"id": "d5utnnkq",
						"name": "reviewed",
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
					"CREATE UNIQUE INDEX ` + "`" + `idx_MVTW8sD` + "`" + ` ON ` + "`" + `time_sheet_reviewers` + "`" + ` (\n  ` + "`" + `time_sheet` + "`" + `,\n  ` + "`" + `reviewer` + "`" + `\n)"
				],
				"listRule": "@request.auth.id = @collection.users.time_sheets_via_approver.approver ||\n@request.auth.id = reviewer",
				"viewRule": "@request.auth.id = @collection.users.time_sheets_via_approver.approver ||\n@request.auth.id = reviewer",
				"createRule": "@request.auth.id != \"\" &&\n@request.auth.time_sheets_via_approver.id ?= time_sheet",
				"updateRule": null,
				"deleteRule": "@request.auth.id != \"\" &&\ntime_sheet ?= @request.auth.time_sheets_via_approver.id",
				"options": {}
			},
			{
				"id": "c9b90wqyjpqa7tk",
				"created": "2024-09-03 19:14:57.257Z",
				"updated": "2024-09-06 20:07:36.264Z",
				"name": "payroll_year_end_dates",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "kxtzlmig",
						"name": "date",
						"type": "text",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": "^\\d{4}-\\d{2}-\\d{2}$"
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
			},
			{
				"id": "6z8rcof9bkpzz1t",
				"created": "2024-09-04 19:30:56.139Z",
				"updated": "2024-09-10 19:56:13.347Z",
				"name": "time_off",
				"type": "view",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "fq96v0fz",
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
					},
					{
						"system": false,
						"id": "giuljwev",
						"name": "manager_uid",
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
						"id": "cnpa5nvi",
						"name": "manager",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "jtjbf7bk",
						"name": "opening_date",
						"type": "text",
						"required": false,
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
						"id": "t5xcprup",
						"name": "opening_ov",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 0,
							"max": 500,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "g9p5inga",
						"name": "opening_op",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 0,
							"max": 500,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "f0qt4hou",
						"name": "used_ov",
						"type": "number",
						"required": false,
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
						"id": "r59xqo1f",
						"name": "used_op",
						"type": "number",
						"required": false,
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
						"id": "fxhe4blz",
						"name": "timesheet_ov",
						"type": "number",
						"required": false,
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
						"id": "ien41elp",
						"name": "timesheet_op",
						"type": "number",
						"required": false,
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
						"id": "hmdwqze0",
						"name": "last_ov",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "3cc7wqgx",
						"name": "last_op",
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
				"listRule": "@request.auth.id = id ||\n@request.auth.id = manager_uid",
				"viewRule": "@request.auth.id = id ||\n@request.auth.id = manager_uid",
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {
					"query": "SELECT \n    p.uid as id,\n    CAST(CONCAT(p.surname, ', ', p.given_name) AS TEXT) AS name,\n    p.manager AS manager_uid,\n    CAST(CONCAT(mp.surname, ', ', mp.given_name) AS TEXT) AS manager,\n    ap.opening_date,\n    ap.opening_ov,\n    ap.opening_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_op,\n    CAST(MAX(CASE WHEN tt.code = 'OV' THEN te.date END) AS TEXT) AS last_ov,\n    CAST(MAX(CASE WHEN tt.code = 'OP' THEN te.date END) AS TEXT) AS last_op\nFROM \n    profiles p\nLEFT JOIN \n    admin_profiles ap ON p.uid = ap.uid\nLEFT JOIN \n    profiles mp ON p.manager = mp.uid\nLEFT JOIN \n    time_entries te ON p.uid = te.uid\nLEFT JOIN \n    time_types tt ON te.time_type = tt.id\nWHERE \n    te.week_ending >= ap.opening_date\nGROUP BY \n    p.uid, p.surname, p.given_name, p.manager, mp.surname, mp.given_name, ap.opening_date, ap.opening_ov, ap.opening_op"
				}
			},
			{
				"id": "m19q72syy0e3lvm",
				"created": "2024-09-10 18:39:22.442Z",
				"updated": "2024-10-31 15:58:51.175Z",
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
							"pattern": "^(20[2-9]\\d)-(0{3}[1-9]|0{2}[1-9]\\d|0[1-9]\\d{2}|[1-3]\\d{3}|4[0-8]\\d{2}|49[0-9]{2})$"
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
								"Cancelled",
								"Closed"
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
							"pattern": "^\\d{4}-\\d{2}-\\d{2}$"
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
							"pattern": "^\\d{4}-\\d{2}-\\d{2}$"
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
							"min": 0,
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
							"protected": false
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
					},
					{
						"system": false,
						"id": "fzmkxved",
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
						"id": "mzwtgxtc",
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
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_6Ao8pCT` + "`" + ` ON ` + "`" + `purchase_orders` + "`" + ` (` + "`" + `po_number` + "`" + `) WHERE ` + "`" + `po_number` + "`" + ` != ''"
				],
				"listRule": "@request.auth.id = uid ||\n@request.auth.id = approver ||\n@request.auth.id = second_approver",
				"viewRule": "@request.auth.id = uid ||\n@request.auth.id = approver ||\n@request.auth.id = second_approver",
				"createRule": "// the caller is authenticated\n@request.auth.id != \"\" &&\n\n// no po_number is submitted\n(@request.data.po_number:isset = false || @request.data.po_number = \"\") &&\n\n// status is Unapproved\n@request.data.status = \"Unapproved\" &&\n\n// the uid is missing or is equal to the authenticated user's id\n(@request.data.uid:isset = false || @request.data.uid = @request.auth.id) &&\n\n// no rejection properties are submitted\n@request.data.rejector:isset = false &&\n@request.data.rejected:isset = false &&\n@request.data.rejection_reason:isset = false &&\n\n// approved isn't set and approver has the right claim. Test divisions in payload in hooks\n@request.data.approved:isset = false &&\n@request.data.approver.user_claims_via_uid.cid.name ?= 'po_approver' &&\n\n// no second approver properties are submitted\n@request.data.second_approver:isset = false &&\n@request.data.second_approval:isset = false &&\n@request.data.second_approver_claim:isset = false &&\n\n// no cancellation properties are submitted\n@request.data.cancelled:isset = false &&\n@request.data.canceller:isset = false &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // compare the new category to the new job\n  ( @request.data.job:isset = true && @request.data.category.job = @request.data.job ) ||\n  @request.data.category = \"\"\n)",
				"updateRule": "// only the creator can update the record\nuid = @request.auth.id &&\n\n// status is Unapproved and no approvals have been performed\nstatus = 'Unapproved' &&\napproved = \"\" &&\nsecond_approval = \"\"\n\n// no po_number is submitted\n(@request.data.po_number:isset = false || po_number = @request.data.po_number) &&\n\n// no rejection properties are submitted\n(@request.data.rejector:isset = false || rejector = @request.data.rejector) &&\n(@request.data.rejected:isset = false || rejected = @request.data.rejected) &&\n(@request.data.rejection_reason:isset = false || rejection_reason = @request.data.rejection_reason) &&\n\n// approved isn't set and approver has the right claim. Test divisions in payload in hooks\n(@request.data.approved:isset = false || approved = @request.data.approved) &&\n@request.data.approver.user_claims_via_uid.cid.name ?= 'po_approver' &&\n\n// no second approver properties are submitted\n(@request.data.second_approver:isset = false || second_approver = @request.data.second_approver) &&\n(@request.data.second_approval:isset = false || second_approval = @request.data.second_approval) &&\n(@request.data.second_approver_claim:isset = false || second_approver_claim = @request.data.second_approver_claim) &&\n\n// no cancellation properties are submitted\n(@request.data.cancelled:isset = false || cancelled = @request.data.cancelled) &&\n(@request.data.canceller:isset = false || canceller = @request.data.canceller) &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // the job is unchanged, compare the new category to job\n  ( @request.data.job:isset = false && @request.data.category.job = job ) ||\n  // the job has changed, compare the new category to the new job\n  ( @request.data.job:isset = true && @request.data.category.job = @request.data.job ) ||\n  @request.data.category = \"\"\n)",
				"deleteRule": "@request.auth.id = uid && status = 'Unapproved'",
				"options": {}
			},
			{
				"id": "nrwhbwowokwu6cr",
				"created": "2024-09-20 19:18:51.476Z",
				"updated": "2024-10-10 14:49:55.859Z",
				"name": "categories",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "oyndkpey",
						"name": "name",
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
						"id": "cedjug8b",
						"name": "job",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "yovqzrnnomp0lkx",
							"cascadeDelete": true,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_SF6A76x` + "`" + ` ON ` + "`" + `categories` + "`" + ` (\n  ` + "`" + `job` + "`" + `,\n  ` + "`" + `name` + "`" + `\n)"
				],
				"listRule": "@request.auth.id != \"\"",
				"viewRule": "@request.auth.id != \"\"",
				"createRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job'",
				"updateRule": null,
				"deleteRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job' &&\n\n// prevent deletion of categories if there are referencing time_entries\n@collection.time_entries.category != id &&\n\n// prevent deletion of categories if there are referencing purchase orders\n@collection.purchase_orders.category != id &&\n\n// prevent deletion of categories if there are referencing expenses\n@collection.expenses.category != id",
				"options": {}
			},
			{
				"id": "o1vpz1mm7qsfoyy",
				"created": "2024-09-25 15:35:25.447Z",
				"updated": "2024-10-30 20:34:23.593Z",
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
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "st2japdo",
						"name": "total",
						"type": "number",
						"required": false,
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
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
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
					},
					{
						"system": false,
						"id": "6ocqzyet",
						"name": "pay_period_ending",
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
						"id": "tahxw786",
						"name": "allowance_types",
						"type": "select",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"maxSelect": 4,
							"values": [
								"Lodging",
								"Breakfast",
								"Lunch",
								"Dinner"
							]
						}
					},
					{
						"system": false,
						"id": "cpt1x5gr",
						"name": "submitted",
						"type": "bool",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {}
					},
					{
						"system": false,
						"id": "djy3zkz8",
						"name": "committer",
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
						"id": "bmzx8tgn",
						"name": "committed",
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
						"id": "d13a8jxo",
						"name": "committed_week_ending",
						"type": "text",
						"required": false,
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
						"id": "hsvbnev9",
						"name": "distance",
						"type": "number",
						"required": false,
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
						"id": "gv2z62zj",
						"name": "cc_last_4_digits",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": "^\\d{4}$"
						}
					},
					{
						"system": false,
						"id": "pxd0mvyh",
						"name": "purchase_order",
						"type": "relation",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "m19q72syy0e3lvm",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					}
				],
				"indexes": [],
				"listRule": "uid = @request.auth.id ||\n(approver = @request.auth.id && submitted = true) ||\n(approved != \"\" && @request.auth.user_claims_via_uid.cid.name ?= 'commit') ||\n(committed != \"\" && @request.auth.user_claims_via_uid.cid.name ?= 'report')",
				"viewRule": "uid = @request.auth.id ||\n(approver = @request.auth.id && submitted = true) ||\n(approved != \"\" && @request.auth.user_claims_via_uid.cid.name ?= 'commit') ||\n(committed != \"\" && @request.auth.user_claims_via_uid.cid.name ?= 'report')",
				"createRule": "// the caller is authenticated\n@request.auth.id != \"\" &&\n\n// the uid is equal to the authenticated user's id\n@request.data.uid = @request.auth.id &&\n\n// no rejection properties are submitted\n@request.data.rejector:isset = false &&\n@request.data.rejected:isset = false &&\n@request.data.rejection_reason:isset = false &&\n\n// no approval properties are submitted\n@request.data.approved:isset = false &&\n@request.data.approver:isset = false &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // compare the new category to the new job\n  ( @request.data.job:isset = true && @request.data.category.job = @request.data.job ) ||\n  @request.data.category = \"\"\n)",
				"updateRule": "// only the creator can update the record\nuid = @request.auth.id &&\n\n// the uid must not change\n(@request.data.uid:isset = false || uid = @request.data.uid) &&\n\n// no rejection properties are submitted\n(@request.data.rejector:isset = false || rejector = @request.data.rejector) &&\n(@request.data.rejected:isset = false || rejected = @request.data.rejected) &&\n(@request.data.rejection_reason:isset = false || rejection_reason = @request.data.rejection_reason) &&\n\n// submitted is not changed\n(@request.data.submitted:isset = false || submitted = @request.data.submitted) &&\n\n// no approval properties are submitted\n(@request.data.approved:isset = false || approved = @request.data.approved) &&\n(@request.data.approver:isset = false || approver = @request.data.approver) &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // the job is unchanged, compare the new category to job\n  ( @request.data.job:isset = false && @request.data.category.job = job ) ||\n  // the job has changed, compare the new category to the new job\n  ( @request.data.job:isset = true && @request.data.category.job = @request.data.job ) ||\n  @request.data.category = \"\"\n)",
				"deleteRule": "@request.auth.id = uid && committed = \"\"",
				"options": {}
			},
			{
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
			},
			{
				"id": "1v6i9rrpniuatcx",
				"created": "2024-10-08 20:08:54.731Z",
				"updated": "2024-10-09 16:40:20.640Z",
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
				"listRule": "@request.auth.id != \"\"",
				"viewRule": "@request.auth.id != \"\"",
				"createRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job'",
				"updateRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job'",
				"deleteRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job' &&\n// prevent deletion of clients if there are referencing jobs\n@collection.jobs.client != id",
				"options": {}
			},
			{
				"id": "3v7wxidd2f9yhf9",
				"created": "2024-10-08 20:13:27.681Z",
				"updated": "2024-10-10 14:55:55.497Z",
				"name": "contacts",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "isgvpgue",
						"name": "surname",
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
						"id": "sdagw2zd",
						"name": "given_name",
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
						"id": "hfcua49b",
						"name": "email",
						"type": "email",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"exceptDomains": null,
							"onlyDomains": null
						}
					},
					{
						"system": false,
						"id": "w4csqqjx",
						"name": "client",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "1v6i9rrpniuatcx",
							"cascadeDelete": true,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					}
				],
				"indexes": [
					"CREATE INDEX ` + "`" + `idx_KxKk01Y` + "`" + ` ON ` + "`" + `contacts` + "`" + ` (\n  ` + "`" + `surname` + "`" + `,\n  ` + "`" + `given_name` + "`" + `\n)",
					"CREATE UNIQUE INDEX ` + "`" + `idx_0KoVkzQ` + "`" + ` ON ` + "`" + `contacts` + "`" + ` (` + "`" + `email` + "`" + `)"
				],
				"listRule": "@request.auth.id != \"\"",
				"viewRule": "@request.auth.id != \"\"",
				"createRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job'",
				"updateRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job'",
				"deleteRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job' &&\n// prevent deletion of contacts if there are referencing jobs\n@collection.jobs.contact != id",
				"options": {}
			},
			{
				"id": "5z24r2v5jgh8qft",
				"created": "2024-10-18 15:27:27.001Z",
				"updated": "2024-10-22 18:03:15.601Z",
				"name": "time_amendments",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "oszzgyip",
						"name": "division",
						"type": "relation",
						"required": false,
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
						"id": "2h0enwkz",
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
						"id": "1esmkvan",
						"name": "hours",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 0,
							"max": 18,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "c4hphpdm",
						"name": "description",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": ""
						}
					},
					{
						"system": false,
						"id": "ad7feyjt",
						"name": "time_type",
						"type": "relation",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"collectionId": "cnqv0wm8hly7r3n",
							"cascadeDelete": false,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "yjapnpeh",
						"name": "meals_hours",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 0,
							"max": 3,
							"noDecimal": false
						}
					},
					{
						"system": false,
						"id": "dgwsotxu",
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
						"id": "8yws3dwb",
						"name": "work_record",
						"type": "text",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": "^[FKQ][0-9]{2}-[0-9]{3,}(-[0-9]+)?$"
						}
					},
					{
						"system": false,
						"id": "5zavlc9z",
						"name": "payout_request_amount",
						"type": "number",
						"required": false,
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
						"id": "zhfe6rbd",
						"name": "date",
						"type": "text",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": "^\\d{4}-\\d{2}-\\d{2}$"
						}
					},
					{
						"system": false,
						"id": "9e1umw0s",
						"name": "week_ending",
						"type": "text",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": "^\\d{4}-\\d{2}-\\d{2}$"
						}
					},
					{
						"system": false,
						"id": "xy68bh6o",
						"name": "tsid",
						"type": "relation",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"collectionId": "fpri53nrr2xgoov",
							"cascadeDelete": true,
							"minSelect": null,
							"maxSelect": 1,
							"displayFields": null
						}
					},
					{
						"system": false,
						"id": "i1uzlmch",
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
					},
					{
						"system": false,
						"id": "clpvzg0c",
						"name": "creator",
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
						"id": "cjxpxn9c",
						"name": "committed",
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
						"id": "anj6odqu",
						"name": "committer",
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
						"id": "dbenhrit",
						"name": "salary",
						"type": "bool",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {}
					},
					{
						"system": false,
						"id": "vwe32gf1",
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
						"id": "6xltfvly",
						"name": "committed_week_ending",
						"type": "text",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": "^\\d{4}-\\d{2}-\\d{2}$"
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
			},
			{
				"id": "kn6f5sfmzjogw63",
				"created": "2024-10-22 18:48:13.306Z",
				"updated": "2024-10-22 20:13:08.729Z",
				"name": "po_approvers",
				"type": "view",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "nyoqujsd",
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
						"id": "2d5gnz78",
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
					},
					{
						"system": false,
						"id": "vgaujcuy",
						"name": "divisions",
						"type": "json",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"maxSize": 2000000
						}
					}
				],
				"indexes": [],
				"listRule": "@request.auth.id != \"\"",
				"viewRule": "@request.auth.id != \"\"",
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {
					"query": "SELECT p.uid AS id, p.surname AS surname, p.given_name AS given_name, u.payload as divisions \nFROM profiles p\nINNER JOIN user_claims u ON p.uid = u.uid\nINNER JOIN claims c ON u.cid = c.id\nWHERE c.name = 'po_approver'"
				}
			}
		]`

		collections := []*models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collections); err != nil {
			return err
		}

		return daos.New(db).ImportCollections(collections, true, nil)
	}, func(db dbx.Builder) error {
		return nil
	})
}
