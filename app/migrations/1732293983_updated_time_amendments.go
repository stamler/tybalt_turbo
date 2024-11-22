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

		collection.CreateRule = types.Pointer("@request.auth.user_claims_via_uid.cid.name ?= 'tame' &&\n// no tsid is submitted, it's set in the hook\n(@request.data.tsid:isset = false || @request.data.tsid = \"\") &&\n\n// no committed properties are submitted\n@request.data.committed:isset = false &&\n@request.data.committer:isset = false &&\n@request.data.committed_week_ending:isset = false")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("5z24r2v5jgh8qft")
		if err != nil {
			return err
		}

		collection.CreateRule = types.Pointer("@request.auth.user_claims_via_uid.cid.name ?= 'tame' &&\n// no tsid is submitted, it's set in the hook\n(@request.data.tsid:isset = false || @request.data.tsid = \"\")")

		return dao.SaveCollection(collection)
	})
}
