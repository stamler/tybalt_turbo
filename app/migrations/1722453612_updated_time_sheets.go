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

		collection, err := dao.FindCollectionByNameOrId("fpri53nrr2xgoov")
		if err != nil {
			return err
		}

		// add
		new_manager := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "sbjeqyrv",
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
		new_submitted := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "32m2ceei",
			"name": "submitted",
			"type": "bool",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {}
		}`), new_submitted); err != nil {
			return err
		}
		collection.Schema.AddField(new_submitted)

		// add
		new_approved := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "7jfhbubn",
			"name": "approved",
			"type": "bool",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {}
		}`), new_approved); err != nil {
			return err
		}
		collection.Schema.AddField(new_approved)

		// add
		new_locked := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "79frceqq",
			"name": "locked",
			"type": "bool",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {}
		}`), new_locked); err != nil {
			return err
		}
		collection.Schema.AddField(new_locked)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("fpri53nrr2xgoov")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("sbjeqyrv")

		// remove
		collection.Schema.RemoveField("32m2ceei")

		// remove
		collection.Schema.RemoveField("7jfhbubn")

		// remove
		collection.Schema.RemoveField("79frceqq")

		return dao.SaveCollection(collection)
	})
}
