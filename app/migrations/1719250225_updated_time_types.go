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

		collection, err := dao.FindCollectionByNameOrId("cnqv0wm8hly7r3n")
		if err != nil {
			return err
		}

		// add
		new_required_fields := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "onuwxebx",
			"name": "required_fields",
			"type": "json",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSize": 2000000
			}
		}`), new_required_fields); err != nil {
			return err
		}
		collection.Schema.AddField(new_required_fields)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("cnqv0wm8hly7r3n")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("onuwxebx")

		return dao.SaveCollection(collection)
	})
}