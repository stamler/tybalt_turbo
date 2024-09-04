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

		// remove
		collection.Schema.RemoveField("swykowb3")

		// remove
		collection.Schema.RemoveField("px3jfbga")

		// remove
		collection.Schema.RemoveField("wq7xn48o")

		// remove
		collection.Schema.RemoveField("g9iwzjys")

		// remove
		collection.Schema.RemoveField("bvo6osf2")

		// remove
		collection.Schema.RemoveField("ujw9qcn4")

		// remove
		collection.Schema.RemoveField("e0b1ebr0")

		// remove
		collection.Schema.RemoveField("1ckas4jx")

		// remove
		collection.Schema.RemoveField("arurscwa")

		// remove
		collection.Schema.RemoveField("1jpxmqqq")

		// remove
		collection.Schema.RemoveField("mfulwgbh")

		// remove
		collection.Schema.RemoveField("ey7pyq7u")

		// remove
		collection.Schema.RemoveField("vlomia98")

		// remove
		collection.Schema.RemoveField("trmxbhne")

		// add
		new_surname := &schema.SchemaField{}
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
		}`), new_surname); err != nil {
			return err
		}
		collection.Schema.AddField(new_surname)

		// add
		new_given_name := &schema.SchemaField{}
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
		}`), new_given_name); err != nil {
			return err
		}
		collection.Schema.AddField(new_given_name)

		// add
		new_manager_uid := &schema.SchemaField{}
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
		}`), new_manager_uid); err != nil {
			return err
		}
		collection.Schema.AddField(new_manager_uid)

		// add
		new_manager_surname := &schema.SchemaField{}
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
		}`), new_manager_surname); err != nil {
			return err
		}
		collection.Schema.AddField(new_manager_surname)

		// add
		new_manager_given_name := &schema.SchemaField{}
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
		}`), new_manager_given_name); err != nil {
			return err
		}
		collection.Schema.AddField(new_manager_given_name)

		// add
		new_opening_date := &schema.SchemaField{}
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
		}`), new_opening_date); err != nil {
			return err
		}
		collection.Schema.AddField(new_opening_date)

		// add
		new_opening_ov := &schema.SchemaField{}
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
		}`), new_opening_ov); err != nil {
			return err
		}
		collection.Schema.AddField(new_opening_ov)

		// add
		new_opening_op := &schema.SchemaField{}
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
		}`), new_opening_op); err != nil {
			return err
		}
		collection.Schema.AddField(new_opening_op)

		// add
		new_used_ov := &schema.SchemaField{}
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
		}`), new_used_ov); err != nil {
			return err
		}
		collection.Schema.AddField(new_used_ov)

		// add
		new_used_op := &schema.SchemaField{}
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
		}`), new_used_op); err != nil {
			return err
		}
		collection.Schema.AddField(new_used_op)

		// add
		new_timesheet_ov := &schema.SchemaField{}
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
		}`), new_timesheet_ov); err != nil {
			return err
		}
		collection.Schema.AddField(new_timesheet_ov)

		// add
		new_timesheet_op := &schema.SchemaField{}
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
		}`), new_timesheet_op); err != nil {
			return err
		}
		collection.Schema.AddField(new_timesheet_op)

		// add
		new_last_ov := &schema.SchemaField{}
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
		}`), new_last_ov); err != nil {
			return err
		}
		collection.Schema.AddField(new_last_ov)

		// add
		new_last_op := &schema.SchemaField{}
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

		// add
		del_surname := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_surname); err != nil {
			return err
		}
		collection.Schema.AddField(del_surname)

		// add
		del_given_name := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_given_name); err != nil {
			return err
		}
		collection.Schema.AddField(del_given_name)

		// add
		del_manager_uid := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_manager_uid); err != nil {
			return err
		}
		collection.Schema.AddField(del_manager_uid)

		// add
		del_manager_surname := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_manager_surname); err != nil {
			return err
		}
		collection.Schema.AddField(del_manager_surname)

		// add
		del_manager_given_name := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_manager_given_name); err != nil {
			return err
		}
		collection.Schema.AddField(del_manager_given_name)

		// add
		del_opening_date := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_opening_date); err != nil {
			return err
		}
		collection.Schema.AddField(del_opening_date)

		// add
		del_opening_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_opening_ov); err != nil {
			return err
		}
		collection.Schema.AddField(del_opening_ov)

		// add
		del_opening_op := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_opening_op); err != nil {
			return err
		}
		collection.Schema.AddField(del_opening_op)

		// add
		del_used_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_used_ov); err != nil {
			return err
		}
		collection.Schema.AddField(del_used_ov)

		// add
		del_used_op := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_used_op); err != nil {
			return err
		}
		collection.Schema.AddField(del_used_op)

		// add
		del_timesheet_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_timesheet_ov); err != nil {
			return err
		}
		collection.Schema.AddField(del_timesheet_ov)

		// add
		del_timesheet_op := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_timesheet_op); err != nil {
			return err
		}
		collection.Schema.AddField(del_timesheet_op)

		// add
		del_last_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_last_ov); err != nil {
			return err
		}
		collection.Schema.AddField(del_last_ov)

		// add
		del_last_op := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_last_op); err != nil {
			return err
		}
		collection.Schema.AddField(del_last_op)

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

		return dao.SaveCollection(collection)
	})
}
