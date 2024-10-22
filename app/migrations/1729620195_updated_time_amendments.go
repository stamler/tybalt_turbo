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

		// add
		new_creator := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "clpvzg0c",
			"name": "creator",
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
		}`), new_creator); err != nil {
			return err
		}
		collection.Schema.AddField(new_creator)

		// add
		new_committed := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "cjxpxn9c",
			"name": "committed",
			"type": "date",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": "",
				"max": ""
			}
		}`), new_committed); err != nil {
			return err
		}
		collection.Schema.AddField(new_committed)

		// add
		new_committer := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "anj6odqu",
			"name": "committer",
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
		}`), new_committer); err != nil {
			return err
		}
		collection.Schema.AddField(new_committer)

		// add
		new_salary := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "dbenhrit",
			"name": "salary",
			"type": "bool",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {}
		}`), new_salary); err != nil {
			return err
		}
		collection.Schema.AddField(new_salary)

		// add
		new_work_week_hours := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "vwe32gf1",
			"name": "work_week_hours",
			"type": "number",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 8,
				"max": 40,
				"noDecimal": false
			}
		}`), new_work_week_hours); err != nil {
			return err
		}
		collection.Schema.AddField(new_work_week_hours)

		// add
		new_committed_week_ending := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "6xltfvly",
			"name": "committed_week_ending",
			"type": "text",
			"required": true,
			"presentable": true,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": "^\\d{4}-\\d{2}-\\d{2}$"
			}
		}`), new_committed_week_ending); err != nil {
			return err
		}
		collection.Schema.AddField(new_committed_week_ending)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("5z24r2v5jgh8qft")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("clpvzg0c")

		// remove
		collection.Schema.RemoveField("cjxpxn9c")

		// remove
		collection.Schema.RemoveField("anj6odqu")

		// remove
		collection.Schema.RemoveField("dbenhrit")

		// remove
		collection.Schema.RemoveField("vwe32gf1")

		// remove
		collection.Schema.RemoveField("6xltfvly")

		return dao.SaveCollection(collection)
	})
}
