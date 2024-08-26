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

		collection, err := dao.FindCollectionByNameOrId("fpri53nrr2xgoov")
		if err != nil {
			return err
		}

		collection.UpdateRule = types.Pointer("(rejected = true && rejector != \"\" && rejection_reason != \"\") || (rejected = false)")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("fpri53nrr2xgoov")
		if err != nil {
			return err
		}

		collection.UpdateRule = nil

		return dao.SaveCollection(collection)
	})
}
