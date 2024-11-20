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

		collection.ListRule = types.Pointer("@request.auth.id != \"\" &&\n(uid = @request.auth.id || manager = @request.auth.id || @request.auth.user_claims_via_uid.cid.name ?= 'tame')")

		collection.ViewRule = types.Pointer("// The caller is logged in and either\n@request.auth.id != \"\" &&\n(\n  // 1. The caller id matches the record uid or\n  @request.auth.id = uid ||\n  // 2. The record has the tapr claim\n  uid.user_claims_via_uid.cid.name ?= 'tapr' ||\n  // 3. The record has the po_approver claim\n  uid.user_claims_via_uid.cid.name ?= 'po_approver' ||\n  // 4. The caller has the 'tame' claim\n  @request.auth.user_claims_via_uid.cid.name ?= 'tame'\n)")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("glmf9xpnwgpwudm")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.id != \"\" &&\n(uid = @request.auth.id || manager = @request.auth.id)")

		collection.ViewRule = types.Pointer("// The caller is logged in and either\n@request.auth.id != \"\" &&\n(\n  // 1. The caller id matches the record uid or\n  @request.auth.id = uid ||\n  // 2. The record has the tapr claim\n  uid.user_claims_via_uid.cid.name ?= 'tapr' ||\n  // 3. The record has the po_approver claim\n  uid.user_claims_via_uid.cid.name ?= 'po_approver'\n)")

		return dao.SaveCollection(collection)
	})
}
