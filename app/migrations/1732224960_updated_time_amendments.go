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

		collection, err := dao.FindCollectionByNameOrId("5z24r2v5jgh8qft")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("dbenhrit")

		// remove
		collection.Schema.RemoveField("vwe32gf1")

		// add
		new_skip_tsid_check := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "eobhfwdi",
			"name": "skip_tsid_check",
			"type": "bool",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {}
		}`), new_skip_tsid_check); err != nil {
			return err
		}
		collection.Schema.AddField(new_skip_tsid_check)

		// update
		edit_tsid := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "xy68bh6o",
			"name": "tsid",
			"type": "relation",
			"required": false,
			"presentable": true,
			"unique": false,
			"options": {
				"collectionId": "fpri53nrr2xgoov",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), edit_tsid); err != nil {
			return err
		}
		collection.Schema.AddField(edit_tsid)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("5z24r2v5jgh8qft")
		if err != nil {
			return err
		}

		// add
		del_salary := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "dbenhrit",
			"name": "salary",
			"type": "bool",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {}
		}`), del_salary); err != nil {
			return err
		}
		collection.Schema.AddField(del_salary)

		// add
		del_work_week_hours := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "vwe32gf1",
			"name": "work_week_hours",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 8,
				"max": 40,
				"noDecimal": false
			}
		}`), del_work_week_hours); err != nil {
			return err
		}
		collection.Schema.AddField(del_work_week_hours)

		// remove
		collection.Schema.RemoveField("eobhfwdi")

		// update
		edit_tsid := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "xy68bh6o",
			"name": "tsid",
			"type": "relation",
			"required": false,
			"presentable": true,
			"unique": false,
			"options": {
				"collectionId": "fpri53nrr2xgoov",
				"cascadeDelete": true,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), edit_tsid); err != nil {
			return err
		}
		collection.Schema.AddField(edit_tsid)

		return dao.SaveCollection(collection)
	})
}
