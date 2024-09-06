package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models/schema"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("6z8rcof9bkpzz1t")
		if err != nil {
			return err
		}

		options := map[string]any{}
		if err := json.Unmarshal([]byte(`{
			"query": "SELECT \n    p.uid as id,\n    CONCAT(p.surname, ', ', p.given_name) AS name,\n    p.manager AS manager_uid,\n    CONCAT(mp.surname, ', ', mp.given_name) AS manager,\n    ap.opening_date,\n    ap.opening_ov,\n    ap.opening_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_op,\n    CAST(MAX(CASE WHEN tt.code = 'OV' THEN te.date END) AS TEXT) AS last_ov,\n    CAST(MAX(CASE WHEN tt.code = 'OP' THEN te.date END) AS TEXT) AS last_op\nFROM \n    profiles p\nLEFT JOIN \n    admin_profiles ap ON p.uid = ap.uid\nLEFT JOIN \n    profiles mp ON p.manager = mp.uid\nLEFT JOIN \n    time_entries te ON p.uid = te.uid\nLEFT JOIN \n    time_types tt ON te.time_type = tt.id\nWHERE \n    te.week_ending >= ap.opening_date\nGROUP BY \n    p.uid, p.surname, p.given_name, p.manager, mp.surname, mp.given_name, ap.opening_date, ap.opening_ov, ap.opening_op"
		}`), &options); err != nil {
			return err
		}
		collection.SetOptions(options)

		// remove
		collection.Schema.RemoveField("kpeq7fzq")

		// remove
		collection.Schema.RemoveField("5vfjm86d")

		// remove
		collection.Schema.RemoveField("cdrbg300")

		// remove
		collection.Schema.RemoveField("xcuzlfly")

		// remove
		collection.Schema.RemoveField("1obbkxyg")

		// remove
		collection.Schema.RemoveField("ndt5p4qo")

		// remove
		collection.Schema.RemoveField("zgzskevm")

		// remove
		collection.Schema.RemoveField("fktypvve")

		// remove
		collection.Schema.RemoveField("0oqye8fv")

		// remove
		collection.Schema.RemoveField("u0nwvj3l")

		// remove
		collection.Schema.RemoveField("den7vdt3")

		// remove
		collection.Schema.RemoveField("6xqx8kzq")

		// remove
		collection.Schema.RemoveField("p2r8kfwj")

		// remove
		collection.Schema.RemoveField("phicqr6x")

		// add
		new_name := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_name); err != nil {
			return err
		}
		collection.Schema.AddField(new_name)

		// add
		new_manager_uid := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_manager_uid); err != nil {
			return err
		}
		collection.Schema.AddField(new_manager_uid)

		// add
		new_manager := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_manager); err != nil {
			return err
		}
		collection.Schema.AddField(new_manager)

		// add
		new_opening_date := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_opening_date); err != nil {
			return err
		}
		collection.Schema.AddField(new_opening_date)

		// add
		new_opening_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_opening_ov); err != nil {
			return err
		}
		collection.Schema.AddField(new_opening_ov)

		// add
		new_opening_op := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_opening_op); err != nil {
			return err
		}
		collection.Schema.AddField(new_opening_op)

		// add
		new_used_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_used_ov); err != nil {
			return err
		}
		collection.Schema.AddField(new_used_ov)

		// add
		new_used_op := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_used_op); err != nil {
			return err
		}
		collection.Schema.AddField(new_used_op)

		// add
		new_timesheet_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_timesheet_ov); err != nil {
			return err
		}
		collection.Schema.AddField(new_timesheet_ov)

		// add
		new_timesheet_op := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_timesheet_op); err != nil {
			return err
		}
		collection.Schema.AddField(new_timesheet_op)

		// add
		new_last_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_last_ov); err != nil {
			return err
		}
		collection.Schema.AddField(new_last_ov)

		// add
		new_last_op := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_last_op); err != nil {
			return err
		}
		collection.Schema.AddField(new_last_op)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("6z8rcof9bkpzz1t")
		if err != nil {
			return err
		}

		options := map[string]any{}
		if err := json.Unmarshal([]byte(`{
			"query": "SELECT \n    p.uid as id,\n    p.surname,\n    p.given_name,\n    p.manager AS manager_uid,\n    mp.surname AS manager_surname,\n    mp.given_name AS manager_given_name,\n    ap.opening_date,\n    ap.opening_ov,\n    ap.opening_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_op,\n    CAST(MAX(CASE WHEN tt.code = 'OV' THEN te.date END) AS TEXT) AS last_ov,\n    CAST(MAX(CASE WHEN tt.code = 'OP' THEN te.date END) AS TEXT) AS last_op\nFROM \n    profiles p\nLEFT JOIN \n    admin_profiles ap ON p.uid = ap.uid\nLEFT JOIN \n    profiles mp ON p.manager = mp.uid\nLEFT JOIN \n    time_entries te ON p.uid = te.uid\nLEFT JOIN \n    time_types tt ON te.time_type = tt.id\nWHERE \n    te.week_ending >= ap.opening_date\nGROUP BY \n    p.uid, p.surname, p.given_name, p.manager, mp.surname, mp.given_name, ap.opening_date, ap.opening_ov, ap.opening_op"
		}`), &options); err != nil {
			return err
		}
		collection.SetOptions(options)

		// add
		del_surname := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "kpeq7fzq",
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
		}`), del_surname); err != nil {
			return err
		}
		collection.Schema.AddField(del_surname)

		// add
		del_given_name := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "5vfjm86d",
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
		}`), del_given_name); err != nil {
			return err
		}
		collection.Schema.AddField(del_given_name)

		// add
		del_manager_uid := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "cdrbg300",
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
		}`), del_manager_uid); err != nil {
			return err
		}
		collection.Schema.AddField(del_manager_uid)

		// add
		del_manager_surname := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "xcuzlfly",
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
		}`), del_manager_surname); err != nil {
			return err
		}
		collection.Schema.AddField(del_manager_surname)

		// add
		del_manager_given_name := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "1obbkxyg",
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
		}`), del_manager_given_name); err != nil {
			return err
		}
		collection.Schema.AddField(del_manager_given_name)

		// add
		del_opening_date := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "ndt5p4qo",
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
		}`), del_opening_date); err != nil {
			return err
		}
		collection.Schema.AddField(del_opening_date)

		// add
		del_opening_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "zgzskevm",
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
		}`), del_opening_ov); err != nil {
			return err
		}
		collection.Schema.AddField(del_opening_ov)

		// add
		del_opening_op := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "fktypvve",
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
		}`), del_opening_op); err != nil {
			return err
		}
		collection.Schema.AddField(del_opening_op)

		// add
		del_used_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "0oqye8fv",
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
		}`), del_used_ov); err != nil {
			return err
		}
		collection.Schema.AddField(del_used_ov)

		// add
		del_used_op := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "u0nwvj3l",
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
		}`), del_used_op); err != nil {
			return err
		}
		collection.Schema.AddField(del_used_op)

		// add
		del_timesheet_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "den7vdt3",
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
		}`), del_timesheet_ov); err != nil {
			return err
		}
		collection.Schema.AddField(del_timesheet_ov)

		// add
		del_timesheet_op := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "6xqx8kzq",
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
		}`), del_timesheet_op); err != nil {
			return err
		}
		collection.Schema.AddField(del_timesheet_op)

		// add
		del_last_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "p2r8kfwj",
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
		}`), del_last_ov); err != nil {
			return err
		}
		collection.Schema.AddField(del_last_ov)

		// add
		del_last_op := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "phicqr6x",
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
		}`), del_last_op); err != nil {
			return err
		}
		collection.Schema.AddField(del_last_op)

		// remove
		collection.Schema.RemoveField("xzme4vo9")

		// remove
		collection.Schema.RemoveField("ag18tebo")

		// remove
		collection.Schema.RemoveField("ae7nheo3")

		// remove
		collection.Schema.RemoveField("rntgtrld")

		// remove
		collection.Schema.RemoveField("txei3zb8")

		// remove
		collection.Schema.RemoveField("xjwp5y3f")

		// remove
		collection.Schema.RemoveField("fjee4nop")

		// remove
		collection.Schema.RemoveField("rrbauxhr")

		// remove
		collection.Schema.RemoveField("2ki5ozes")

		// remove
		collection.Schema.RemoveField("1mitcqxa")

		// remove
		collection.Schema.RemoveField("siti0flt")

		// remove
		collection.Schema.RemoveField("cw5lcpuq")

		return dao.SaveCollection(collection)
	})
}
