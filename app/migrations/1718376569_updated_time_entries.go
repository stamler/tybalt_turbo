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

		// remove
		collection.Schema.RemoveField("xio2lxq5")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("ranctx5xgih6n3a")
		if err != nil {
			return err
		}

		// add
		del_job_hours := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "xio2lxq5",
			"name": "job_hours",
			"type": "number",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"min": 0,
				"max": 18,
				"noDecimal": false
			}
		}`), del_job_hours); err != nil {
			return err
		}
		collection.Schema.AddField(del_job_hours)

		return dao.SaveCollection(collection)
	})
}
