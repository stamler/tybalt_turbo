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

		collection.CreateRule = types.Pointer("// the caller is authenticated\n@request.auth.id != \"\" &&\n\n// the uid is equal to the authenticated user's id\n@request.data.uid = @request.auth.id &&\n\n// no rejection properties are submitted\n@request.data.rejector:isset = false &&\n@request.data.rejected:isset = false &&\n@request.data.rejection_reason:isset = false &&\n\n// no approval properties are submitted\n@request.data.approved:isset = false &&\n@request.data.approver:isset = false &&\n\n// if present, vendor is active\n(@request.data.vendor = \"\" || @request.data.vendor.status = \"Active\") &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // compare the new category to the new job\n  ( @request.data.job:isset = true && @request.data.category.job = @request.data.job ) ||\n  @request.data.category = \"\"\n)")

		collection.UpdateRule = types.Pointer("// only the creator can update the record\nuid = @request.auth.id &&\n\n// the uid must not change\n(@request.data.uid:isset = false || uid = @request.data.uid) &&\n\n// no rejection properties are submitted\n(@request.data.rejector:isset = false || rejector = @request.data.rejector) &&\n(@request.data.rejected:isset = false || rejected = @request.data.rejected) &&\n(@request.data.rejection_reason:isset = false || rejection_reason = @request.data.rejection_reason) &&\n\n// submitted is not changed\n(@request.data.submitted:isset = false || submitted = @request.data.submitted) &&\n\n// no approval properties are submitted\n(@request.data.approved:isset = false || approved = @request.data.approved) &&\n(@request.data.approver:isset = false || approver = @request.data.approver) &&\n\n// if present, vendor is active\n(@request.data.vendor = \"\" || @request.data.vendor.status = \"Active\") &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // the job is unchanged, compare the new category to job\n  ( @request.data.job:isset = false && @request.data.category.job = job ) ||\n  // the job has changed, compare the new category to the new job\n  ( @request.data.job:isset = true && @request.data.category.job = @request.data.job ) ||\n  @request.data.category = \"\"\n)")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("o1vpz1mm7qsfoyy")
		if err != nil {
			return err
		}

		collection.CreateRule = types.Pointer("// the caller is authenticated\n@request.auth.id != \"\" &&\n\n// the uid is equal to the authenticated user's id\n@request.data.uid = @request.auth.id &&\n\n// no rejection properties are submitted\n@request.data.rejector:isset = false &&\n@request.data.rejected:isset = false &&\n@request.data.rejection_reason:isset = false &&\n\n// no approval properties are submitted\n@request.data.approved:isset = false &&\n@request.data.approver:isset = false &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // compare the new category to the new job\n  ( @request.data.job:isset = true && @request.data.category.job = @request.data.job ) ||\n  @request.data.category = \"\"\n)")

		collection.UpdateRule = types.Pointer("// only the creator can update the record\nuid = @request.auth.id &&\n\n// the uid must not change\n(@request.data.uid:isset = false || uid = @request.data.uid) &&\n\n// no rejection properties are submitted\n(@request.data.rejector:isset = false || rejector = @request.data.rejector) &&\n(@request.data.rejected:isset = false || rejected = @request.data.rejected) &&\n(@request.data.rejection_reason:isset = false || rejection_reason = @request.data.rejection_reason) &&\n\n// submitted is not changed\n(@request.data.submitted:isset = false || submitted = @request.data.submitted) &&\n\n// no approval properties are submitted\n(@request.data.approved:isset = false || approved = @request.data.approved) &&\n(@request.data.approver:isset = false || approver = @request.data.approver) &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // the job is unchanged, compare the new category to job\n  ( @request.data.job:isset = false && @request.data.category.job = job ) ||\n  // the job has changed, compare the new category to the new job\n  ( @request.data.job:isset = true && @request.data.category.job = @request.data.job ) ||\n  @request.data.category = \"\"\n)")

		return dao.SaveCollection(collection)
	})
}
