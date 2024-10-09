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

		collection, err := dao.FindCollectionByNameOrId("1v6i9rrpniuatcx")
		if err != nil {
			return err
		}

		collection.DeleteRule = types.Pointer("@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job' &&\n// prevent deletion of clients if there are referencing jobs\n@collection.jobs.client != id")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("1v6i9rrpniuatcx")
		if err != nil {
			return err
		}

		collection.DeleteRule = nil

		return dao.SaveCollection(collection)
	})
}
