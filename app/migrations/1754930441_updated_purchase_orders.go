package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"createRule": "// the caller is authenticated\n@request.auth.id != \"\" &&\n\n// no po_number is submitted\n(@request.body.po_number:isset = false || @request.body.po_number = \"\") &&\n\n// status is Unapproved\n@request.body.status = \"Unapproved\" &&\n\n// the uid is missing or is equal to the authenticated user's id\n(@request.body.uid:isset = false || @request.body.uid = @request.auth.id) &&\n\n// no rejection properties are submitted\n@request.body.rejector:isset = false &&\n@request.body.rejected:isset = false &&\n@request.body.rejection_reason:isset = false &&\n\n// approved isn't set. We check that the approver has the appropriate claim and divisions in payload in hooks, commenting out the previous check for po_approver here. \n@request.body.approved:isset = false &&\n//@request.body.approver.user_claims_via_uid.cid.name ?= 'po_approver' &&\n\n// no second approver properties are submitted\n@request.body.second_approver:isset = false &&\n@request.body.second_approval:isset = false &&\n\n// no cancellation properties are submitted\n@request.body.cancelled:isset = false &&\n@request.body.canceller:isset = false &&\n\n// no closed properties are submitted\n@request.body.closed:isset = false &&\n@request.body.closer:isset = false &&\n@request.body.closed_by_system:isset = false &&\n\n// vendor is active (disabled, perform this in the hook for better error messages)\n// @request.body.vendor.status = \"Active\" &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // compare the new category to the new job\n  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||\n  @request.body.category = \"\"\n)",
			"updateRule": "// only the creator can update the record\nuid = @request.auth.id &&\n\n// status is Unapproved and no approvals have been performed\nstatus = 'Unapproved' &&\napproved = \"\" &&\nsecond_approval = \"\"\n\n// no po_number is submitted\n(@request.body.po_number:isset = false || po_number = @request.body.po_number) &&\n\n// no rejection properties are submitted\n(@request.body.rejector:isset = false || rejector = @request.body.rejector) &&\n(@request.body.rejected:isset = false || rejected = @request.body.rejected) &&\n(@request.body.rejection_reason:isset = false || rejection_reason = @request.body.rejection_reason) &&\n\n// approved is unchanged\n(@request.body.approved:isset = false || approved = @request.body.approved) &&\n\n// no second approver properties are submitted\n(@request.body.second_approver:isset = false || second_approver = @request.body.second_approver) &&\n(@request.body.second_approval:isset = false || second_approval = @request.body.second_approval) &&\n\n// no cancellation properties are submitted\n(@request.body.cancelled:isset = false || cancelled = @request.body.cancelled) &&\n(@request.body.canceller:isset = false || canceller = @request.body.canceller) &&\n\n// no closed properties are submitted\n(@request.body.closed:isset = false || closed = @request.body.closed) &&\n(@request.body.closer:isset = false || closer = @request.body.closer) &&\n(@request.body.closed_by_system:isset = false || closed_by_system = @request.body.closed_by_system) &&\n\n// vendor is active (disabled, perform this in the hook for better error messages)\n// @request.body.vendor.status = \"Active\" &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // the job is unchanged, compare the new category to job\n  ( @request.body.job:isset = false && @request.body.category.job = job ) ||\n  // the job has changed, compare the new category to the new job\n  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||\n  @request.body.category = \"\"\n)"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"createRule": "// the caller is authenticated\n@request.auth.id != \"\" &&\n\n// no po_number is submitted\n(@request.body.po_number:isset = false || @request.body.po_number = \"\") &&\n\n// status is Unapproved\n@request.body.status = \"Unapproved\" &&\n\n// the uid is missing or is equal to the authenticated user's id\n(@request.body.uid:isset = false || @request.body.uid = @request.auth.id) &&\n\n// no rejection properties are submitted\n@request.body.rejector:isset = false &&\n@request.body.rejected:isset = false &&\n@request.body.rejection_reason:isset = false &&\n\n// approved isn't set. We check that the approver has the appropriate claim and divisions in payload in hooks, commenting out the previous check for po_approver here. \n@request.body.approved:isset = false &&\n//@request.body.approver.user_claims_via_uid.cid.name ?= 'po_approver' &&\n\n// no second approver properties are submitted\n@request.body.second_approver:isset = false &&\n@request.body.second_approval:isset = false &&\n\n// no cancellation properties are submitted\n@request.body.cancelled:isset = false &&\n@request.body.canceller:isset = false &&\n\n// no closed properties are submitted\n@request.body.closed:isset = false &&\n@request.body.closer:isset = false &&\n@request.body.closed_by_system:isset = false &&\n\n// vendor is active\n@request.body.vendor.status = \"Active\" &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // compare the new category to the new job\n  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||\n  @request.body.category = \"\"\n)",
			"updateRule": "// only the creator can update the record\nuid = @request.auth.id &&\n\n// status is Unapproved and no approvals have been performed\nstatus = 'Unapproved' &&\napproved = \"\" &&\nsecond_approval = \"\"\n\n// no po_number is submitted\n(@request.body.po_number:isset = false || po_number = @request.body.po_number) &&\n\n// no rejection properties are submitted\n(@request.body.rejector:isset = false || rejector = @request.body.rejector) &&\n(@request.body.rejected:isset = false || rejected = @request.body.rejected) &&\n(@request.body.rejection_reason:isset = false || rejection_reason = @request.body.rejection_reason) &&\n\n// approved is unchanged\n(@request.body.approved:isset = false || approved = @request.body.approved) &&\n\n// no second approver properties are submitted\n(@request.body.second_approver:isset = false || second_approver = @request.body.second_approver) &&\n(@request.body.second_approval:isset = false || second_approval = @request.body.second_approval) &&\n\n// no cancellation properties are submitted\n(@request.body.cancelled:isset = false || cancelled = @request.body.cancelled) &&\n(@request.body.canceller:isset = false || canceller = @request.body.canceller) &&\n\n// no closed properties are submitted\n(@request.body.closed:isset = false || closed = @request.body.closed) &&\n(@request.body.closer:isset = false || closer = @request.body.closer) &&\n(@request.body.closed_by_system:isset = false || closed_by_system = @request.body.closed_by_system) &&\n\n// vendor is active\n@request.body.vendor.status = \"Active\" &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // the job is unchanged, compare the new category to job\n  ( @request.body.job:isset = false && @request.body.category.job = job ) ||\n  // the job has changed, compare the new category to the new job\n  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||\n  @request.body.category = \"\"\n)"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
