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

		// update
		edit_rejection_reason := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "8wtvhwar",
			"name": "rejection_reason",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), edit_rejection_reason); err != nil {
			return err
		}
		collection.Schema.AddField(edit_rejection_reason)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("fpri53nrr2xgoov")
		if err != nil {
			return err
		}

		// update
		edit_rejection_reason := &schema.SchemaField{}
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
		}`), edit_rejection_reason); err != nil {
			return err
		}
		collection.Schema.AddField(edit_rejection_reason)

		return dao.SaveCollection(collection)
	})
}
