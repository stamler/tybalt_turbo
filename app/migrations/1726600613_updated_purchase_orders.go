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

		collection.ListRule = types.Pointer("@request.auth.id = uid ||\n@request.auth.id = approver ||\n@request.auth.id = second_approver")

		collection.CreateRule = types.Pointer("// the caller is authenticated\n@request.auth.id != \"\" &&\n\n// no po_number is submitted\n@request.data.po_number:isset = false &&\n\n// status is Unapproved\n@request.data.status = \"Unapproved\" &&\n\n// the uid is missing or is equal to the authenticated user's id\n(@request.data.uid:isset = false || @request.data.uid = @request.auth.id) &&\n\n// no rejection properties are submitted\n@request.data.rejector:isset = false &&\n@request.data.rejected:isset = false &&\n@request.data.rejection_reason:isset = false &&\n\n// no approval properties are submitted\n@request.data.approved:isset = false &&\n@request.data.approver:isset = false &&\n\n// no second approver properties are submitted\n@request.data.second_approver:isset = false &&\n@request.data.second_approval:isset = false &&\n@request.data.second_approver_claim:isset = false &&\n\n// no cancellation properties are submitted\n@request.data.cancelled:isset = false &&\n@request.data.canceller:isset = false")

		collection.UpdateRule = types.Pointer("// only the creator can update the record\nuid = @request.auth.id &&\n\n// status is Unapproved\nstatus = 'Unapproved' &&\n\n// no po_number is submitted\n(@request.data.po_number:isset = false || po_number = @request.data.po_number) &&\n\n// no rejection properties are submitted\n(@request.data.rejector:isset = false || rejector = @request.data.rejector) &&\n(@request.data.rejected:isset = false || rejected = @request.data.rejected) &&\n(@request.data.rejection_reason:isset = false || rejection_reason = @request.data.rejection_reason) &&\n\n// no approval properties are submitted\n(@request.data.approved:isset = false || approved = @request.data.approved) &&\n(@request.data.approver:isset = false || approver = @request.data.approver) &&\n\n// no second approver properties are submitted\n(@request.data.second_approver:isset = false || second_approver = @request.data.second_approver) &&\n(@request.data.second_approval:isset = false || second_approval = @request.data.second_approval) &&\n(@request.data.second_approver_claim:isset = false || second_approver_claim = @request.data.second_approver_claim) &&\n\n// no cancellation properties are submitted\n(@request.data.cancelled:isset = false || cancelled = @request.data.cancelled) &&\n(@request.data.canceller:isset = false || canceller = @request.data.canceller)")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer("@request.auth.id != \"\" &&\n(\n  @request.auth.id = uid ||\n  @request.auth.id = approver ||\n  @request.auth.id = second_approver\n)")

		collection.CreateRule = types.Pointer("@request.auth.id != \"\"")

		collection.UpdateRule = types.Pointer("@request.auth.id = uid && status = 'Unapproved'")

		return dao.SaveCollection(collection)
	})
}
