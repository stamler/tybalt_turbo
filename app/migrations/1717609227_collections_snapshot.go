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
				"updated": "2024-03-22 12:59:17.612Z",
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
					},
					{
						"system": false,
						"id": "users_avatar",
						"name": "avatar",
						"type": "file",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"mimeTypes": [
								"image/jpeg",
								"image/png",
								"image/svg+xml",
								"image/gif",
								"image/webp"
							],
							"thumbs": null,
							"maxSelect": 1,
							"maxSize": 5242880,
							"protected": false
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
					"allowEmailAuth": true,
					"allowOAuth2Auth": true,
					"allowUsernameAuth": true,
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
				"updated": "2024-06-04 15:57:10.945Z",
				"name": "jobs",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "zloyds7s",
						"name": "job_number",
						"type": "text",
						"required": true,
						"presentable": true,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": "(P)?[0-9]{2}-[0-9]{3,4}(-[0-9]{1,2})?(-[0-9])?"
						}
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_V1RKd7H` + "`" + ` ON ` + "`" + `jobs` + "`" + ` (` + "`" + `job_number` + "`" + `)"
				],
				"listRule": "@request.auth.id != \"\"",
				"viewRule": null,
				"createRule": "@request.auth.id = \"f2j5a8vk006baub\"",
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "glmf9xpnwgpwudm",
				"created": "2024-04-03 18:24:43.543Z",
				"updated": "2024-04-03 18:27:58.290Z",
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
						"id": "ocxmutn0",
						"name": "opening_datetime_off",
						"type": "text",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": null,
							"max": null,
							"pattern": "^(?:\\d{4})-(?:0[1-9]|1[0-2])-(?:0[1-9]|[1-2][0-9]|3[0-1])$"
						}
					},
					{
						"system": false,
						"id": "mghymcxc",
						"name": "opening_op",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 0,
							"max": 10000,
							"noDecimal": true
						}
					},
					{
						"system": false,
						"id": "suw6v59k",
						"name": "opening_ov",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 0,
							"max": 100000,
							"noDecimal": true
						}
					},
					{
						"system": false,
						"id": "v6thasef",
						"name": "untracked_time_off",
						"type": "bool",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {}
					},
					{
						"system": false,
						"id": "9qfzn9ab",
						"name": "timestamp",
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
						"id": "65fcj2kd",
						"name": "default_charge_out_rate",
						"type": "number",
						"required": false,
						"presentable": false,
						"unique": false,
						"options": {
							"min": 5000,
							"max": 100000,
							"noDecimal": true
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
				"id": "cnqv0wm8hly7r3n",
				"created": "2024-06-03 16:51:22.959Z",
				"updated": "2024-06-04 15:57:19.717Z",
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
					}
				],
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_fQtszvd` + "`" + ` ON ` + "`" + `time_types` + "`" + ` (` + "`" + `code` + "`" + `)"
				],
				"listRule": "  @request.auth.id != \"\"",
				"viewRule": null,
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "3esdddggow6dykr",
				"created": "2024-06-03 16:59:48.189Z",
				"updated": "2024-06-04 15:57:29.487Z",
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
				"listRule": "  @request.auth.id != \"\"",
				"viewRule": null,
				"createRule": null,
				"updateRule": null,
				"deleteRule": null,
				"options": {}
			},
			{
				"id": "ranctx5xgih6n3a",
				"created": "2024-06-04 13:35:40.992Z",
				"updated": "2024-06-04 18:25:09.287Z",
				"name": "time_entries",
				"type": "base",
				"system": false,
				"schema": [
					{
						"system": false,
						"id": "svbnxyon",
						"name": "date",
						"type": "date",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": "2024-06-01 08:00:00.000Z",
							"max": "2050-05-31 08:00:00.000Z"
						}
					},
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
						"id": "lriva8hh",
						"name": "week_ending",
						"type": "date",
						"required": true,
						"presentable": false,
						"unique": false,
						"options": {
							"min": "",
							"max": ""
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
						"id": "xio2lxq5",
						"name": "job_hours",
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
						"name": "workrecord",
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
					}
				],
				"indexes": [],
				"listRule": "@request.auth.id != \"\"",
				"viewRule": "@request.auth.id != \"\"",
				"createRule": "@request.auth.id != \"\"",
				"updateRule": "@request.auth.id != \"\"",
				"deleteRule": null,
				"options": {}
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
