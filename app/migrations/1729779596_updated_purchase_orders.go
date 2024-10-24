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

		collection.UpdateRule = types.Pointer("// only the creator can update the record\nuid = @request.auth.id &&\n\n// status is Unapproved and no approvals have been performed\nstatus = 'Unapproved' &&\napproved = \"\" &&\nsecond_approval = \"\"\n\n// no po_number is submitted\n(@request.data.po_number:isset = false || po_number = @request.data.po_number) &&\n\n// no rejection properties are submitted\n(@request.data.rejector:isset = false || rejector = @request.data.rejector) &&\n(@request.data.rejected:isset = false || rejected = @request.data.rejected) &&\n(@request.data.rejection_reason:isset = false || rejection_reason = @request.data.rejection_reason) &&\n\n// approved isn't set and approver has the right claim. Test divisions in payload in hooks\n(@request.data.approved:isset = false || approved = @request.data.approved) &&\n@request.data.approver.user_claims_via_uid.cid.name = 'po_approver' &&\n\n// no second approver properties are submitted\n(@request.data.second_approver:isset = false || second_approver = @request.data.second_approver) &&\n(@request.data.second_approval:isset = false || second_approval = @request.data.second_approval) &&\n(@request.data.second_approver_claim:isset = false || second_approver_claim = @request.data.second_approver_claim) &&\n\n// no cancellation properties are submitted\n(@request.data.cancelled:isset = false || cancelled = @request.data.cancelled) &&\n(@request.data.canceller:isset = false || canceller = @request.data.canceller) &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // the job is unchanged, compare the new category to job\n  ( @request.data.job:isset = false && @request.data.category.job = job ) ||\n  // the job has changed, compare the new category to the new job\n  ( @request.data.job:isset = true && @request.data.category.job = @request.data.job ) ||\n  @request.data.category = \"\"\n)")

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		collection.UpdateRule = types.Pointer("// only the creator can update the record\nuid = @request.auth.id &&\n\n// status is Unapproved and no approvals have been performed\nstatus = 'Unapproved' &&\napproved = \"\" &&\nsecond_approval = \"\"\n\n// no po_number is submitted\n(@request.data.po_number:isset = false || po_number = @request.data.po_number) &&\n\n// no rejection properties are submitted\n(@request.data.rejector:isset = false || rejector = @request.data.rejector) &&\n(@request.data.rejected:isset = false || rejected = @request.data.rejected) &&\n(@request.data.rejection_reason:isset = false || rejection_reason = @request.data.rejection_reason) &&\n\n// approved isn't set and approver has the right claim. Test divisions in payload in hooks\n(@request.data.approved:isset = false || approved = @request.data.approved) &&\n//@request.data.approver.user_claims_via_uid.cid.name = 'po_approver' &&\n\n// no second approver properties are submitted\n(@request.data.second_approver:isset = false || second_approver = @request.data.second_approver) &&\n(@request.data.second_approval:isset = false || second_approval = @request.data.second_approval) &&\n(@request.data.second_approver_claim:isset = false || second_approver_claim = @request.data.second_approver_claim) &&\n\n// no cancellation properties are submitted\n(@request.data.cancelled:isset = false || cancelled = @request.data.cancelled) &&\n(@request.data.canceller:isset = false || canceller = @request.data.canceller) &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // the job is unchanged, compare the new category to job\n  ( @request.data.job:isset = false && @request.data.category.job = job ) ||\n  // the job has changed, compare the new category to the new job\n  ( @request.data.job:isset = true && @request.data.category.job = @request.data.job ) ||\n  @request.data.category = \"\"\n)")

		return dao.SaveCollection(collection)
	})
}
