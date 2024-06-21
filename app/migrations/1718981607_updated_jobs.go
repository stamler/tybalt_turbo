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

		collection, err := dao.FindCollectionByNameOrId("yovqzrnnomp0lkx")
		if err != nil {
			return err
		}

		collection.CreateRule = types.Pointer("@collection.claims.user_claims_via_cid.uid.id = @request.auth.id &&\n@collection.claims.name = 'job'")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("yovqzrnnomp0lkx")
		if err != nil {
			return err
		}

		collection.CreateRule = types.Pointer("@request.auth.id = \"f2j5a8vk006baub\"")

		return dao.SaveCollection(collection)
	})
}
