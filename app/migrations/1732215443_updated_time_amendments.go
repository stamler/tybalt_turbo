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
		edit_hours := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "1esmkvan",
			"name": "hours",
			"type": "number",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": -18,
				"max": 18,
				"noDecimal": false
			}
		}`), edit_hours); err != nil {
			return err
		}
		collection.Schema.AddField(edit_hours)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("5z24r2v5jgh8qft")
		if err != nil {
			return err
		}

		// update
		edit_hours := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "1esmkvan",
			"name": "hours",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": -18,
				"max": 18,
				"noDecimal": false
			}
		}`), edit_hours); err != nil {
			return err
		}
		collection.Schema.AddField(edit_hours)

		return dao.SaveCollection(collection)
	})
}
