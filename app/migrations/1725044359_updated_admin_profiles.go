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

		collection, err := dao.FindCollectionByNameOrId("zc850lb2wclrr87")
		if err != nil {
			return err
		}

		// add
		new_skip_min_time_check := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "fmuapxvl",
			"name": "skip_min_time_check",
			"type": "select",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSelect": 1,
				"values": [
					"no",
					"on_next_bundle",
					"yes"
				]
			}
		}`), new_skip_min_time_check); err != nil {
			return err
		}
		collection.Schema.AddField(new_skip_min_time_check)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("zc850lb2wclrr87")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("fmuapxvl")

		return dao.SaveCollection(collection)
	})
}
