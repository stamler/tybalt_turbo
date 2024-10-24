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

		collection, err := dao.FindCollectionByNameOrId("glmf9xpnwgpwudm")
		if err != nil {
			return err
		}

		collection.ViewRule = types.Pointer("// A user is logged in and either\n@request.auth.id != \"\" &&\n(\n  // 1. The user's id matches the record's uid or\n  @request.auth.id = uid ||\n  // 2. The user's id matches the record's manager or\n  @request.auth.id = manager ||\n  // 3. The user's id matches a purchase order's uid and that purchase order's approver matches the record's uid\n  (\n    @request.auth.id = @collection.purchase_orders.uid &&\n    @collection.purchase_orders.approver = uid\n  )\n)")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("glmf9xpnwgpwudm")
		if err != nil {
			return err
		}

		collection.ViewRule = types.Pointer("@request.auth.id != \"\" &&\n(uid = @request.auth.id || manager = @request.auth.id)")

		return dao.SaveCollection(collection)
	})
}
