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

		collection, err := dao.FindCollectionByNameOrId("yovqzrnnomp0lkx")
		if err != nil {
			return err
		}

		// add
		new_client := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "efj2t5lj",
			"name": "client",
			"type": "relation",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "1v6i9rrpniuatcx",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), new_client); err != nil {
			return err
		}
		collection.Schema.AddField(new_client)

		// add
		new_contact := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "k65clvxw",
			"name": "contact",
			"type": "relation",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "3v7wxidd2f9yhf9",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), new_contact); err != nil {
			return err
		}
		collection.Schema.AddField(new_contact)

		// add
		new_manager := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "erlnpgrl",
			"name": "manager",
			"type": "relation",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"collectionId": "_pb_users_auth_",
				"cascadeDelete": false,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), new_manager); err != nil {
			return err
		}
		collection.Schema.AddField(new_manager)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("yovqzrnnomp0lkx")
		if err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("efj2t5lj")

		// remove
		collection.Schema.RemoveField("k65clvxw")

		// remove
		collection.Schema.RemoveField("erlnpgrl")

		return dao.SaveCollection(collection)
	})
}
