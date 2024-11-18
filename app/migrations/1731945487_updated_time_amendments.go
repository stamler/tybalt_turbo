package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("5z24r2v5jgh8qft")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.user_claims_via_uid.cid.name ?= 'tame' ||\n@request.auth.user_claims_via_uid.cid.name ?= 'report'")

		collection.ViewRule = types.Pointer("@request.auth.user_claims_via_uid.cid.name ?= 'tame' ||\n@request.auth.user_claims_via_uid.cid.name ?= 'report'")

		collection.CreateRule = types.Pointer("@request.auth.user_claims_via_uid.cid.name ?= 'tame'")

		collection.UpdateRule = types.Pointer("@request.auth.user_claims_via_uid.cid.name ?= 'tame' &&\ncommitted = \"\"")

		collection.DeleteRule = types.Pointer("@request.auth.user_claims_via_uid.cid.name ?= 'tame' &&\ncommitted = \"\"")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("5z24r2v5jgh8qft")
		if err != nil {
			return err
		}

		collection.ListRule = nil

		collection.ViewRule = nil

		collection.CreateRule = nil

		collection.UpdateRule = nil

		collection.DeleteRule = nil

		return dao.SaveCollection(collection)
	})
}
