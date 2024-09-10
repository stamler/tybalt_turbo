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

		// update
		edit_po_number := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "tjcbf5e3",
			"name": "po_number",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": "^(20[2-9]\\d)-(0[1-9]\\d{2}|[1-4]\\d{3}|[1-9]\\d{1,2})$"
			}
		}`), edit_po_number); err != nil {
			return err
		}
		collection.Schema.AddField(edit_po_number)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		// update
		edit_po_number := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "tjcbf5e3",
			"name": "po_number",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": "^(20[2-9]\\d)-(\\d{4})$"
			}
		}`), edit_po_number); err != nil {
			return err
		}
		collection.Schema.AddField(edit_po_number)

		return dao.SaveCollection(collection)
	})
}
