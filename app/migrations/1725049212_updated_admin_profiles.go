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

		collection, err := dao.FindCollectionByNameOrId("zc850lb2wclrr87")
		if err != nil {
			return err
		}

		// add
		new_opening_date := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_opening_date); err != nil {
			return err
		}
		collection.Schema.AddField(new_opening_date)

		// add
		new_opening_op := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_opening_op); err != nil {
			return err
		}
		collection.Schema.AddField(new_opening_op)

		// add
		new_opening_ov := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
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
		}`), new_opening_ov); err != nil {
			return err
		}
		collection.Schema.AddField(new_opening_ov)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("zc850lb2wclrr87")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("jtq5elga")

		// remove
		collection.Schema.RemoveField("gnwvxtyk")

		// remove
		collection.Schema.RemoveField("4pjevdlg")

		return dao.SaveCollection(collection)
	})
}
