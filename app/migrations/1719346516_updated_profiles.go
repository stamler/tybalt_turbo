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

		collection, err := dao.FindCollectionByNameOrId("glmf9xpnwgpwudm")
		if err != nil {
			return err
		}

		// add
		new_uid := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "e8mbl3rh",
			"name": "uid",
			"type": "relation",
			"required": false,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "_pb_users_auth_",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), new_uid); err != nil {
			return err
		}
		collection.Schema.AddField(new_uid)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("glmf9xpnwgpwudm")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("e8mbl3rh")

		return dao.SaveCollection(collection)
	})
}
