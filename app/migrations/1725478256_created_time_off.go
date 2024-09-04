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
			"id": "6z8rcof9bkpzz1t",
			"created": "2024-09-04 19:30:56.139Z",
			"updated": "2024-09-04 19:30:56.139Z",
			"name": "time_off",
			"type": "view",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "swykowb3",
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
					"id": "px3jfbga",
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
					"id": "wq7xn48o",
					"name": "manager_uid",
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
					"id": "g9iwzjys",
					"name": "manager_surname",
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
					"id": "bvo6osf2",
					"name": "manager_given_name",
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
					"id": "ujw9qcn4",
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
					"id": "e0b1ebr0",
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
					"id": "1ckas4jx",
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
					"id": "arurscwa",
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
					"id": "1jpxmqqq",
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
					"id": "mfulwgbh",
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
					"id": "ey7pyq7u",
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
					"id": "vlomia98",
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
					"id": "trmxbhne",
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
				"query": "SELECT \n    p.uid as id,\n    p.surname,\n    p.given_name,\n    p.manager AS manager_uid,\n    mp.surname AS manager_surname,\n    mp.given_name AS manager_given_name,\n    ap.opening_date,\n    ap.opening_ov,\n    ap.opening_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_op,\n    CAST(MAX(CASE WHEN tt.code = 'OV' THEN te.date END) AS TEXT) AS last_ov,\n    CAST(MAX(CASE WHEN tt.code = 'OP' THEN te.date END) AS TEXT) AS last_op\nFROM \n    profiles p\nLEFT JOIN \n    admin_profiles ap ON p.uid = ap.uid\nLEFT JOIN \n    profiles mp ON p.manager = mp.uid\nLEFT JOIN \n    time_entries te ON p.uid = te.uid\nLEFT JOIN \n    time_types tt ON te.time_type = tt.id\nWHERE \n    te.week_ending >= ap.opening_date\nGROUP BY \n    p.uid, p.surname, p.given_name, p.manager, mp.surname, mp.given_name, ap.opening_date, ap.opening_ov, ap.opening_op"
			}
		}`

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("6z8rcof9bkpzz1t")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
