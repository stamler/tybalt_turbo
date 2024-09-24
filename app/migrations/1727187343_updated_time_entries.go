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

		collection, err := dao.FindCollectionByNameOrId("ranctx5xgih6n3a")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.id = uid ||\n@request.auth.id = tsid.approver ||\n@request.auth.user_claims_via_uid.cid.name ?= 'report'")

		collection.ViewRule = types.Pointer("@request.auth.id = uid ||\n@request.auth.id = tsid.approver ||\n@request.auth.user_claims_via_uid.cid.name ?= 'report'")

		collection.UpdateRule = types.Pointer("// the creating user can edit if the entry is not yet part of a timesheet\nuid = @request.auth.id && tsid = \"\" &&\n// if present, the category belongs to the job, otherwise is blank\n(\n  // the job is unchanged, compare the new category to job\n  ( @request.data.job:isset = false && @request.data.category.job = job ) ||\n  // the job has changed, compare the new category to the new job\n  ( @request.data.job:isset = true && @request.data.category.job = @request.data.job ) ||\n  @request.data.category = \"\"\n)")

		collection.DeleteRule = types.Pointer("// request is from the creator and entry is not part of timesheet\n@request.auth.id = uid && tsid = \"\"")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("ranctx5xgih6n3a")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.id != \"\"")

		collection.ViewRule = types.Pointer("@request.auth.id != \"\"")

		collection.UpdateRule = types.Pointer("@request.auth.id != \"\"")

		collection.DeleteRule = types.Pointer("@request.auth.id != \"\"")

		return dao.SaveCollection(collection)
	})
}
