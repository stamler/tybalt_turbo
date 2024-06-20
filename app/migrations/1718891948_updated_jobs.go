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

		collection, err := dao.FindCollectionByNameOrId("yovqzrnnomp0lkx")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("0fnhavsw")

		// add
		new_description := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "seuuugpd",
			"name": "description",
			"type": "text",
			"required": true,
			"presentable": true,
			"unique": false,
			"options": {
				"min": 3,
				"max": null,
				"pattern": ""
			}
		}`), new_description); err != nil {
			return err
		}
		collection.Schema.AddField(new_description)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("yovqzrnnomp0lkx")
		if err != nil {
			return err
		}

		// add
		del_name := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "0fnhavsw",
			"name": "name",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), del_name); err != nil {
			return err
		}
		collection.Schema.AddField(del_name)

		// remove
		collection.Schema.RemoveField("seuuugpd")

		return dao.SaveCollection(collection)
	})
}
