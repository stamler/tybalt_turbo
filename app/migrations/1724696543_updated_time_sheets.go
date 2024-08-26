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

		// update
		edit_week_ending := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "xoebt068",
			"name": "week_ending",
			"type": "text",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": "^\\d{4}-\\d{2}-\\d{2}$"
			}
		}`), edit_week_ending); err != nil {
			return err
		}
		collection.Schema.AddField(edit_week_ending)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("fpri53nrr2xgoov")
		if err != nil {
			return err
		}

		// update
		edit_week_ending := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "xoebt068",
			"name": "week_ending",
			"type": "text",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": "^\\\\d{4}-\\\\d{2}-\\\\d{2}$"
			}
		}`), edit_week_ending); err != nil {
			return err
		}
		collection.Schema.AddField(edit_week_ending)

		return dao.SaveCollection(collection)
	})
}
