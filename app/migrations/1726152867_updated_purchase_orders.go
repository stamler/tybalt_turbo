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
		edit_end_date := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "hqtvqmtx",
			"name": "end_date",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": "^\\d{4}-\\d{2}-\\d{2}$"
			}
		}`), edit_end_date); err != nil {
			return err
		}
		collection.Schema.AddField(edit_end_date)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		// update
		edit_end_date := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "hqtvqmtx",
			"name": "end_date",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": "^\\\\d{4}-\\\\d{2}-\\\\d{2}$"
			}
		}`), edit_end_date); err != nil {
			return err
		}
		collection.Schema.AddField(edit_end_date)

		return dao.SaveCollection(collection)
	})
}
