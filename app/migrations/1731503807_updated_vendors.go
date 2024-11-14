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

		collection, err := dao.FindCollectionByNameOrId("y0xvnesailac971")
		if err != nil {
			return err
		}

		// update
		edit_alias := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "sxfocdv1",
			"name": "alias",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 3,
				"max": null,
				"pattern": ""
			}
		}`), edit_alias); err != nil {
			return err
		}
		collection.Schema.AddField(edit_alias)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("y0xvnesailac971")
		if err != nil {
			return err
		}

		// update
		edit_alias := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "sxfocdv1",
			"name": "alias",
			"type": "text",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 3,
				"max": null,
				"pattern": ""
			}
		}`), edit_alias); err != nil {
			return err
		}
		collection.Schema.AddField(edit_alias)

		return dao.SaveCollection(collection)
	})
}
