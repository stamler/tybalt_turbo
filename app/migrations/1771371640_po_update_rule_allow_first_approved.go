package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const purchaseOrdersUpdateRuleBeforeFirstApprovedEdits = `// only the creator can update the record
uid = @request.auth.id &&

// status is Unapproved and no approvals have been performed
status = 'Unapproved' &&
approved = "" &&
second_approval = ""

// no po_number is submitted
(@request.body.po_number:isset = false || po_number = @request.body.po_number) &&

// no rejection properties are submitted
(@request.body.rejector:isset = false || rejector = @request.body.rejector) &&
(@request.body.rejected:isset = false || rejected = @request.body.rejected) &&
(@request.body.rejection_reason:isset = false || rejection_reason = @request.body.rejection_reason) &&

// approved is unchanged
(@request.body.approved:isset = false || approved = @request.body.approved) &&

// no second approver properties are submitted
(@request.body.second_approver:isset = false || second_approver = @request.body.second_approver) &&
(@request.body.second_approval:isset = false || second_approval = @request.body.second_approval) &&

// no cancellation properties are submitted
(@request.body.cancelled:isset = false || cancelled = @request.body.cancelled) &&
(@request.body.canceller:isset = false || canceller = @request.body.canceller) &&

// no closed properties are submitted
(@request.body.closed:isset = false || closed = @request.body.closed) &&
(@request.body.closer:isset = false || closer = @request.body.closer) &&
(@request.body.closed_by_system:isset = false || closed_by_system = @request.body.closed_by_system) &&

// vendor is active (disabled, perform this in the hook for better error messages)
// @request.body.vendor.status = "Active" &&

// if present, the category belongs to the job, otherwise is blank
(
  // the job is unchanged, compare the new category to job
  ( @request.body.job:isset = false && @request.body.category.job = job ) ||
  // the job has changed, compare the new category to the new job
  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||
  @request.body.category = ""
)`

const purchaseOrdersUpdateRuleAllowFirstApprovedEdits = `// only the creator can update the record
uid = @request.auth.id &&

// status is Unapproved and second approval has not been performed
status = 'Unapproved' &&
second_approval = ""

// no po_number is submitted
(@request.body.po_number:isset = false || po_number = @request.body.po_number) &&

// no rejection properties are submitted
(@request.body.rejector:isset = false || rejector = @request.body.rejector) &&
(@request.body.rejected:isset = false || rejected = @request.body.rejected) &&
(@request.body.rejection_reason:isset = false || rejection_reason = @request.body.rejection_reason) &&

// approved is unchanged
(@request.body.approved:isset = false || approved = @request.body.approved) &&

// no second approver properties are submitted
(@request.body.second_approver:isset = false || second_approver = @request.body.second_approver) &&
(@request.body.second_approval:isset = false || second_approval = @request.body.second_approval) &&

// no cancellation properties are submitted
(@request.body.cancelled:isset = false || cancelled = @request.body.cancelled) &&
(@request.body.canceller:isset = false || canceller = @request.body.canceller) &&

// no closed properties are submitted
(@request.body.closed:isset = false || closed = @request.body.closed) &&
(@request.body.closer:isset = false || closer = @request.body.closer) &&
(@request.body.closed_by_system:isset = false || closed_by_system = @request.body.closed_by_system) &&

// vendor is active (disabled, perform this in the hook for better error messages)
// @request.body.vendor.status = "Active" &&

// if present, the category belongs to the job, otherwise is blank
(
  // the job is unchanged, compare the new category to job
  ( @request.body.job:isset = false && @request.body.category.job = job ) ||
  // the job has changed, compare the new category to the new job
  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||
  @request.body.category = ""
)`

func init() {
	m.Register(func(app core.App) error {
		return setPurchaseOrderUpdateRule(app, purchaseOrdersUpdateRuleAllowFirstApprovedEdits)
	}, func(app core.App) error {
		return setPurchaseOrderUpdateRule(app, purchaseOrdersUpdateRuleBeforeFirstApprovedEdits)
	})
}

func setPurchaseOrderUpdateRule(app core.App, rule string) error {
	collection, err := app.FindCollectionByNameOrId("purchase_orders")
	if err != nil {
		return err
	}

	collection.UpdateRule = stringPtr(rule)
	return app.SaveNoValidate(collection)
}
