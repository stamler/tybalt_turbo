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

		collection, err := dao.FindCollectionByNameOrId("3v7wxidd2f9yhf9")
		if err != nil {
			return err
		}

		// update
		edit_surname := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "isgvpgue",
			"name": "surname",
			"type": "text",
			"required": true,
			"presentable": true,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), edit_surname); err != nil {
			return err
		}
		collection.Schema.AddField(edit_surname)

		// update
		edit_given_name := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "sdagw2zd",
			"name": "given_name",
			"type": "text",
			"required": true,
			"presentable": true,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), edit_given_name); err != nil {
			return err
		}
		collection.Schema.AddField(edit_given_name)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("3v7wxidd2f9yhf9")
		if err != nil {
			return err
		}

		// update
		edit_surname := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "isgvpgue",
			"name": "surname",
			"type": "text",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), edit_surname); err != nil {
			return err
		}
		collection.Schema.AddField(edit_surname)

		// update
		edit_given_name := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "sdagw2zd",
			"name": "given_name",
			"type": "text",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": ""
			}
		}`), edit_given_name); err != nil {
			return err
		}
		collection.Schema.AddField(edit_given_name)

		return dao.SaveCollection(collection)
	})
}
