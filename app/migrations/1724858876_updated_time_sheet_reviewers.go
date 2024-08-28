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

		collection.ViewRule = types.Pointer("@request.auth.id = @collection.users.time_sheets_via_approver.approver ||\n@request.auth.id = reviewer")

		collection.DeleteRule = types.Pointer("@request.auth.id != \"\" &&\n@request.auth.time_sheets_via_approver.id ?= time_sheet")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("g3surmbkacieshv")
		if err != nil {
			return err
		}

		collection.ViewRule = nil

		collection.DeleteRule = nil

		return dao.SaveCollection(collection)
	})
}
