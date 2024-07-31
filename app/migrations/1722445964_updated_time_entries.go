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

		// add
		new_tsid := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "gih0hrty",
			"name": "tsid",
			"type": "relation",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "fpri53nrr2xgoov",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), new_tsid); err != nil {
			return err
		}
		collection.Schema.AddField(new_tsid)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("ranctx5xgih6n3a")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("gih0hrty")

		return dao.SaveCollection(collection)
	})
}
