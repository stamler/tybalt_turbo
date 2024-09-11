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
		edit_total := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "azgktu8n",
			"name": "total",
			"type": "number",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 0,
				"max": null,
				"noDecimal": false
			}
		}`), edit_total); err != nil {
			return err
		}
		collection.Schema.AddField(edit_total)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		// update
		edit_total := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "azgktu8n",
			"name": "total",
			"type": "number",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"noDecimal": false
			}
		}`), edit_total); err != nil {
			return err
		}
		collection.Schema.AddField(edit_total)

		return dao.SaveCollection(collection)
	})
}
