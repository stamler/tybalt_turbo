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
				"updated": "2024-08-30 20:16:47.805Z",
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
				"viewRule": "id = @request.auth.id",
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
				"updated": "2024-06-21 18:06:58.975Z",
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
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_V1RKd7H` + "`" + ` ON ` + "`" + `jobs` + "`" + ` (` + "`" + `number` + "`" + `)"
				],
				"listRule": "@request.auth.id != \"\"",
				"viewRule": "@request.auth.id != \"\"",
				"createRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job'",
				"updateRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job'",
				"deleteRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'admin' &&\n// prevent deletion of jobs if there are referencing time_entries\n@collection.time_entries.job != id",
				"options": {}
			},
			{
				"id": "glmf9xpnwgpwudm",
				"created": "2024-04-03 18:24:43.543Z",
				"updated": "2024-09-04 20:16:55.635Z",
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
				"viewRule": "@request.auth.id != \"\" &&\n(uid = @request.auth.id || manager = @request.auth.id)",
				"createRule": "@request.auth.id != \"\" &&\nuid = @request.auth.id",
				"updateRule": "@request.auth.id != \"\" &&\nuid = @request.auth.id",
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "cnqv0wm8hly7r3n",
				"created": "2024-06-03 16:51:22.959Z",
				"updated": "2024-06-24 17:45:18.337Z",
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
				"updated": "2024-07-02 19:06:16.299Z",
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
				"updated": "2024-07-31 17:12:44.555Z",
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
						"id": "ymg43f6u",
						"name": "category",
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
					}
				],
				"indexes": [],
				"listRule": "@request.auth.id != \"\"",
				"viewRule": "@request.auth.id != \"\"",
				"createRule": "@request.auth.id != \"\"",
				"updateRule": "@request.auth.id != \"\"",
				"deleteRule": "@request.auth.id = uid",
				"options": {}
			},
			{
				"id": "l0tpyvfnr1inncv",
				"created": "2024-06-21 14:19:34.603Z",
				"updated": "2024-06-21 14:19:34.603Z",
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
				"viewRule": null,
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "pmxhrqhngh60icm",
				"created": "2024-06-21 14:41:11.871Z",
				"updated": "2024-06-21 14:41:11.871Z",
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
				"updated": "2024-06-26 01:48:25.926Z",
				"name": "managers",
				"type": "view",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "1c0gdbiw",
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
						"id": "2obso2rr",
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
				"updated": "2024-08-29 13:07:57.097Z",
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
						"id": "79frceqq",
						"name": "locked",
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
						"id": "4939m45n",
						"name": "locker",
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
						"id": "ovtg2rsw",
						"name": "rejected",
						"type": "bool",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {}
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
				"updated": "2024-09-04 20:12:25.609Z",
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
				"updated": "2024-08-28 15:48:41.687Z",
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
				"updated": "2024-09-04 12:32:22.596Z",
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
				"updated": "2024-09-06 18:26:13.756Z",
				"name": "time_off",
				"type": "view",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "xzme4vo9",
						"name": "name",
						"type": "json",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"maxSize": 1
						}
					},
					{
						"system": false,
						"id": "ag18tebo",
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
						"id": "ae7nheo3",
						"name": "manager",
						"type": "json",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"maxSize": 1
						}
					},
					{
						"system": false,
						"id": "rntgtrld",
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
						"id": "txei3zb8",
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
						"id": "xjwp5y3f",
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
						"id": "fjee4nop",
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
						"id": "rrbauxhr",
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
						"id": "2ki5ozes",
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
						"id": "1mitcqxa",
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
						"id": "siti0flt",
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
						"id": "cw5lcpuq",
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
					"query": "SELECT \n    p.uid as id,\n    CONCAT(p.surname, ', ', p.given_name) AS name,\n    p.manager AS manager_uid,\n    CONCAT(mp.surname, ', ', mp.given_name) AS manager,\n    ap.opening_date,\n    ap.opening_ov,\n    ap.opening_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_op,\n    CAST(MAX(CASE WHEN tt.code = 'OV' THEN te.date END) AS TEXT) AS last_ov,\n    CAST(MAX(CASE WHEN tt.code = 'OP' THEN te.date END) AS TEXT) AS last_op\nFROM \n    profiles p\nLEFT JOIN \n    admin_profiles ap ON p.uid = ap.uid\nLEFT JOIN \n    profiles mp ON p.manager = mp.uid\nLEFT JOIN \n    time_entries te ON p.uid = te.uid\nLEFT JOIN \n    time_types tt ON te.time_type = tt.id\nWHERE \n    te.week_ending >= ap.opening_date\nGROUP BY \n    p.uid, p.surname, p.given_name, p.manager, mp.surname, mp.given_name, ap.opening_date, ap.opening_ov, ap.opening_op"
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
