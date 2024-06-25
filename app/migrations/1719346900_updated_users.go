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

		collection, err := dao.FindCollectionByNameOrId("_pb_users_auth_")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("users_avatar")

		// add
		new_opening_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "bhlukmt6",
			"name": "opening_ov",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
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
			"id": "r5ztf8bw",
			"name": "opening_op",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"noDecimal": false
			}
		}`), new_opening_op); err != nil {
			return err
		}
		collection.Schema.AddField(new_opening_op)

		// add
		new_opening_date := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "fdbckmbl",
			"name": "opening_date",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), new_opening_date); err != nil {
			return err
		}
		collection.Schema.AddField(new_opening_date)

		// add
		new_untracked_time_off := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "0w0eomtf",
			"name": "untracked_time_off",
			"type": "bool",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {}
		}`), new_untracked_time_off); err != nil {
			return err
		}
		collection.Schema.AddField(new_untracked_time_off)

		// add
		new_default_charge_out_rate := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "c8r5y8bx",
			"name": "default_charge_out_rate",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"noDecimal": false
			}
		}`), new_default_charge_out_rate); err != nil {
			return err
		}
		collection.Schema.AddField(new_default_charge_out_rate)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("_pb_users_auth_")
		if err != nil {
			return err
		}

		// add
		del_avatar := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), del_avatar); err != nil {
			return err
		}
		collection.Schema.AddField(del_avatar)

		// remove
		collection.Schema.RemoveField("bhlukmt6")

		// remove
		collection.Schema.RemoveField("r5ztf8bw")

		// remove
		collection.Schema.RemoveField("fdbckmbl")

		// remove
		collection.Schema.RemoveField("0w0eomtf")

		// remove
		collection.Schema.RemoveField("c8r5y8bx")

		return dao.SaveCollection(collection)
	})
}
