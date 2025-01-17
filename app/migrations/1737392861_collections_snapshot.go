package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		jsonData := `[
			{
				"authAlert": {
					"emailTemplate": {
						"body": "<p>Hello,</p>\n<p>We noticed a login to your {APP_NAME} account from a new location.</p>\n<p>If this was you, you may disregard this email.</p>\n<p><strong>If this wasn't you, you should immediately change your {APP_NAME} account password to revoke access from all other locations.</strong></p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
						"subject": "Login from a new location"
					},
					"enabled": true
				},
				"authRule": "",
				"authToken": {
					"duration": 1209600
				},
				"confirmEmailChangeTemplate": {
					"body": "<p>Hello,</p>\n<p>Click on the button below to confirm your new email address.</p>\n<p>\n  <a class=\"btn\" href=\"{APP_URL}/_/#/auth/confirm-email-change/{TOKEN}\" target=\"_blank\" rel=\"noopener\">Confirm new email</a>\n</p>\n<p><i>If you didn't ask to change your email address, you can ignore this email.</i></p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
					"subject": "Confirm your {APP_NAME} new email address"
				},
				"createRule": "",
				"deleteRule": "id = @request.auth.id",
				"emailChangeToken": {
					"duration": 1800
				},
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"cost": 10,
						"hidden": true,
						"id": "password901924565",
						"max": 0,
						"min": 8,
						"name": "password",
						"pattern": "",
						"presentable": false,
						"required": true,
						"system": true,
						"type": "password"
					},
					{
						"autogeneratePattern": "[a-zA-Z0-9_]{50}",
						"hidden": true,
						"id": "text2504183744",
						"max": 60,
						"min": 30,
						"name": "tokenKey",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"exceptDomains": null,
						"hidden": false,
						"id": "email3885137012",
						"name": "email",
						"onlyDomains": null,
						"presentable": false,
						"required": false,
						"system": true,
						"type": "email"
					},
					{
						"hidden": false,
						"id": "bool1547992806",
						"name": "emailVisibility",
						"presentable": false,
						"required": false,
						"system": true,
						"type": "bool"
					},
					{
						"hidden": false,
						"id": "bool256245529",
						"name": "verified",
						"presentable": false,
						"required": false,
						"system": true,
						"type": "bool"
					},
					{
						"autogeneratePattern": "users[0-9]{6}",
						"hidden": false,
						"id": "text4166911607",
						"max": 150,
						"min": 3,
						"name": "username",
						"pattern": "^[\\w][\\w\\.\\-]*$",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "users_name",
						"max": 0,
						"min": 0,
						"name": "name",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"fileToken": {
					"duration": 120
				},
				"id": "_pb_users_auth_",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `__pb_users_auth__username_idx` + "`" + ` ON ` + "`" + `users` + "`" + ` (username COLLATE NOCASE)",
					"CREATE UNIQUE INDEX ` + "`" + `__pb_users_auth__email_idx` + "`" + ` ON ` + "`" + `users` + "`" + ` (` + "`" + `email` + "`" + `) WHERE ` + "`" + `email` + "`" + ` != ''",
					"CREATE UNIQUE INDEX ` + "`" + `__pb_users_auth__tokenKey_idx` + "`" + ` ON ` + "`" + `users` + "`" + ` (` + "`" + `tokenKey` + "`" + `)"
				],
				"listRule": "id = @request.auth.id",
				"manageRule": null,
				"mfa": {
					"duration": 1800,
					"enabled": false,
					"rule": ""
				},
				"name": "users",
				"oauth2": {
					"enabled": false,
					"mappedFields": {
						"avatarURL": "",
						"id": "",
						"name": "",
						"username": "username"
					}
				},
				"otp": {
					"duration": 180,
					"emailTemplate": {
						"body": "<p>Hello,</p>\n<p>Your one-time password is: <strong>{OTP}</strong></p>\n<p><i>If you didn't ask for the one-time password, you can ignore this email.</i></p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
						"subject": "OTP for {APP_NAME}"
					},
					"enabled": false,
					"length": 8
				},
				"passwordAuth": {
					"enabled": false,
					"identityFields": []
				},
				"passwordResetToken": {
					"duration": 1800
				},
				"resetPasswordTemplate": {
					"body": "<p>Hello,</p>\n<p>Click on the button below to reset your password.</p>\n<p>\n  <a class=\"btn\" href=\"{APP_URL}/_/#/auth/confirm-password-reset/{TOKEN}\" target=\"_blank\" rel=\"noopener\">Reset password</a>\n</p>\n<p><i>If you didn't ask to reset your password, you can ignore this email.</i></p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
					"subject": "Reset your {APP_NAME} password"
				},
				"system": false,
				"type": "auth",
				"updateRule": "id = @request.auth.id",
				"verificationTemplate": {
					"body": "<p>Hello,</p>\n<p>Thank you for joining us at {APP_NAME}.</p>\n<p>Click on the button below to verify your email address.</p>\n<p>\n  <a class=\"btn\" href=\"{APP_URL}/_/#/auth/confirm-verification/{TOKEN}\" target=\"_blank\" rel=\"noopener\">Verify</a>\n</p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
					"subject": "Verify your {APP_NAME} email"
				},
				"verificationToken": {
					"duration": 604800
				},
				"viewRule": "// The caller is logged in and either\n@request.auth.id != \"\" &&\n(\n  // 1. The caller id is equal to the record id or\n  @request.auth.id = id ||\n  // 2. The record has the po_approver claim or\n  user_claims_via_uid.cid.name ?= 'po_approver' ||\n  // 3. The record has the tapr claim or\n  user_claims_via_uid.cid.name ?= 'tapr' ||\n  // 4. The caller has the tapr claim\n  @request.auth.user_claims_via_uid.cid.name ?= 'tapr'\n)"
			},
			{
				"createRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job' &&\n\n// the contact belongs to the client\n@request.body.contact.client = @request.body.client",
				"deleteRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'admin' &&\n\n// prevent deletion of jobs if there are referencing time_entries\n@collection.time_entries.job != id &&\n\n// prevent deletion of jobs if there are referencing purchase orders\n@collection.purchase_orders.job != id &&\n\n// prevent deletion of jobs if there are referencing expenses\n@collection.expenses.job != id",
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "zloyds7s",
						"max": 0,
						"min": 0,
						"name": "number",
						"pattern": "^(P)?[0-9]{2}-[0-9]{3,4}(-[0-9]{1,2})?(-[0-9])?$",
						"presentable": true,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "seuuugpd",
						"max": 0,
						"min": 3,
						"name": "description",
						"pattern": "",
						"presentable": true,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"cascadeDelete": false,
						"collectionId": "1v6i9rrpniuatcx",
						"hidden": false,
						"id": "efj2t5lj",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "client",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"cascadeDelete": false,
						"collectionId": "3v7wxidd2f9yhf9",
						"hidden": false,
						"id": "k65clvxw",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "contact",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"cascadeDelete": false,
						"collectionId": "_pb_users_auth_",
						"hidden": false,
						"id": "erlnpgrl",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "manager",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "yovqzrnnomp0lkx",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_V1RKd7H` + "`" + ` ON ` + "`" + `jobs` + "`" + ` (` + "`" + `number` + "`" + `)"
				],
				"listRule": "@request.auth.id != \"\"",
				"name": "jobs",
				"system": false,
				"type": "base",
				"updateRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job' &&\n\n// the contact belongs to the client\n@request.body.contact.client = @request.body.client",
				"viewRule": "@request.auth.id != \"\""
			},
			{
				"createRule": "@request.auth.id != \"\" &&\nuid = @request.auth.id",
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "fxlkxvsy",
						"max": 48,
						"min": 2,
						"name": "surname",
						"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "e7uz2a2n",
						"max": 48,
						"min": 2,
						"name": "given_name",
						"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"cascadeDelete": false,
						"collectionId": "_pb_users_auth_",
						"hidden": false,
						"id": "gudkt7qq",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "manager",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"cascadeDelete": false,
						"collectionId": "_pb_users_auth_",
						"hidden": false,
						"id": "rwknt5er",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "alternate_manager",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"cascadeDelete": false,
						"collectionId": "3esdddggow6dykr",
						"hidden": false,
						"id": "naf0546m",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "default_division",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"cascadeDelete": false,
						"collectionId": "_pb_users_auth_",
						"hidden": false,
						"id": "e8mbl3rh",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "uid",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "glmf9xpnwgpwudm",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_dvV9kj4` + "`" + ` ON ` + "`" + `profiles` + "`" + ` (` + "`" + `uid` + "`" + `)"
				],
				"listRule": "@request.auth.id != \"\" &&\n(uid = @request.auth.id || manager = @request.auth.id || @request.auth.user_claims_via_uid.cid.name ?= 'tame')",
				"name": "profiles",
				"system": false,
				"type": "base",
				"updateRule": "@request.auth.id != \"\" &&\nuid = @request.auth.id",
				"viewRule": "// The caller is logged in and either\n@request.auth.id != \"\" &&\n(\n  // 1. The caller id matches the record uid or\n  @request.auth.id = uid ||\n  // 2. The record has the tapr claim\n  uid.user_claims_via_uid.cid.name ?= 'tapr' ||\n  // 3. The record has the po_approver claim\n  uid.user_claims_via_uid.cid.name ?= 'po_approver' ||\n  // 4. The caller has the 'tame' claim\n  @request.auth.user_claims_via_uid.cid.name ?= 'tame'\n)"
			},
			{
				"createRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'tt'",
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "eoitnxlx",
						"max": 0,
						"min": 1,
						"name": "code",
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
						"id": "rwphtkdf",
						"max": 0,
						"min": 2,
						"name": "name",
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
						"id": "q4ppqv3i",
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
						"id": "wfkvnoh0",
						"maxSize": 2000000,
						"name": "allowed_fields",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "json"
					},
					{
						"hidden": false,
						"id": "onuwxebx",
						"maxSize": 2000000,
						"name": "required_fields",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "json"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "cnqv0wm8hly7r3n",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_fQtszvd` + "`" + ` ON ` + "`" + `time_types` + "`" + ` (` + "`" + `code` + "`" + `)"
				],
				"listRule": "@request.auth.id != \"\"",
				"name": "time_types",
				"system": false,
				"type": "base",
				"updateRule": null,
				"viewRule": "@request.auth.id != \"\""
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "cmlhnbq8",
						"max": 0,
						"min": 1,
						"name": "code",
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
						"id": "xc9wslmg",
						"max": 0,
						"min": 2,
						"name": "name",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "3esdddggow6dykr",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_rbNPJNF` + "`" + ` ON ` + "`" + `divisions` + "`" + ` (` + "`" + `code` + "`" + `)"
				],
				"listRule": "@request.auth.id != \"\"",
				"name": "divisions",
				"system": false,
				"type": "base",
				"updateRule": null,
				"viewRule": "@request.auth.id != \"\""
			},
			{
				"createRule": "@request.auth.id != \"\" &&\n// if present, the category belongs to the job, otherwise is blank\n(\n  // compare the new category to the new job\n  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||\n  @request.body.category = \"\"\n)",
				"deleteRule": "// request is from the creator and entry is not part of timesheet\n@request.auth.id = uid && tsid = \"\"",
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
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
						"collectionId": "3esdddggow6dykr",
						"hidden": false,
						"id": "jlqkb6jb",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "division",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"cascadeDelete": false,
						"collectionId": "_pb_users_auth_",
						"hidden": false,
						"id": "rjasv0rb",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "uid",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "amfas3ce",
						"max": 18,
						"min": 0,
						"name": "hours",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "4eu16q2p",
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
						"cascadeDelete": false,
						"collectionId": "cnqv0wm8hly7r3n",
						"hidden": false,
						"id": "xkbfo3ev",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "time_type",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "r18fowxw",
						"max": 3,
						"min": 0,
						"name": "meals_hours",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"cascadeDelete": false,
						"collectionId": "yovqzrnnomp0lkx",
						"hidden": false,
						"id": "jcncwdjc",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "job",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "fjcrzqdc",
						"max": 0,
						"min": 0,
						"name": "work_record",
						"pattern": "^[FKQ][0-9]{2}-[0-9]{3,}(-[0-9]+)?$",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "n8ys3o83",
						"max": null,
						"min": null,
						"name": "payout_request_amount",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "qjavwq6p",
						"max": 0,
						"min": 0,
						"name": "date",
						"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
						"presentable": true,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "kbl2eccm",
						"max": 0,
						"min": 0,
						"name": "week_ending",
						"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
						"presentable": true,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"cascadeDelete": false,
						"collectionId": "fpri53nrr2xgoov",
						"hidden": false,
						"id": "gih0hrty",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "tsid",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"cascadeDelete": false,
						"collectionId": "nrwhbwowokwu6cr",
						"hidden": false,
						"id": "l5mlhdph",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "category",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "ranctx5xgih6n3a",
				"indexes": [],
				"listRule": "@request.auth.id = uid ||\n@request.auth.id = tsid.approver ||\n@request.auth.user_claims_via_uid.cid.name ?= 'report'",
				"name": "time_entries",
				"system": false,
				"type": "base",
				"updateRule": "// the creating user can edit if the entry is not yet part of a timesheet\nuid = @request.auth.id && tsid = \"\" &&\n// if present, the category belongs to the job, otherwise is blank\n(\n  // the job is unchanged, compare the new category to job\n  ( @request.body.job:isset = false && @request.body.category.job = job ) ||\n  // the job has changed, compare the new category to the new job\n  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||\n  @request.body.category = \"\"\n)",
				"viewRule": "@request.auth.id = uid ||\n@request.auth.id = tsid.approver ||\n@request.auth.user_claims_via_uid.cid.name ?= 'report'"
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "xcillp3i",
						"max": 0,
						"min": 2,
						"name": "name",
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
						"id": "7zmxmcdq",
						"max": 0,
						"min": 5,
						"name": "description",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "l0tpyvfnr1inncv",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_3KEX8wA` + "`" + ` ON ` + "`" + `claims` + "`" + ` (` + "`" + `name` + "`" + `)"
				],
				"listRule": null,
				"name": "claims",
				"system": false,
				"type": "base",
				"updateRule": null,
				"viewRule": "@request.auth.id != \"\""
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
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
						"id": "pkwnhskh",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "uid",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"cascadeDelete": false,
						"collectionId": "l0tpyvfnr1inncv",
						"hidden": false,
						"id": "1xyqocjd",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "cid",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "gfyrln8y",
						"maxSize": 2000000,
						"name": "payload",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "json"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "pmxhrqhngh60icm",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_6dSZCrb` + "`" + ` ON ` + "`" + `user_claims` + "`" + ` (\n  ` + "`" + `uid` + "`" + `,\n  ` + "`" + `cid` + "`" + `\n)"
				],
				"listRule": null,
				"name": "user_claims",
				"system": false,
				"type": "base",
				"updateRule": null,
				"viewRule": null
			},
			{
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
						"autogeneratePattern": "",
						"hidden": false,
						"id": "_clone_k7yd",
						"max": 48,
						"min": 2,
						"name": "surname",
						"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "_clone_J0fF",
						"max": 48,
						"min": 2,
						"name": "given_name",
						"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					}
				],
				"id": "phpak4pjznt98yu",
				"indexes": [],
				"listRule": "@request.auth.id != \"\"",
				"name": "managers",
				"system": false,
				"type": "view",
				"updateRule": null,
				"viewQuery": "SELECT p.uid AS id, p.surname AS surname, p.given_name AS given_name \nFROM profiles p\nINNER JOIN user_claims u ON p.uid = u.uid\nINNER JOIN claims c ON u.cid = c.id\nWHERE c.name = 'tapr'",
				"viewRule": "@request.auth.id != \"\""
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
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
						"id": "1hsureno",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "uid",
						"presentable": true,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "wdwbzxxl",
						"max": 40,
						"min": 8,
						"name": "work_week_hours",
						"onlyInt": false,
						"presentable": false,
						"required": true,
						"system": false,
						"type": "number"
					},
					{
						"hidden": false,
						"id": "toak4dg5",
						"name": "salary",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "bool"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "xoebt068",
						"max": 0,
						"min": 0,
						"name": "week_ending",
						"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
						"presentable": true,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "32m2ceei",
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
						"id": "pfwfhk8a",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "approver",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"cascadeDelete": false,
						"collectionId": "_pb_users_auth_",
						"hidden": false,
						"id": "lwzae5gf",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "rejector",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "8wtvhwar",
						"max": 0,
						"min": 0,
						"name": "rejection_reason",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "yzugnurw",
						"max": "",
						"min": "",
						"name": "approved",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "date"
					},
					{
						"hidden": false,
						"id": "vue3mlk0",
						"max": "",
						"min": "",
						"name": "rejected",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "date"
					},
					{
						"hidden": false,
						"id": "fjjylizi",
						"max": "",
						"min": "",
						"name": "committed",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "date"
					},
					{
						"cascadeDelete": false,
						"collectionId": "_pb_users_auth_",
						"hidden": false,
						"id": "8sig1vra",
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
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "fpri53nrr2xgoov",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_NSP4DAc` + "`" + ` ON ` + "`" + `time_sheets` + "`" + ` (\n  ` + "`" + `uid` + "`" + `,\n  ` + "`" + `week_ending` + "`" + `\n)"
				],
				"listRule": "@request.auth.id != \"\" && (\n  uid = @request.auth.id\n)",
				"name": "time_sheets",
				"system": false,
				"type": "base",
				"updateRule": "(rejected = true && rejector != \"\" && rejection_reason != \"\") || (rejected = false)",
				"viewRule": "@request.auth.id != \"\" && (\n  uid = @request.auth.id\n)"
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
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
						"id": "4hsjcwtw",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "uid",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "6of5hjva",
						"max": 40,
						"min": 8,
						"name": "work_week_hours",
						"onlyInt": false,
						"presentable": false,
						"required": true,
						"system": false,
						"type": "number"
					},
					{
						"hidden": false,
						"id": "pgwqbaui",
						"name": "salary",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "bool"
					},
					{
						"hidden": false,
						"id": "nd2tweu3",
						"max": 1000,
						"min": 50,
						"name": "default_charge_out_rate",
						"onlyInt": false,
						"presentable": false,
						"required": true,
						"system": false,
						"type": "number"
					},
					{
						"hidden": false,
						"id": "6yqnu4zu",
						"name": "off_rotation_permitted",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "bool"
					},
					{
						"hidden": false,
						"id": "fmuapxvl",
						"maxSelect": 1,
						"name": "skip_min_time_check",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "select",
						"values": [
							"no",
							"on_next_bundle",
							"yes"
						]
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "jtq5elga",
						"max": 0,
						"min": 0,
						"name": "opening_date",
						"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "gnwvxtyk",
						"max": 500,
						"min": 0,
						"name": "opening_op",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"hidden": false,
						"id": "4pjevdlg",
						"max": 500,
						"min": 0,
						"name": "opening_ov",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "d6fgkrwy",
						"max": 0,
						"min": 0,
						"name": "payroll_id",
						"pattern": "^(?:[1-9]\\d*|CMS[0-9]{1,2})$",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "zc850lb2wclrr87",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_UpEVC7E` + "`" + ` ON ` + "`" + `admin_profiles` + "`" + ` (` + "`" + `uid` + "`" + `)",
					"CREATE UNIQUE INDEX ` + "`" + `idx_XnQ4v11` + "`" + ` ON ` + "`" + `admin_profiles` + "`" + ` (` + "`" + `payroll_id` + "`" + `)"
				],
				"listRule": null,
				"name": "admin_profiles",
				"system": false,
				"type": "base",
				"updateRule": null,
				"viewRule": null
			},
			{
				"createRule": "@request.auth.id != \"\" &&\n@request.auth.time_sheets_via_approver.id ?= time_sheet",
				"deleteRule": "@request.auth.id != \"\" &&\ntime_sheet ?= @request.auth.time_sheets_via_approver.id",
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"cascadeDelete": true,
						"collectionId": "fpri53nrr2xgoov",
						"hidden": false,
						"id": "6i9fbu28",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "time_sheet",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"cascadeDelete": false,
						"collectionId": "_pb_users_auth_",
						"hidden": false,
						"id": "lelfbeex",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "reviewer",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "d5utnnkq",
						"max": "",
						"min": "",
						"name": "reviewed",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "date"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "g3surmbkacieshv",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_MVTW8sD` + "`" + ` ON ` + "`" + `time_sheet_reviewers` + "`" + ` (\n  ` + "`" + `time_sheet` + "`" + `,\n  ` + "`" + `reviewer` + "`" + `\n)"
				],
				"listRule": "@request.auth.id = @collection.users.time_sheets_via_approver.approver ||\n@request.auth.id = reviewer",
				"name": "time_sheet_reviewers",
				"system": false,
				"type": "base",
				"updateRule": null,
				"viewRule": "@request.auth.id = @collection.users.time_sheets_via_approver.approver ||\n@request.auth.id = reviewer"
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "kxtzlmig",
						"max": 0,
						"min": 0,
						"name": "date",
						"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
						"presentable": true,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "c9b90wqyjpqa7tk",
				"indexes": [],
				"listRule": null,
				"name": "payroll_year_end_dates",
				"system": false,
				"type": "base",
				"updateRule": null,
				"viewRule": null
			},
			{
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
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text1579384326",
						"max": 0,
						"min": 0,
						"name": "name",
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
						"id": "_clone_uIbF",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "manager_uid",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4196672953",
						"max": 0,
						"min": 0,
						"name": "manager",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "_clone_YocW",
						"max": 0,
						"min": 0,
						"name": "opening_date",
						"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "_clone_wdx4",
						"max": 500,
						"min": 0,
						"name": "opening_ov",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"hidden": false,
						"id": "_clone_dI6X",
						"max": 500,
						"min": 0,
						"name": "opening_op",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"hidden": false,
						"id": "number2013203736",
						"max": null,
						"min": null,
						"name": "used_ov",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"hidden": false,
						"id": "number2661066797",
						"max": null,
						"min": null,
						"name": "used_op",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"hidden": false,
						"id": "number238981232",
						"max": null,
						"min": null,
						"name": "timesheet_ov",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"hidden": false,
						"id": "number3881645381",
						"max": null,
						"min": null,
						"name": "timesheet_op",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text1948140072",
						"max": 0,
						"min": 0,
						"name": "last_ov",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text2642274077",
						"max": 0,
						"min": 0,
						"name": "last_op",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					}
				],
				"id": "6z8rcof9bkpzz1t",
				"indexes": [],
				"listRule": "@request.auth.id = id ||\n@request.auth.id = manager_uid",
				"name": "time_off",
				"system": false,
				"type": "view",
				"updateRule": null,
				"viewQuery": "SELECT \n    p.uid as id,\n    CAST(CONCAT(p.surname, ', ', p.given_name) AS TEXT) AS name,\n    p.manager AS manager_uid,\n    CAST(CONCAT(mp.surname, ', ', mp.given_name) AS TEXT) AS manager,\n    ap.opening_date,\n    ap.opening_ov,\n    ap.opening_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_op,\n    CAST(MAX(CASE WHEN tt.code = 'OV' THEN te.date END) AS TEXT) AS last_ov,\n    CAST(MAX(CASE WHEN tt.code = 'OP' THEN te.date END) AS TEXT) AS last_op\nFROM \n    profiles p\nLEFT JOIN \n    admin_profiles ap ON p.uid = ap.uid\nLEFT JOIN \n    profiles mp ON p.manager = mp.uid\nLEFT JOIN \n    time_entries te ON p.uid = te.uid\nLEFT JOIN \n    time_types tt ON te.time_type = tt.id\nWHERE \n    te.week_ending >= ap.opening_date\nGROUP BY \n    p.uid, p.surname, p.given_name, p.manager, mp.surname, mp.given_name, ap.opening_date, ap.opening_ov, ap.opening_op",
				"viewRule": "@request.auth.id = id ||\n@request.auth.id = manager_uid"
			},
			{
				"createRule": "// the caller is authenticated\n@request.auth.id != \"\" &&\n\n// no po_number is submitted\n(@request.body.po_number:isset = false || @request.body.po_number = \"\") &&\n\n// status is Unapproved\n@request.body.status = \"Unapproved\" &&\n\n// the uid is missing or is equal to the authenticated user's id\n(@request.body.uid:isset = false || @request.body.uid = @request.auth.id) &&\n\n// no rejection properties are submitted\n@request.body.rejector:isset = false &&\n@request.body.rejected:isset = false &&\n@request.body.rejection_reason:isset = false &&\n\n// approved isn't set and approver has the right claim. Test divisions in payload in hooks\n@request.body.approved:isset = false &&\n@request.body.approver.user_claims_via_uid.cid.name ?= 'po_approver' &&\n\n// no second approver properties are submitted\n@request.body.second_approver:isset = false &&\n@request.body.second_approval:isset = false &&\n@request.body.second_approver_claim:isset = false &&\n\n// no cancellation properties are submitted\n@request.body.cancelled:isset = false &&\n@request.body.canceller:isset = false &&\n\n// vendor is active\n@request.body.vendor.status = \"Active\" &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // compare the new category to the new job\n  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||\n  @request.body.category = \"\"\n)",
				"deleteRule": "@request.auth.id = uid && status = 'Unapproved'",
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "tjcbf5e3",
						"max": 0,
						"min": 0,
						"name": "po_number",
						"pattern": "^(20[2-9]\\d)-(0{3}[1-9]|0{2}[1-9]\\d|0[1-9]\\d{2}|[1-3]\\d{3}|4[0-8]\\d{2}|49[0-9]{2})(?:-(0[1-9]|[1-9]\\d))?$",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "od79ozm1",
						"maxSelect": 1,
						"name": "status",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "select",
						"values": [
							"Unapproved",
							"Active",
							"Cancelled",
							"Closed"
						]
					},
					{
						"cascadeDelete": false,
						"collectionId": "_pb_users_auth_",
						"hidden": false,
						"id": "l0bykiha",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "uid",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "wwwtd51w",
						"maxSelect": 1,
						"name": "type",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "select",
						"values": [
							"Normal",
							"Cumulative",
							"Recurring"
						]
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "4c4auzt9",
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
						"autogeneratePattern": "",
						"hidden": false,
						"id": "hqtvqmtx",
						"max": 0,
						"min": 0,
						"name": "end_date",
						"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "65m4tbko",
						"maxSelect": 1,
						"name": "frequency",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "select",
						"values": [
							"Weekly",
							"Biweekly",
							"Monthly"
						]
					},
					{
						"cascadeDelete": false,
						"collectionId": "3esdddggow6dykr",
						"hidden": false,
						"id": "nfuhmtlf",
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
						"id": "6uz2s2c6",
						"max": 0,
						"min": 5,
						"name": "description",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "azgktu8n",
						"max": null,
						"min": 0,
						"name": "total",
						"onlyInt": false,
						"presentable": false,
						"required": true,
						"system": false,
						"type": "number"
					},
					{
						"hidden": false,
						"id": "qakahtme",
						"maxSelect": 1,
						"name": "payment_type",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "select",
						"values": [
							"OnAccount",
							"Expense",
							"CorporateCreditCard"
						]
					},
					{
						"hidden": false,
						"id": "0clolnui",
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
						"id": "5rekg0iz",
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
						"id": "qj3tjhw6",
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
						"id": "war1qt5e",
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
						"id": "xiadfk0k",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "approver",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "kmdaym5e",
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
						"collectionId": "l0tpyvfnr1inncv",
						"hidden": false,
						"id": "elntbwwz",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "second_approver_claim",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"cascadeDelete": false,
						"collectionId": "_pb_users_auth_",
						"hidden": false,
						"id": "wwnnme9m",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "second_approver",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "j3v3g8vs",
						"max": "",
						"min": "",
						"name": "second_approval",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "date"
					},
					{
						"cascadeDelete": false,
						"collectionId": "_pb_users_auth_",
						"hidden": false,
						"id": "4tjxswnx",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "canceller",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "lm1hbt7h",
						"max": "",
						"min": "",
						"name": "cancelled",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "date"
					},
					{
						"cascadeDelete": false,
						"collectionId": "yovqzrnnomp0lkx",
						"hidden": false,
						"id": "fzmkxved",
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
						"id": "mzwtgxtc",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "category",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"cascadeDelete": false,
						"collectionId": "y0xvnesailac971",
						"hidden": false,
						"id": "kbqsgaiq",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "vendor",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"cascadeDelete": false,
						"collectionId": "m19q72syy0e3lvm",
						"hidden": false,
						"id": "lfdyy6et",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "parent_po",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "m19q72syy0e3lvm",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_6Ao8pCT` + "`" + ` ON ` + "`" + `purchase_orders` + "`" + ` (` + "`" + `po_number` + "`" + `) WHERE ` + "`" + `po_number` + "`" + ` != ''"
				],
				"listRule": "@request.auth.id = uid ||\n@request.auth.id = approver ||\n@request.auth.id = second_approver",
				"name": "purchase_orders",
				"system": false,
				"type": "base",
				"updateRule": "// only the creator can update the record\nuid = @request.auth.id &&\n\n// status is Unapproved and no approvals have been performed\nstatus = 'Unapproved' &&\napproved = \"\" &&\nsecond_approval = \"\"\n\n// no po_number is submitted\n(@request.body.po_number:isset = false || po_number = @request.body.po_number) &&\n\n// no rejection properties are submitted\n(@request.body.rejector:isset = false || rejector = @request.body.rejector) &&\n(@request.body.rejected:isset = false || rejected = @request.body.rejected) &&\n(@request.body.rejection_reason:isset = false || rejection_reason = @request.body.rejection_reason) &&\n\n// approved isn't set and approver has the right claim. Test divisions in payload in hooks\n(@request.body.approved:isset = false || approved = @request.body.approved) &&\n@request.body.approver.user_claims_via_uid.cid.name ?= 'po_approver' &&\n\n// no second approver properties are submitted\n(@request.body.second_approver:isset = false || second_approver = @request.body.second_approver) &&\n(@request.body.second_approval:isset = false || second_approval = @request.body.second_approval) &&\n(@request.body.second_approver_claim:isset = false || second_approver_claim = @request.body.second_approver_claim) &&\n\n// no cancellation properties are submitted\n(@request.body.cancelled:isset = false || cancelled = @request.body.cancelled) &&\n(@request.body.canceller:isset = false || canceller = @request.body.canceller) &&\n\n// vendor is active\n@request.body.vendor.status = \"Active\" &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // the job is unchanged, compare the new category to job\n  ( @request.body.job:isset = false && @request.body.category.job = job ) ||\n  // the job has changed, compare the new category to the new job\n  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||\n  @request.body.category = \"\"\n)",
				"viewRule": "@request.auth.id = uid ||\n@request.auth.id = approver ||\n@request.auth.id = second_approver"
			},
			{
				"createRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job'",
				"deleteRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job' &&\n\n// prevent deletion of categories if there are referencing time_entries\n@collection.time_entries.category != id &&\n\n// prevent deletion of categories if there are referencing purchase orders\n@collection.purchase_orders.category != id &&\n\n// prevent deletion of categories if there are referencing expenses\n@collection.expenses.category != id",
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "oyndkpey",
						"max": 0,
						"min": 3,
						"name": "name",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"cascadeDelete": true,
						"collectionId": "yovqzrnnomp0lkx",
						"hidden": false,
						"id": "cedjug8b",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "job",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "nrwhbwowokwu6cr",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_SF6A76x` + "`" + ` ON ` + "`" + `categories` + "`" + ` (\n  ` + "`" + `job` + "`" + `,\n  ` + "`" + `name` + "`" + `\n)"
				],
				"listRule": "@request.auth.id != \"\"",
				"name": "categories",
				"system": false,
				"type": "base",
				"updateRule": null,
				"viewRule": "@request.auth.id != \"\""
			},
			{
				"createRule": "// the caller is authenticated\n@request.auth.id != \"\" &&\n\n// the uid is equal to the authenticated user's id\n@request.body.uid = @request.auth.id &&\n\n// no rejection properties are submitted\n@request.body.rejector:isset = false &&\n@request.body.rejected:isset = false &&\n@request.body.rejection_reason:isset = false &&\n\n// no approval properties are submitted\n@request.body.approved:isset = false &&\n@request.body.approver:isset = false &&\n\n// no committed properties are submitted\n@request.body.committed:isset = false &&\n@request.body.committer:isset = false &&\n@request.body.committed_week_ending:isset = false &&\n\n// if present, vendor is active\n(@request.body.vendor = \"\" || @request.body.vendor.status = \"Active\") &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // compare the new category to the new job\n  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||\n  @request.body.category = \"\"\n)",
				"deleteRule": "@request.auth.id = uid && committed = \"\"",
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
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
						"id": "1pjwom6l",
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
						"id": "8suftgyi",
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
						"id": "cggnkeqm",
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
						"id": "spdshefk",
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
						"id": "st2japdo",
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
						"id": "puynywev",
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
						"autogeneratePattern": "",
						"hidden": false,
						"id": "oplty6th",
						"max": 0,
						"min": 0,
						"name": "vendor_name",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "edbixzlo",
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
						"id": "wjdoqxuu",
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
						"id": "yy4wgwrx",
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
						"id": "fpshyvya",
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
						"id": "uoh8s8ea",
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
						"id": "p19lerrm",
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
						"id": "3f4rryq3",
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
						"id": "gszhhxl6",
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
						"id": "6ocqzyet",
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
						"id": "tahxw786",
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
						"id": "cpt1x5gr",
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
						"id": "djy3zkz8",
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
						"id": "bmzx8tgn",
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
						"id": "d13a8jxo",
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
						"id": "hsvbnev9",
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
						"id": "gv2z62zj",
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
						"id": "pxd0mvyh",
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
						"id": "zbkxxgao",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "vendor",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "o1vpz1mm7qsfoyy",
				"indexes": [],
				"listRule": "uid = @request.auth.id ||\n(approver = @request.auth.id && submitted = true) ||\n(approved != \"\" && @request.auth.user_claims_via_uid.cid.name ?= 'commit') ||\n(committed != \"\" && @request.auth.user_claims_via_uid.cid.name ?= 'report')",
				"name": "expenses",
				"system": false,
				"type": "base",
				"updateRule": "// only the creator can update the record\nuid = @request.auth.id &&\n\n// the uid must not change\n(@request.body.uid:isset = false || uid = @request.body.uid) &&\n\n// no rejection properties are submitted\n(@request.body.rejector:isset = false || rejector = @request.body.rejector) &&\n(@request.body.rejected:isset = false || rejected = @request.body.rejected) &&\n(@request.body.rejection_reason:isset = false || rejection_reason = @request.body.rejection_reason) &&\n\n// submitted is not changed\n(@request.body.submitted:isset = false || submitted = @request.body.submitted) &&\n\n// no approval properties are submitted\n(@request.body.approved:isset = false || approved = @request.body.approved) &&\n(@request.body.approver:isset = false || approver = @request.body.approver) &&\n\n// no committed properties are submitted\n(@request.body.committed:isset = false || committed = @request.body.committed) &&\n(@request.body.committer:isset = false || committer = @request.body.committer) &&\n(@request.body.committed_week_ending:isset = false || committed_week_ending = @request.body.committed_week_ending) &&\n\n// if present, vendor is active\n(@request.body.vendor = \"\" || @request.body.vendor.status = \"Active\") &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // the job is unchanged, compare the new category to job\n  ( @request.body.job:isset = false && @request.body.category.job = job ) ||\n  // the job has changed, compare the new category to the new job\n  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||\n  @request.body.category = \"\"\n)",
				"viewRule": "uid = @request.auth.id ||\n(approver = @request.auth.id && submitted = true) ||\n(approved != \"\" && @request.auth.user_claims_via_uid.cid.name ?= 'commit') ||\n(committed != \"\" && @request.auth.user_claims_via_uid.cid.name ?= 'report')"
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "27qaxv2u",
						"max": 0,
						"min": 0,
						"name": "effective_date",
						"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "nwllwvdz",
						"max": null,
						"min": 0,
						"name": "breakfast",
						"onlyInt": false,
						"presentable": false,
						"required": true,
						"system": false,
						"type": "number"
					},
					{
						"hidden": false,
						"id": "iz3crqwa",
						"max": null,
						"min": 0,
						"name": "lunch",
						"onlyInt": false,
						"presentable": false,
						"required": true,
						"system": false,
						"type": "number"
					},
					{
						"hidden": false,
						"id": "uzmiw2za",
						"max": null,
						"min": 0,
						"name": "dinner",
						"onlyInt": false,
						"presentable": false,
						"required": true,
						"system": false,
						"type": "number"
					},
					{
						"hidden": false,
						"id": "drfzivwc",
						"max": null,
						"min": 0,
						"name": "lodging",
						"onlyInt": false,
						"presentable": false,
						"required": true,
						"system": false,
						"type": "number"
					},
					{
						"hidden": false,
						"id": "uf3fazuz",
						"maxSize": 2000000,
						"name": "mileage",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "json"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "kbohbd4ww45zf23",
				"indexes": [],
				"listRule": null,
				"name": "expense_rates",
				"system": false,
				"type": "base",
				"updateRule": null,
				"viewRule": null
			},
			{
				"createRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job'",
				"deleteRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job' &&\n// prevent deletion of clients if there are referencing jobs\n@collection.jobs.client != id",
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "hpftesxg",
						"max": 0,
						"min": 2,
						"name": "name",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "1v6i9rrpniuatcx",
				"indexes": [
					"CREATE INDEX ` + "`" + `idx_aXJh3FO` + "`" + ` ON ` + "`" + `clients` + "`" + ` (` + "`" + `name` + "`" + `)"
				],
				"listRule": "@request.auth.id != \"\"",
				"name": "clients",
				"system": false,
				"type": "base",
				"updateRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job'",
				"viewRule": "@request.auth.id != \"\""
			},
			{
				"createRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job'",
				"deleteRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job' &&\n// prevent deletion of contacts if there are referencing jobs\n@collection.jobs.contact != id",
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "isgvpgue",
						"max": 0,
						"min": 0,
						"name": "surname",
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
						"id": "sdagw2zd",
						"max": 0,
						"min": 0,
						"name": "given_name",
						"pattern": "",
						"presentable": true,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"exceptDomains": null,
						"hidden": false,
						"id": "hfcua49b",
						"name": "email",
						"onlyDomains": null,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "email"
					},
					{
						"cascadeDelete": true,
						"collectionId": "1v6i9rrpniuatcx",
						"hidden": false,
						"id": "w4csqqjx",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "client",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "3v7wxidd2f9yhf9",
				"indexes": [
					"CREATE INDEX ` + "`" + `idx_KxKk01Y` + "`" + ` ON ` + "`" + `client_contacts` + "`" + ` (\n  ` + "`" + `surname` + "`" + `,\n  ` + "`" + `given_name` + "`" + `\n)",
					"CREATE UNIQUE INDEX ` + "`" + `idx_0KoVkzQ` + "`" + ` ON ` + "`" + `client_contacts` + "`" + ` (` + "`" + `email` + "`" + `)"
				],
				"listRule": "@request.auth.id != \"\"",
				"name": "client_contacts",
				"system": false,
				"type": "base",
				"updateRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job'",
				"viewRule": "@request.auth.id != \"\""
			},
			{
				"createRule": "@request.auth.user_claims_via_uid.cid.name ?= 'tame' &&\n// no tsid is submitted, it's set in the hook\n(@request.body.tsid:isset = false || @request.body.tsid = \"\") &&\n\n// no committed properties are submitted\n@request.body.committed:isset = false &&\n@request.body.committer:isset = false &&\n@request.body.committed_week_ending:isset = false &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // compare the new category to the new job\n  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||\n  @request.body.category = \"\"\n)",
				"deleteRule": "@request.auth.user_claims_via_uid.cid.name ?= 'tame' &&\ncommitted = \"\"",
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
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
						"collectionId": "3esdddggow6dykr",
						"hidden": false,
						"id": "oszzgyip",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "division",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"cascadeDelete": false,
						"collectionId": "_pb_users_auth_",
						"hidden": false,
						"id": "2h0enwkz",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "uid",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "1esmkvan",
						"max": 18,
						"min": -18,
						"name": "hours",
						"onlyInt": false,
						"presentable": false,
						"required": true,
						"system": false,
						"type": "number"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "c4hphpdm",
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
						"cascadeDelete": false,
						"collectionId": "cnqv0wm8hly7r3n",
						"hidden": false,
						"id": "ad7feyjt",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "time_type",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "yjapnpeh",
						"max": 3,
						"min": 0,
						"name": "meals_hours",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"cascadeDelete": false,
						"collectionId": "yovqzrnnomp0lkx",
						"hidden": false,
						"id": "dgwsotxu",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "job",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "8yws3dwb",
						"max": 0,
						"min": 0,
						"name": "work_record",
						"pattern": "^[FKQ][0-9]{2}-[0-9]{3,}(-[0-9]+)?$",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "5zavlc9z",
						"max": null,
						"min": null,
						"name": "payout_request_amount",
						"onlyInt": false,
						"presentable": false,
						"required": false,
						"system": false,
						"type": "number"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "zhfe6rbd",
						"max": 0,
						"min": 0,
						"name": "date",
						"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
						"presentable": true,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "9e1umw0s",
						"max": 0,
						"min": 0,
						"name": "week_ending",
						"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
						"presentable": true,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"cascadeDelete": false,
						"collectionId": "fpri53nrr2xgoov",
						"hidden": false,
						"id": "xy68bh6o",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "tsid",
						"presentable": true,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"cascadeDelete": false,
						"collectionId": "nrwhbwowokwu6cr",
						"hidden": false,
						"id": "i1uzlmch",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "category",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"cascadeDelete": false,
						"collectionId": "_pb_users_auth_",
						"hidden": false,
						"id": "clpvzg0c",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "creator",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "relation"
					},
					{
						"hidden": false,
						"id": "cjxpxn9c",
						"max": "",
						"min": "",
						"name": "committed",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "date"
					},
					{
						"cascadeDelete": false,
						"collectionId": "_pb_users_auth_",
						"hidden": false,
						"id": "anj6odqu",
						"maxSelect": 1,
						"minSelect": 0,
						"name": "committer",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "relation"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "6xltfvly",
						"max": 0,
						"min": 0,
						"name": "committed_week_ending",
						"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
						"presentable": true,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "eobhfwdi",
						"name": "skip_tsid_check",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "bool"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "5z24r2v5jgh8qft",
				"indexes": [],
				"listRule": "@request.auth.user_claims_via_uid.cid.name ?= 'tame' ||\n@request.auth.user_claims_via_uid.cid.name ?= 'report'",
				"name": "time_amendments",
				"system": false,
				"type": "base",
				"updateRule": "@request.auth.user_claims_via_uid.cid.name ?= 'tame' &&\ncommitted = \"\" &&\n// no tsid is submitted, it's set in the hook\n(@request.body.tsid:isset = false || tsid = @request.body.tsid) &&\n\n// no committed properties are submitted\n(@request.body.committed:isset = false || committed = @request.body.committed) &&\n(@request.body.committer:isset = false || committer = @request.body.committer) &&\n(@request.body.committed_week_ending:isset = false || committed_week_ending = @request.body.committed_week_ending) &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // the job is unchanged, compare the new category to job\n  ( @request.body.job:isset = false && @request.body.category.job = job ) ||\n  // the job has changed, compare the new category to the new job\n  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||\n  @request.body.category = \"\"\n)",
				"viewRule": "@request.auth.user_claims_via_uid.cid.name ?= 'tame' ||\n@request.auth.user_claims_via_uid.cid.name ?= 'report'"
			},
			{
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
						"autogeneratePattern": "",
						"hidden": false,
						"id": "_clone_54ng",
						"max": 48,
						"min": 2,
						"name": "surname",
						"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "_clone_EZkx",
						"max": 48,
						"min": 2,
						"name": "given_name",
						"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "_clone_9Tvi",
						"maxSize": 2000000,
						"name": "divisions",
						"presentable": false,
						"required": false,
						"system": false,
						"type": "json"
					}
				],
				"id": "kn6f5sfmzjogw63",
				"indexes": [],
				"listRule": "@request.auth.id != \"\"",
				"name": "po_approvers",
				"system": false,
				"type": "view",
				"updateRule": null,
				"viewQuery": "SELECT p.uid AS id, p.surname AS surname, p.given_name AS given_name, u.payload as divisions \nFROM profiles p\nINNER JOIN user_claims u ON p.uid = u.uid\nINNER JOIN claims c ON u.cid = c.id\nWHERE c.name = 'po_approver'",
				"viewRule": "@request.auth.id != \"\""
			},
			{
				"createRule": "@request.auth.user_claims_via_uid.cid.name ?= 'payables_admin'",
				"deleteRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'payables_admin' &&\n\n// prevent deletion of vendors if there are referencing purchase orders\n@collection.purchase_orders.job != id &&\n\n// prevent deletion of vendors if there are referencing expenses\n@collection.expenses.job != id",
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "so6nx9uo",
						"max": 0,
						"min": 3,
						"name": "name",
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
						"id": "sxfocdv1",
						"max": 0,
						"min": 3,
						"name": "alias",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "7lzhalcf",
						"maxSelect": 1,
						"name": "status",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "select",
						"values": [
							"Active",
							"Inactive"
						]
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "y0xvnesailac971",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_GCZxhiM` + "`" + ` ON ` + "`" + `vendors` + "`" + ` (` + "`" + `name` + "`" + `)",
					"CREATE UNIQUE INDEX ` + "`" + `idx_c8OTvkU` + "`" + ` ON ` + "`" + `vendors` + "`" + ` (` + "`" + `alias` + "`" + `) WHERE ` + "`" + `alias` + "`" + ` != ''"
				],
				"listRule": "@request.auth.id != \"\"",
				"name": "vendors",
				"system": false,
				"type": "base",
				"updateRule": "@request.auth.user_claims_via_uid.cid.name ?= 'payables_admin'",
				"viewRule": "@request.auth.id != \"\""
			},
			{
				"createRule": null,
				"deleteRule": "@request.auth.user_claims_via_uid.cid.name ?= 'absorb'",
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "mm9oylkv",
						"max": 0,
						"min": 0,
						"name": "collection_name",
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
						"id": "vjvkevat",
						"max": 0,
						"min": 0,
						"name": "target_id",
						"pattern": "",
						"presentable": true,
						"primaryKey": false,
						"required": true,
						"system": false,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "zt83vc63",
						"maxSize": 2000000,
						"name": "absorbed_records",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "json"
					},
					{
						"hidden": false,
						"id": "d80tdp67",
						"maxSize": 2000000,
						"name": "updated_references",
						"presentable": false,
						"required": true,
						"system": false,
						"type": "json"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": false,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": false,
						"type": "autodate"
					}
				],
				"id": "yw3bni1ad22grdo",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_T0t8iRR` + "`" + ` ON ` + "`" + `absorb_actions` + "`" + ` (` + "`" + `collection_name` + "`" + `)"
				],
				"listRule": "@request.auth.user_claims_via_uid.cid.name ?= 'absorb'",
				"name": "absorb_actions",
				"system": false,
				"type": "base",
				"updateRule": null,
				"viewRule": "@request.auth.user_claims_via_uid.cid.name ?= 'absorb'"
			},
			{
				"authAlert": {
					"emailTemplate": {
						"body": "<p>Hello,</p>\n<p>We noticed a login to your {APP_NAME} account from a new location.</p>\n<p>If this was you, you may disregard this email.</p>\n<p><strong>If this wasn't you, you should immediately change your {APP_NAME} account password to revoke access from all other locations.</strong></p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
						"subject": "Login from a new location"
					},
					"enabled": true
				},
				"authRule": "",
				"authToken": {
					"duration": 1209600
				},
				"confirmEmailChangeTemplate": {
					"body": "<p>Hello,</p>\n<p>Click on the button below to confirm your new email address.</p>\n<p>\n  <a class=\"btn\" href=\"{APP_URL}/_/#/auth/confirm-email-change/{TOKEN}\" target=\"_blank\" rel=\"noopener\">Confirm new email</a>\n</p>\n<p><i>If you didn't ask to change your email address, you can ignore this email.</i></p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
					"subject": "Confirm your {APP_NAME} new email address"
				},
				"createRule": null,
				"deleteRule": null,
				"emailChangeToken": {
					"duration": 1800
				},
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"cost": 0,
						"hidden": true,
						"id": "password901924565",
						"max": 0,
						"min": 8,
						"name": "password",
						"pattern": "",
						"presentable": false,
						"required": true,
						"system": true,
						"type": "password"
					},
					{
						"autogeneratePattern": "[a-zA-Z0-9]{50}",
						"hidden": true,
						"id": "text2504183744",
						"max": 60,
						"min": 30,
						"name": "tokenKey",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"exceptDomains": null,
						"hidden": false,
						"id": "email3885137012",
						"name": "email",
						"onlyDomains": null,
						"presentable": false,
						"required": true,
						"system": true,
						"type": "email"
					},
					{
						"hidden": false,
						"id": "bool1547992806",
						"name": "emailVisibility",
						"presentable": false,
						"required": false,
						"system": true,
						"type": "bool"
					},
					{
						"hidden": false,
						"id": "bool256245529",
						"name": "verified",
						"presentable": false,
						"required": false,
						"system": true,
						"type": "bool"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": true,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": true,
						"type": "autodate"
					}
				],
				"fileToken": {
					"duration": 120
				},
				"id": "pbc_3142635823",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_tokenKey_pbc_3142635823` + "`" + ` ON ` + "`" + `_superusers` + "`" + ` (` + "`" + `tokenKey` + "`" + `)",
					"CREATE UNIQUE INDEX ` + "`" + `idx_email_pbc_3142635823` + "`" + ` ON ` + "`" + `_superusers` + "`" + ` (` + "`" + `email` + "`" + `) WHERE ` + "`" + `email` + "`" + ` != ''"
				],
				"listRule": null,
				"manageRule": null,
				"mfa": {
					"duration": 1800,
					"enabled": false,
					"rule": ""
				},
				"name": "_superusers",
				"oauth2": {
					"enabled": false,
					"mappedFields": {
						"avatarURL": "",
						"id": "",
						"name": "",
						"username": ""
					}
				},
				"otp": {
					"duration": 180,
					"emailTemplate": {
						"body": "<p>Hello,</p>\n<p>Your one-time password is: <strong>{OTP}</strong></p>\n<p><i>If you didn't ask for the one-time password, you can ignore this email.</i></p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
						"subject": "OTP for {APP_NAME}"
					},
					"enabled": false,
					"length": 8
				},
				"passwordAuth": {
					"enabled": true,
					"identityFields": [
						"email"
					]
				},
				"passwordResetToken": {
					"duration": 1800
				},
				"resetPasswordTemplate": {
					"body": "<p>Hello,</p>\n<p>Click on the button below to reset your password.</p>\n<p>\n  <a class=\"btn\" href=\"{APP_URL}/_/#/auth/confirm-password-reset/{TOKEN}\" target=\"_blank\" rel=\"noopener\">Reset password</a>\n</p>\n<p><i>If you didn't ask to reset your password, you can ignore this email.</i></p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
					"subject": "Reset your {APP_NAME} password"
				},
				"system": true,
				"type": "auth",
				"updateRule": null,
				"verificationTemplate": {
					"body": "<p>Hello,</p>\n<p>Thank you for joining us at {APP_NAME}.</p>\n<p>Click on the button below to verify your email address.</p>\n<p>\n  <a class=\"btn\" href=\"{APP_URL}/_/#/auth/confirm-verification/{TOKEN}\" target=\"_blank\" rel=\"noopener\">Verify</a>\n</p>\n<p>\n  Thanks,<br/>\n  {APP_NAME} team\n</p>",
					"subject": "Verify your {APP_NAME} email"
				},
				"verificationToken": {
					"duration": 259200
				},
				"viewRule": null
			},
			{
				"createRule": null,
				"deleteRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId",
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text455797646",
						"max": 0,
						"min": 0,
						"name": "collectionRef",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text127846527",
						"max": 0,
						"min": 0,
						"name": "recordRef",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text2462348188",
						"max": 0,
						"min": 0,
						"name": "provider",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text1044722854",
						"max": 0,
						"min": 0,
						"name": "providerId",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": true,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": true,
						"type": "autodate"
					}
				],
				"id": "pbc_2281828961",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_externalAuths_record_provider` + "`" + ` ON ` + "`" + `_externalAuths` + "`" + ` (collectionRef, recordRef, provider)",
					"CREATE UNIQUE INDEX ` + "`" + `idx_externalAuths_collection_provider` + "`" + ` ON ` + "`" + `_externalAuths` + "`" + ` (collectionRef, provider, providerId)"
				],
				"listRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId",
				"name": "_externalAuths",
				"system": true,
				"type": "base",
				"updateRule": null,
				"viewRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId"
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text455797646",
						"max": 0,
						"min": 0,
						"name": "collectionRef",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text127846527",
						"max": 0,
						"min": 0,
						"name": "recordRef",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text1582905952",
						"max": 0,
						"min": 0,
						"name": "method",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": true,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": true,
						"type": "autodate"
					}
				],
				"id": "pbc_2279338944",
				"indexes": [
					"CREATE INDEX ` + "`" + `idx_mfas_collectionRef_recordRef` + "`" + ` ON ` + "`" + `_mfas` + "`" + ` (collectionRef,recordRef)"
				],
				"listRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId",
				"name": "_mfas",
				"system": true,
				"type": "base",
				"updateRule": null,
				"viewRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId"
			},
			{
				"createRule": null,
				"deleteRule": null,
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text455797646",
						"max": 0,
						"min": 0,
						"name": "collectionRef",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text127846527",
						"max": 0,
						"min": 0,
						"name": "recordRef",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"cost": 8,
						"hidden": true,
						"id": "password901924565",
						"max": 0,
						"min": 0,
						"name": "password",
						"pattern": "",
						"presentable": false,
						"required": true,
						"system": true,
						"type": "password"
					},
					{
						"autogeneratePattern": "",
						"hidden": true,
						"id": "text3866985172",
						"max": 0,
						"min": 0,
						"name": "sentTo",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": false,
						"system": true,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": true,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": true,
						"type": "autodate"
					}
				],
				"id": "pbc_1638494021",
				"indexes": [
					"CREATE INDEX ` + "`" + `idx_otps_collectionRef_recordRef` + "`" + ` ON ` + "`" + `_otps` + "`" + ` (collectionRef, recordRef)"
				],
				"listRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId",
				"name": "_otps",
				"system": true,
				"type": "base",
				"updateRule": null,
				"viewRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId"
			},
			{
				"createRule": null,
				"deleteRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId",
				"fields": [
					{
						"autogeneratePattern": "[a-z0-9]{15}",
						"hidden": false,
						"id": "text3208210256",
						"max": 15,
						"min": 15,
						"name": "id",
						"pattern": "^[a-z0-9]+$",
						"presentable": false,
						"primaryKey": true,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text455797646",
						"max": 0,
						"min": 0,
						"name": "collectionRef",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text127846527",
						"max": 0,
						"min": 0,
						"name": "recordRef",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"autogeneratePattern": "",
						"hidden": false,
						"id": "text4228609354",
						"max": 0,
						"min": 0,
						"name": "fingerprint",
						"pattern": "",
						"presentable": false,
						"primaryKey": false,
						"required": true,
						"system": true,
						"type": "text"
					},
					{
						"hidden": false,
						"id": "autodate2990389176",
						"name": "created",
						"onCreate": true,
						"onUpdate": false,
						"presentable": false,
						"system": true,
						"type": "autodate"
					},
					{
						"hidden": false,
						"id": "autodate3332085495",
						"name": "updated",
						"onCreate": true,
						"onUpdate": true,
						"presentable": false,
						"system": true,
						"type": "autodate"
					}
				],
				"id": "pbc_4275539003",
				"indexes": [
					"CREATE UNIQUE INDEX ` + "`" + `idx_authOrigins_unique_pairs` + "`" + ` ON ` + "`" + `_authOrigins` + "`" + ` (collectionRef, recordRef, fingerprint)"
				],
				"listRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId",
				"name": "_authOrigins",
				"system": true,
				"type": "base",
				"updateRule": null,
				"viewRule": "@request.auth.id != '' && recordRef = @request.auth.id && collectionRef = @request.auth.collectionId"
			}
		]`

		return app.ImportCollectionsByMarshaledJSON([]byte(jsonData), false)
	}, func(app core.App) error {
		return nil
	})
}
