package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("5z24r2v5jgh8qft")
		if err != nil {
			return err
		}

		collection.CreateRule = types.Pointer("@request.auth.user_claims_via_uid.cid.name ?= 'tame' &&\n// no tsid is submitted, it's set in the hook\n(@request.data.tsid:isset = false || @request.data.tsid = \"\")")

		collection.UpdateRule = types.Pointer("@request.auth.user_claims_via_uid.cid.name ?= 'tame' &&\ncommitted = \"\" &&\n// no tsid is submitted, it's set in the hook\n(@request.data.tsid:isset = false || @request.data.tsid = \"\")")

		// update
		edit_tsid := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "xy68bh6o",
			"name": "tsid",
			"type": "relation",
			"required": false,
			"presentable": true,
			"unique": false,
			"options": {
				"collectionId": "fpri53nrr2xgoov",
				"cascadeDelete": true,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), edit_tsid); err != nil {
			return err
		}
		collection.Schema.AddField(edit_tsid)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("5z24r2v5jgh8qft")
		if err != nil {
			return err
		}

		collection.CreateRule = types.Pointer("@request.auth.user_claims_via_uid.cid.name ?= 'tame'")

		collection.UpdateRule = types.Pointer("@request.auth.user_claims_via_uid.cid.name ?= 'tame' &&\ncommitted = \"\"")

		// update
		edit_tsid := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "xy68bh6o",
			"name": "tsid",
			"type": "relation",
			"required": true,
			"presentable": true,
			"unique": false,
			"options": {
				"collectionId": "fpri53nrr2xgoov",
				"cascadeDelete": true,
				"minSelect": null,
				"maxSelect": 1,
				"displayFields": null
			}
		}`), edit_tsid); err != nil {
			return err
		}
		collection.Schema.AddField(edit_tsid)

		return dao.SaveCollection(collection)
	})
}
