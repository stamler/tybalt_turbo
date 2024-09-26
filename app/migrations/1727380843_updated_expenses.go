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

		collection, err := dao.FindCollectionByNameOrId("o1vpz1mm7qsfoyy")
		if err != nil {
			return err
		}

		// update
		edit_allowance_types := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "tahxw786",
			"name": "allowance_types",
			"type": "select",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSelect": 4,
				"values": [
					"Lodging",
					"Breakfast",
					"Lunch",
					"Dinner"
				]
			}
		}`), edit_allowance_types); err != nil {
			return err
		}
		collection.Schema.AddField(edit_allowance_types)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("o1vpz1mm7qsfoyy")
		if err != nil {
			return err
		}

		// update
		edit_allowance_types := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "tahxw786",
			"name": "allowance_types",
			"type": "select",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"maxSelect": 2,
				"values": [
					"Lodging",
					"Breakfast",
					"Lunch",
					"Dinner"
				]
			}
		}`), edit_allowance_types); err != nil {
			return err
		}
		collection.Schema.AddField(edit_allowance_types)

		return dao.SaveCollection(collection)
	})
}
