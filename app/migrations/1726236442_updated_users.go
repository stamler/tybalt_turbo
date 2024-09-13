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

		collection, err := dao.FindCollectionByNameOrId("_pb_users_auth_")
		if err != nil {
			return err
		}

		collection.ViewRule = types.Pointer("// A user is logged in and either\n@request.auth.id != \"\" &&\n(\n  // 1. The user's id is equal to the record's id or\n  @request.auth.id = id ||\n  // 2. The user's id is the approver of a purchase order and the purchase order's uid matches the record's id or\n  (\n    id = @collection.purchase_orders.uid &&\n    @collection.purchase_orders.approver = @request.auth.id\n  ) ||\n  // 3. The user's id is the uid of a purchase order and the purchase order's approver matches the record's id\n  (\n    @collection.purchase_orders.approver = id &&\n    @collection.purchase_orders.uid = @request.auth.id\n  )\n)")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("_pb_users_auth_")
		if err != nil {
			return err
		}

		collection.ViewRule = types.Pointer("id = @request.auth.id")

		return dao.SaveCollection(collection)
	})
}
