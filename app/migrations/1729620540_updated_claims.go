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

		collection, err := dao.FindCollectionByNameOrId("l0tpyvfnr1inncv")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("eilof972")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("l0tpyvfnr1inncv")
		if err != nil {
			return err
		}

		// add
		del_payload := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "eilof972",
			"name": "payload",
			"type": "json",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSize": 2000000
			}
		}`), del_payload); err != nil {
			return err
		}
		collection.Schema.AddField(del_payload)

		return dao.SaveCollection(collection)
	})
}
