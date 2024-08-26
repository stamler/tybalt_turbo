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
		new_rejected := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "ovtg2rsw",
			"name": "rejected",
			"type": "bool",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {}
		}`), new_rejected); err != nil {
			return err
		}
		collection.Schema.AddField(new_rejected)

		// add
		new_rejector := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "lwzae5gf",
			"name": "rejector",
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
		}`), new_rejector); err != nil {
			return err
		}
		collection.Schema.AddField(new_rejector)

		// add
		new_rejection_reason := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "8wtvhwar",
			"name": "rejection_reason",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 5,
				"max": null,
				"pattern": ""
			}
		}`), new_rejection_reason); err != nil {
			return err
		}
		collection.Schema.AddField(new_rejection_reason)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("fpri53nrr2xgoov")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("ovtg2rsw")

		// remove
		collection.Schema.RemoveField("lwzae5gf")

		// remove
		collection.Schema.RemoveField("8wtvhwar")

		return dao.SaveCollection(collection)
	})
}
