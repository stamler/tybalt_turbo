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

		// remove
		collection.Schema.RemoveField("7jfhbubn")

		// add
		new_approved := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "yzugnurw",
			"name": "approved",
			"type": "date",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": "",
				"max": ""
			}
		}`), new_approved); err != nil {
			return err
		}
		collection.Schema.AddField(new_approved)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("fpri53nrr2xgoov")
		if err != nil {
			return err
		}

		// add
		del_approved := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "7jfhbubn",
			"name": "approved",
			"type": "bool",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {}
		}`), del_approved); err != nil {
			return err
		}
		collection.Schema.AddField(del_approved)

		// remove
		collection.Schema.RemoveField("yzugnurw")

		return dao.SaveCollection(collection)
	})
}
