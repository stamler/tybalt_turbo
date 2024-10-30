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

		collection, err := dao.FindCollectionByNameOrId("o1vpz1mm7qsfoyy")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("uid = @request.auth.id ||\n(approver = @request.auth.id && submitted = true)")

		collection.ViewRule = types.Pointer("uid = @request.auth.id ||\n(approver = @request.auth.id && submitted = true)")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("o1vpz1mm7qsfoyy")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("uid = @request.auth.id ||\napprover = @request.auth.id")

		collection.ViewRule = types.Pointer("uid = @request.auth.id ||\napprover = @request.auth.id")

		return dao.SaveCollection(collection)
	})
}
