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
			"query": "SELECT \n    p.uid as id,\n    CAST(CONCAT(p.surname, ', ', p.given_name) AS TEXT) AS name,\n    p.manager AS manager_uid,\n    CAST(CONCAT(mp.surname, ', ', mp.given_name) AS TEXT) AS manager,\n    ap.opening_date,\n    ap.opening_ov,\n    ap.opening_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_op,\n    CAST(MAX(CASE WHEN tt.code = 'OV' THEN te.date END) AS TEXT) AS last_ov,\n    CAST(MAX(CASE WHEN tt.code = 'OP' THEN te.date END) AS TEXT) AS last_op\nFROM \n    profiles p\nLEFT JOIN \n    admin_profiles ap ON p.uid = ap.uid\nLEFT JOIN \n    profiles mp ON p.manager = mp.uid\nLEFT JOIN \n    time_entries te ON p.uid = te.uid\nLEFT JOIN \n    time_types tt ON te.time_type = tt.id\nWHERE \n    te.week_ending >= ap.opening_date\nGROUP BY \n    p.uid, p.surname, p.given_name, p.manager, mp.surname, mp.given_name, ap.opening_date, ap.opening_ov, ap.opening_op"
		}`), &options); err != nil {
			return err
		}
		collection.SetOptions(options)

		// remove
		collection.Schema.RemoveField("k83dypdd")

		// remove
		collection.Schema.RemoveField("t4oym4rd")

		// remove
		collection.Schema.RemoveField("pmfegyw4")

		// remove
		collection.Schema.RemoveField("z1qm6lzn")

		// remove
		collection.Schema.RemoveField("egqh1iwe")

		// remove
		collection.Schema.RemoveField("x7wvyssl")

		// remove
		collection.Schema.RemoveField("fisusuvl")

		// remove
		collection.Schema.RemoveField("ndenyzse")

		// remove
		collection.Schema.RemoveField("0gup0m0z")

		// remove
		collection.Schema.RemoveField("mfwge5ju")

		// remove
		collection.Schema.RemoveField("vqctokbw")

		// remove
		collection.Schema.RemoveField("yjqerk2g")

		// add
		new_name := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_name); err != nil {
			return err
		}
		collection.Schema.AddField(new_name)

		// add
		new_manager_uid := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_manager_uid); err != nil {
			return err
		}
		collection.Schema.AddField(new_manager_uid)

		// add
		new_manager := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_manager); err != nil {
			return err
		}
		collection.Schema.AddField(new_manager)

		// add
		new_opening_date := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_opening_date); err != nil {
			return err
		}
		collection.Schema.AddField(new_opening_date)

		// add
		new_opening_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_opening_ov); err != nil {
			return err
		}
		collection.Schema.AddField(new_opening_ov)

		// add
		new_opening_op := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_opening_op); err != nil {
			return err
		}
		collection.Schema.AddField(new_opening_op)

		// add
		new_used_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_used_ov); err != nil {
			return err
		}
		collection.Schema.AddField(new_used_ov)

		// add
		new_used_op := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_used_op); err != nil {
			return err
		}
		collection.Schema.AddField(new_used_op)

		// add
		new_timesheet_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_timesheet_ov); err != nil {
			return err
		}
		collection.Schema.AddField(new_timesheet_ov)

		// add
		new_timesheet_op := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_timesheet_op); err != nil {
			return err
		}
		collection.Schema.AddField(new_timesheet_op)

		// add
		new_last_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_last_ov); err != nil {
			return err
		}
		collection.Schema.AddField(new_last_ov)

		// add
		new_last_op := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
			"query": "SELECT \n    p.uid as id,\n    CONCAT(p.surname, ', ', p.given_name) AS name,\n    p.manager AS manager_uid,\n    CONCAT(mp.surname, ', ', mp.given_name) AS manager,\n    ap.opening_date,\n    ap.opening_ov,\n    ap.opening_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_op,\n    CAST(MAX(CASE WHEN tt.code = 'OV' THEN te.date END) AS TEXT) AS last_ov,\n    CAST(MAX(CASE WHEN tt.code = 'OP' THEN te.date END) AS TEXT) AS last_op\nFROM \n    profiles p\nLEFT JOIN \n    admin_profiles ap ON p.uid = ap.uid\nLEFT JOIN \n    profiles mp ON p.manager = mp.uid\nLEFT JOIN \n    time_entries te ON p.uid = te.uid\nLEFT JOIN \n    time_types tt ON te.time_type = tt.id\nWHERE \n    te.week_ending >= ap.opening_date\nGROUP BY \n    p.uid, p.surname, p.given_name, p.manager, mp.surname, mp.given_name, ap.opening_date, ap.opening_ov, ap.opening_op"
		}`), &options); err != nil {
			return err
		}
		collection.SetOptions(options)

		// add
		del_name := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "k83dypdd",
			"name": "name",
			"type": "json",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSize": 1
			}
		}`), del_name); err != nil {
			return err
		}
		collection.Schema.AddField(del_name)

		// add
		del_manager_uid := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "t4oym4rd",
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
		del_manager := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "pmfegyw4",
			"name": "manager",
			"type": "json",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSize": 1
			}
		}`), del_manager); err != nil {
			return err
		}
		collection.Schema.AddField(del_manager)

		// add
		del_opening_date := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "z1qm6lzn",
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
			"id": "egqh1iwe",
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
			"id": "x7wvyssl",
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
			"id": "fisusuvl",
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
			"id": "ndenyzse",
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
			"id": "0gup0m0z",
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
			"id": "mfwge5ju",
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
			"id": "vqctokbw",
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
			"id": "yjqerk2g",
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
		collection.Schema.RemoveField("fq96v0fz")

		// remove
		collection.Schema.RemoveField("giuljwev")

		// remove
		collection.Schema.RemoveField("cnpa5nvi")

		// remove
		collection.Schema.RemoveField("jtjbf7bk")

		// remove
		collection.Schema.RemoveField("t5xcprup")

		// remove
		collection.Schema.RemoveField("g9p5inga")

		// remove
		collection.Schema.RemoveField("f0qt4hou")

		// remove
		collection.Schema.RemoveField("r59xqo1f")

		// remove
		collection.Schema.RemoveField("fxhe4blz")

		// remove
		collection.Schema.RemoveField("ien41elp")

		// remove
		collection.Schema.RemoveField("hmdwqze0")

		// remove
		collection.Schema.RemoveField("3cc7wqgx")

		return dao.SaveCollection(collection)
	})
}
