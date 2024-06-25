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

		collection, err := dao.FindCollectionByNameOrId("glmf9xpnwgpwudm")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("ocxmutn0")

		// remove
		collection.Schema.RemoveField("mghymcxc")

		// remove
		collection.Schema.RemoveField("suw6v59k")

		// remove
		collection.Schema.RemoveField("v6thasef")

		// remove
		collection.Schema.RemoveField("9qfzn9ab")

		// remove
		collection.Schema.RemoveField("65fcj2kd")

		// add
		new_manager := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "gudkt7qq",
			"name": "manager",
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
		}`), new_manager); err != nil {
			return err
		}
		collection.Schema.AddField(new_manager)

		// add
		new_alternate_manager := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_alternate_manager); err != nil {
			return err
		}
		collection.Schema.AddField(new_alternate_manager)

		// add
		new_default_division := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_default_division); err != nil {
			return err
		}
		collection.Schema.AddField(new_default_division)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("glmf9xpnwgpwudm")
		if err != nil {
			return err
		}

		// add
		del_opening_datetime_off := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_opening_datetime_off); err != nil {
			return err
		}
		collection.Schema.AddField(del_opening_datetime_off)

		// add
		del_opening_op := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_opening_op); err != nil {
			return err
		}
		collection.Schema.AddField(del_opening_op)

		// add
		del_opening_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_opening_ov); err != nil {
			return err
		}
		collection.Schema.AddField(del_opening_ov)

		// add
		del_untracked_time_off := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "v6thasef",
			"name": "untracked_time_off",
			"type": "bool",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {}
		}`), del_untracked_time_off); err != nil {
			return err
		}
		collection.Schema.AddField(del_untracked_time_off)

		// add
		del_timestamp := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_timestamp); err != nil {
			return err
		}
		collection.Schema.AddField(del_timestamp)

		// add
		del_default_charge_out_rate := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_default_charge_out_rate); err != nil {
			return err
		}
		collection.Schema.AddField(del_default_charge_out_rate)

		// remove
		collection.Schema.RemoveField("gudkt7qq")

		// remove
		collection.Schema.RemoveField("rwknt5er")

		// remove
		collection.Schema.RemoveField("naf0546m")

		return dao.SaveCollection(collection)
	})
}
