package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("6z8rcof9bkpzz1t")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"listRule": "@request.auth.id = id ||\n@request.auth.id = manager_uid ||\n@request.auth.user_claims_via_uid.cid.name ?= 'report'",
			"viewRule": "@request.auth.id = id ||\n@request.auth.id = manager_uid ||\n@request.auth.user_claims_via_uid.cid.name ?= 'report'"
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_KcSN")

		// remove field
		collection.Fields.RemoveById("_clone_oTZV")

		// remove field
		collection.Fields.RemoveById("_clone_xXyY")

		// remove field
		collection.Fields.RemoveById("_clone_DZwb")

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(2, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_d5kV",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "manager_uid",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(4, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_nMMb",
			"max": 0,
			"min": 0,
			"name": "opening_date",
			"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(5, []byte(`{
			"hidden": false,
			"id": "_clone_npmM",
			"max": 200,
			"min": 0,
			"name": "opening_ov",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(6, []byte(`{
			"hidden": false,
			"id": "_clone_q4AY",
			"max": 332,
			"min": 0,
			"name": "opening_op",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("6z8rcof9bkpzz1t")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"listRule": "@request.auth.id = id ||\n@request.auth.id = manager_uid",
			"viewRule": "@request.auth.id = id ||\n@request.auth.id = manager_uid"
		}`), &collection); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(2, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_KcSN",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "manager_uid",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(4, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_oTZV",
			"max": 0,
			"min": 0,
			"name": "opening_date",
			"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(5, []byte(`{
			"hidden": false,
			"id": "_clone_xXyY",
			"max": 200,
			"min": 0,
			"name": "opening_ov",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(6, []byte(`{
			"hidden": false,
			"id": "_clone_DZwb",
			"max": 332,
			"min": 0,
			"name": "opening_op",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_d5kV")

		// remove field
		collection.Fields.RemoveById("_clone_nMMb")

		// remove field
		collection.Fields.RemoveById("_clone_npmM")

		// remove field
		collection.Fields.RemoveById("_clone_q4AY")

		return app.Save(collection)
	})
}
