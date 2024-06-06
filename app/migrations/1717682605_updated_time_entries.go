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

		collection, err := dao.FindCollectionByNameOrId("ranctx5xgih6n3a")
		if err != nil {
			return err
		}

		// update
		edit_work_record := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "fjcrzqdc",
			"name": "work_record",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": "^[FKQ][0-9]{2}-[0-9]{3,}(-[0-9]+)?$"
			}
		}`), edit_work_record); err != nil {
			return err
		}
		collection.Schema.AddField(edit_work_record)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("ranctx5xgih6n3a")
		if err != nil {
			return err
		}

		// update
		edit_work_record := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "fjcrzqdc",
			"name": "workrecord",
			"type": "text",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": "^[FKQ][0-9]{2}-[0-9]{3,}(-[0-9]+)?$"
			}
		}`), edit_work_record); err != nil {
			return err
		}
		collection.Schema.AddField(edit_work_record)

		return dao.SaveCollection(collection)
	})
}
