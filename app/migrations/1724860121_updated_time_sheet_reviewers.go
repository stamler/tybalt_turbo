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

		collection, err := dao.FindCollectionByNameOrId("g3surmbkacieshv")
		if err != nil {
			return err
		}

		collection.DeleteRule = types.Pointer("@request.auth.id != \"\" &&\ntime_sheet ?= @request.auth.time_sheets_via_approver.id")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("g3surmbkacieshv")
		if err != nil {
			return err
		}

		collection.DeleteRule = types.Pointer("@request.auth.id != \"\" &&\n@request.auth.time_sheets_via_approver.id ?= time_sheet")

		return dao.SaveCollection(collection)
	})
}
