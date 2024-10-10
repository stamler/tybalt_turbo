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

		collection, err := dao.FindCollectionByNameOrId("nrwhbwowokwu6cr")
		if err != nil {
			return err
		}

		collection.DeleteRule = types.Pointer("@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job' &&\n\n// prevent deletion of categories if there are referencing time_entries\n@collection.time_entries.category != id &&\n\n// prevent deletion of categories if there are referencing purchase orders\n@collection.purchase_orders.category != id &&\n\n// prevent deletion of categories if there are referencing expenses\n@collection.expenses.category != id")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("nrwhbwowokwu6cr")
		if err != nil {
			return err
		}

		collection.DeleteRule = types.Pointer("@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job' &&\n// prevent deletion of categories if there are referencing time_entries\n@collection.time_entries.category != id &&\n// prevent deletion of categories if there are referencing purchase orders\n@collection.purchase_orders.category != id\n// prevent deletion of categories if there are referencing expenses\n//@collection.expenses.category != id")

		return dao.SaveCollection(collection)
	})
}
