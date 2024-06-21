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

		collection, err := dao.FindCollectionByNameOrId("cnqv0wm8hly7r3n")
		if err != nil {
			return err
		}

		collection.CreateRule = types.Pointer("@collection.claims.user_claims_via_cid.uid.id = @request.auth.id &&\n@collection.claims.name = 'tt'")

		collection.DeleteRule = types.Pointer("@collection.claims.user_claims_via_cid.uid.id = @request.auth.id &&\n@collection.claims.name = 'tt'")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("cnqv0wm8hly7r3n")
		if err != nil {
			return err
		}

		collection.CreateRule = nil

		collection.DeleteRule = nil

		return dao.SaveCollection(collection)
	})
}
