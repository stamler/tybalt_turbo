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

		collection, err := dao.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		// add
		new_vendor := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "kbqsgaiq",
			"name": "vendor",
			"type": "relation",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "y0xvnesailac971",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), new_vendor); err != nil {
			return err
		}
		collection.Schema.AddField(new_vendor)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("kbqsgaiq")

		return dao.SaveCollection(collection)
	})
}
