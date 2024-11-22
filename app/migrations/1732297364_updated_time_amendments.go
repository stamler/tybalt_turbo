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

		collection, err := dao.FindCollectionByNameOrId("5z24r2v5jgh8qft")
		if err != nil {
			return err
		}

		collection.CreateRule = types.Pointer("@request.auth.user_claims_via_uid.cid.name ?= 'tame' &&\n// no tsid is submitted, it's set in the hook\n(@request.data.tsid:isset = false || @request.data.tsid = \"\") &&\n\n// no committed properties are submitted\n@request.data.committed:isset = false &&\n@request.data.committer:isset = false &&\n@request.data.committed_week_ending:isset = false &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // compare the new category to the new job\n  ( @request.data.job:isset = true && @request.data.category.job = @request.data.job ) ||\n  @request.data.category = \"\"\n)")

		collection.UpdateRule = types.Pointer("@request.auth.user_claims_via_uid.cid.name ?= 'tame' &&\ncommitted = \"\" &&\n// no tsid is submitted, it's set in the hook\n(@request.data.tsid:isset = false || tsid = @request.data.tsid) &&\n\n// no committed properties are submitted\n(@request.data.committed:isset = false || committed = @request.data.committed) &&\n(@request.data.committer:isset = false || committer = @request.data.committer) &&\n(@request.data.committed_week_ending:isset = false || committed_week_ending = @request.data.committed_week_ending) &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // the job is unchanged, compare the new category to job\n  ( @request.data.job:isset = false && @request.data.category.job = job ) ||\n  // the job has changed, compare the new category to the new job\n  ( @request.data.job:isset = true && @request.data.category.job = @request.data.job ) ||\n  @request.data.category = \"\"\n)")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("5z24r2v5jgh8qft")
		if err != nil {
			return err
		}

		collection.CreateRule = types.Pointer("@request.auth.user_claims_via_uid.cid.name ?= 'tame' &&\n// no tsid is submitted, it's set in the hook\n(@request.data.tsid:isset = false || @request.data.tsid = \"\") &&\n\n// no committed properties are submitted\n@request.data.committed:isset = false &&\n@request.data.committer:isset = false &&\n@request.data.committed_week_ending:isset = false")

		collection.UpdateRule = types.Pointer("@request.auth.user_claims_via_uid.cid.name ?= 'tame' &&\ncommitted = \"\" &&\n// no tsid is submitted, it's set in the hook\n(@request.data.tsid:isset = false || tsid = @request.data.tsid) &&\n\n// no committed properties are submitted\n(@request.data.committed:isset = false || committed = @request.data.committed) &&\n(@request.data.committer:isset = false || committer = @request.data.committer) &&\n(@request.data.committed_week_ending:isset = false || committed_week_ending = @request.data.committed_week_ending)")

		return dao.SaveCollection(collection)
	})
}
