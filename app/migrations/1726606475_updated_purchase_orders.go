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

		collection, err := dao.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		collection.CreateRule = types.Pointer("// the caller is authenticated\n@request.auth.id != \"\" &&\n\n// no po_number is submitted\n(@request.data.po_number:isset = false || @request.data.po_number = \"\") &&\n\n// status is Unapproved\n@request.data.status = \"Unapproved\" &&\n\n// the uid is missing or is equal to the authenticated user's id\n(@request.data.uid:isset = false || @request.data.uid = @request.auth.id) &&\n\n// no rejection properties are submitted\n@request.data.rejector:isset = false &&\n@request.data.rejected:isset = false &&\n@request.data.rejection_reason:isset = false &&\n\n// no approval properties are submitted\n@request.data.approved:isset = false &&\n@request.data.approver:isset = false &&\n\n// no second approver properties are submitted\n@request.data.second_approver:isset = false &&\n@request.data.second_approval:isset = false &&\n@request.data.second_approver_claim:isset = false &&\n\n// no cancellation properties are submitted\n@request.data.cancelled:isset = false &&\n@request.data.canceller:isset = false")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		collection.CreateRule = types.Pointer("// the caller is authenticated\n@request.auth.id != \"\" &&\n\n// no po_number is submitted\n@request.data.po_number:isset = false &&\n\n// status is Unapproved\n@request.data.status = \"Unapproved\" &&\n\n// the uid is missing or is equal to the authenticated user's id\n(@request.data.uid:isset = false || @request.data.uid = @request.auth.id) &&\n\n// no rejection properties are submitted\n@request.data.rejector:isset = false &&\n@request.data.rejected:isset = false &&\n@request.data.rejection_reason:isset = false &&\n\n// no approval properties are submitted\n@request.data.approved:isset = false &&\n@request.data.approver:isset = false &&\n\n// no second approver properties are submitted\n@request.data.second_approver:isset = false &&\n@request.data.second_approval:isset = false &&\n@request.data.second_approver_claim:isset = false &&\n\n// no cancellation properties are submitted\n@request.data.cancelled:isset = false &&\n@request.data.canceller:isset = false")

		return dao.SaveCollection(collection)
	})
}
