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

		collection, err := dao.FindCollectionByNameOrId("pmxhrqhngh60icm")
		if err != nil {
			return err
		}

		// add
		new_payload := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "gfyrln8y",
			"name": "payload",
			"type": "json",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSize": 2000000
			}
		}`), new_payload); err != nil {
			return err
		}
		collection.Schema.AddField(new_payload)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("pmxhrqhngh60icm")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("gfyrln8y")

		return dao.SaveCollection(collection)
	})
}
