package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const expensesUpdateRuleBeforeOwnerHardening = `// only the creator can update the record
uid = @request.auth.id &&

// the uid must not change
(@request.body.uid:isset = false || uid = @request.body.uid) &&

// no rejection properties are submitted
(@request.body.rejector:isset = false || rejector = @request.body.rejector) &&
(@request.body.rejected:isset = false || rejected = @request.body.rejected) &&
(@request.body.rejection_reason:isset = false || rejection_reason = @request.body.rejection_reason) &&

// submitted is not changed
(@request.body.submitted:isset = false || submitted = @request.body.submitted) &&

// no approval properties are submitted
(@request.body.approved:isset = false || approved = @request.body.approved) &&
(@request.body.approver:isset = false || approver = @request.body.approver) &&

// no committed properties are submitted
(@request.body.committed:isset = false || committed = @request.body.committed) &&
(@request.body.committer:isset = false || committer = @request.body.committer) &&
(@request.body.committed_week_ending:isset = false || committed_week_ending = @request.body.committed_week_ending) &&

// if present, vendor is active
(@request.body.vendor = "" || @request.body.vendor.status = "Active") &&

// if present, the category belongs to the job, otherwise is blank
(
  // the job is unchanged, compare the new category to job
  ( @request.body.job:isset = false && @request.body.category.job = job ) ||
  // the job has changed, compare the new category to the new job
  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||
  @request.body.category = ""
)`

const expensesUpdateRuleAfterOwnerHardening = `// only the creator can update the record
uid = @request.auth.id &&

// the uid must not change
@request.body.uid:changed = false &&

// no rejection properties are submitted
(@request.body.rejector:isset = false || rejector = @request.body.rejector) &&
(@request.body.rejected:isset = false || rejected = @request.body.rejected) &&
(@request.body.rejection_reason:isset = false || rejection_reason = @request.body.rejection_reason) &&

// submitted is not changed
(@request.body.submitted:isset = false || submitted = @request.body.submitted) &&

// no approval properties are submitted
(@request.body.approved:isset = false || approved = @request.body.approved) &&
(@request.body.approver:isset = false || approver = @request.body.approver) &&

// no committed properties are submitted
(@request.body.committed:isset = false || committed = @request.body.committed) &&
(@request.body.committer:isset = false || committer = @request.body.committer) &&
(@request.body.committed_week_ending:isset = false || committed_week_ending = @request.body.committed_week_ending) &&

// if present, vendor is active
(@request.body.vendor = "" || @request.body.vendor.status = "Active") &&

// if present, the category belongs to the job, otherwise is blank
(
  // the job is unchanged, compare the new category to job
  ( @request.body.job:isset = false && @request.body.category.job = job ) ||
  // the job has changed, compare the new category to the new job
  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||
  @request.body.category = ""
)`

const profilesUpdateRuleBeforeOwnerHardening = `@request.auth.id != "" &&
uid = @request.auth.id`

const profilesUpdateRuleAfterOwnerHardening = `@request.auth.id != "" &&
uid = @request.auth.id &&
@request.body.uid:changed = false`

const purchaseOrdersUpdateRuleBeforeOwnerHardening = `// only the creator can update the record
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

const purchaseOrdersUpdateRuleAfterOwnerHardening = `// only the creator can update the record
uid = @request.auth.id &&

// the uid must not change
@request.body.uid:changed = false &&

// status is Unapproved and second approval has not been performed
status = 'Unapproved' &&
second_approval = "" &&

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

const timeAmendmentsUpdateRuleBeforeOwnerHardening = `@request.auth.user_claims_via_uid.cid.name ?= 'tame' &&
committed = "" &&
// no tsid is submitted, it's set in the hook
(@request.body.tsid:isset = false || tsid = @request.body.tsid) &&

// no committed properties are submitted
(@request.body.committed:isset = false || committed = @request.body.committed) &&
(@request.body.committer:isset = false || committer = @request.body.committer) &&
(@request.body.committed_week_ending:isset = false || committed_week_ending = @request.body.committed_week_ending) &&

// if present, the category belongs to the job, otherwise is blank
(
  // the job is unchanged, compare the new category to job
  ( @request.body.job:isset = false && @request.body.category.job = job ) ||
  // the job has changed, compare the new category to the new job
  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||
  @request.body.category = ""
)`

const timeAmendmentsUpdateRuleAfterOwnerHardening = `@request.auth.id != "" &&
@request.auth.user_claims_via_uid.cid.name ?= 'tame' &&
committed = "" &&

// subject employee must remain stable after create
@request.body.uid:changed = false &&

// no tsid is submitted, it's set in the hook
(@request.body.tsid:isset = false || tsid = @request.body.tsid) &&

// no committed properties are submitted
(@request.body.committed:isset = false || committed = @request.body.committed) &&
(@request.body.committer:isset = false || committer = @request.body.committer) &&
(@request.body.committed_week_ending:isset = false || committed_week_ending = @request.body.committed_week_ending) &&

// if present, the category belongs to the job, otherwise is blank
(
  // the job is unchanged, compare the new category to job
  ( @request.body.job:isset = false && @request.body.category.job = job ) ||
  // the job has changed, compare the new category to the new job
  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||
  @request.body.category = ""
)`

const timeEntriesUpdateRuleBeforeOwnerHardening = `// the creating user can edit if the entry is not yet part of a timesheet
uid = @request.auth.id && tsid = "" &&
// if present, the category belongs to the job, otherwise is blank
(
  // the job is unchanged, compare the new category to job
  ( @request.body.job:isset = false && @request.body.category.job = job ) ||
  // the job has changed, compare the new category to the new job
  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||
  @request.body.category = ""
)`

const timeEntriesUpdateRuleAfterOwnerHardening = `// the creating user can edit if the entry is not yet part of a timesheet
uid = @request.auth.id && tsid = "" &&

// uid must not change after create
@request.body.uid:changed = false &&

// if present, the category belongs to the job, otherwise is blank
(
  // the job is unchanged, compare the new category to job
  ( @request.body.job:isset = false && @request.body.category.job = job ) ||
  // the job has changed, compare the new category to the new job
  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||
  @request.body.category = ""
)`

const timeSheetsUpdateRuleBeforeOwnerHardening = `(rejected = true && rejector != "" && rejection_reason != "") || (rejected = false)`

type updateRulePatch struct {
	collection string
	before     *string
	after      *string
}

var ownerEditUpdateRulePatches = []updateRulePatch{
	{
		collection: "expenses",
		before:     stringPtr(expensesUpdateRuleBeforeOwnerHardening),
		after:      stringPtr(expensesUpdateRuleAfterOwnerHardening),
	},
	{
		collection: "profiles",
		before:     stringPtr(profilesUpdateRuleBeforeOwnerHardening),
		after:      stringPtr(profilesUpdateRuleAfterOwnerHardening),
	},
	{
		collection: "purchase_orders",
		before:     stringPtr(purchaseOrdersUpdateRuleBeforeOwnerHardening),
		after:      stringPtr(purchaseOrdersUpdateRuleAfterOwnerHardening),
	},
	{
		collection: "time_amendments",
		before:     stringPtr(timeAmendmentsUpdateRuleBeforeOwnerHardening),
		after:      stringPtr(timeAmendmentsUpdateRuleAfterOwnerHardening),
	},
	{
		collection: "time_entries",
		before:     stringPtr(timeEntriesUpdateRuleBeforeOwnerHardening),
		after:      stringPtr(timeEntriesUpdateRuleAfterOwnerHardening),
	},
	{
		collection: "time_sheets",
		before:     stringPtr(timeSheetsUpdateRuleBeforeOwnerHardening),
		after:      nil,
	},
}

func init() {
	m.Register(func(app core.App) error {
		return applyOwnerEditUpdateRules(app, true)
	}, func(app core.App) error {
		return applyOwnerEditUpdateRules(app, false)
	})
}

func applyOwnerEditUpdateRules(app core.App, forward bool) error {
	for _, patch := range ownerEditUpdateRulePatches {
		rule := patch.before
		if forward {
			rule = patch.after
		}

		collection, err := app.FindCollectionByNameOrId(patch.collection)
		if err != nil {
			return err
		}

		collection.UpdateRule = rule
		if err := app.SaveNoValidate(collection); err != nil {
			return err
		}
	}

	return nil
}
