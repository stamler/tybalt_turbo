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

		collection, err := dao.FindCollectionByNameOrId("5z24r2v5jgh8qft")
		if err != nil {
			return err
		}

		// update
		edit_work_week_hours := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "vwe32gf1",
			"name": "work_week_hours",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 8,
				"max": 40,
				"noDecimal": false
			}
		}`), edit_work_week_hours); err != nil {
			return err
		}
		collection.Schema.AddField(edit_work_week_hours)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("5z24r2v5jgh8qft")
		if err != nil {
			return err
		}

		// update
		edit_work_week_hours := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "vwe32gf1",
			"name": "work_week_hours",
			"type": "number",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 8,
				"max": 40,
				"noDecimal": false
			}
		}`), edit_work_week_hours); err != nil {
			return err
		}
		collection.Schema.AddField(edit_work_week_hours)

		return dao.SaveCollection(collection)
	})
}
