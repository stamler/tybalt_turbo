package migrations

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	bookKeeperClaimName = "book_keeper"
	bookKeeperClaimDesc = "Can create eligible PO-linked expenses on behalf of the purchase order owner"

	expensesCreateRuleBeforeCreator = `// the caller is authenticated
@request.auth.id != "" &&

// the pay_period_ending is not set or changed
@request.body.pay_period_ending:changed = false &&

// the uid is equal to the authenticated user's id
@request.body.uid = @request.auth.id &&

// no rejection properties are submitted
@request.body.rejector:isset = false &&
@request.body.rejected:isset = false &&
@request.body.rejection_reason:isset = false &&

// no approval properties are submitted
@request.body.approved:isset = false &&
@request.body.approver:isset = false &&

// no committed properties are submitted
@request.body.committed:isset = false &&
@request.body.committer:isset = false &&
@request.body.committed_week_ending:isset = false &&

// if present, vendor is active
(@request.body.vendor = "" || @request.body.vendor.status = "Active") &&

// if present, the category belongs to the job, otherwise is blank
(
  // compare the new category to the new job
  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||
  @request.body.category = ""
)`

	expensesUpdateRuleBeforeCreator = `// only the creator can update the record
uid = @request.auth.id &&

// the pay_period_ending is not set or changed
@request.body.pay_period_ending:changed = false &&

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

	expensesVisibilityRuleBeforeCreator = `uid = @request.auth.id ||
(approver = @request.auth.id && submitted = true) ||
(approved != "" && @request.auth.user_claims_via_uid.cid.name ?= 'commit') ||
(committed != "" && @request.auth.user_claims_via_uid.cid.name ?= 'report')`

	expensesDeleteRuleBeforeCreator = `@request.auth.id = uid && submitted = false && committed = ""`

	expensesVisibilityRuleWithCreator = `uid = @request.auth.id ||
creator = @request.auth.id ||
(approver = @request.auth.id && submitted = true) ||
(approved != "" && @request.auth.user_claims_via_uid.cid.name ?= 'commit') ||
(committed != "" && @request.auth.user_claims_via_uid.cid.name ?= 'report')`

	expensesCreateRuleWithCreator = `// the caller is authenticated
@request.auth.id != "" &&

// the pay_period_ending is not set or changed
@request.body.pay_period_ending:changed = false &&

// creator is server-managed by the hook
(@request.body.creator:isset = false || @request.body.creator = @request.auth.id) &&

// regular owner flow, or narrow bookkeeper-on-behalf PO flow
(
  @request.body.uid = @request.auth.id ||
  (
    @request.auth.user_claims_via_uid.cid.name ?= "book_keeper" &&
    @request.body.purchase_order:isset = true &&
    @request.body.purchase_order != "" &&
    @request.body.uid != @request.auth.id &&
    (
      @request.body.payment_type = "OnAccount" ||
      @request.body.payment_type = "CorporateCreditCard"
    )
  )
) &&

// no rejection properties are submitted
@request.body.rejector:isset = false &&
@request.body.rejected:isset = false &&
@request.body.rejection_reason:isset = false &&

// no approval properties are submitted
@request.body.approved:isset = false &&
@request.body.approver:isset = false &&

// no committed properties are submitted
@request.body.committed:isset = false &&
@request.body.committer:isset = false &&
@request.body.committed_week_ending:isset = false &&

// if present, vendor is active
(@request.body.vendor = "" || @request.body.vendor.status = "Active") &&

// if present, the category belongs to the job, otherwise is blank
(
  // compare the new category to the new job
  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||
  @request.body.category = ""
)`

	expensesUpdateRuleWithCreator = `// only the effective draft owner can update the record
@request.auth.id = creator &&

// the pay_period_ending is not set or changed
@request.body.pay_period_ending:changed = false &&

// ownership fields must not change
@request.body.uid:changed = false &&
@request.body.creator:changed = false &&

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

	expensesDeleteRuleWithCreator = `@request.auth.id != "" &&
submitted = false &&
committed = "" &&
@request.auth.id = creator`
)

func init() {
	m.Register(func(app core.App) error {
		if err := ensureBookKeeperClaim(app); err != nil {
			return err
		}

		collection, err := app.FindCollectionByNameOrId("expenses")
		if err != nil {
			return err
		}

		if collection.Fields.GetByName("creator") == nil {
			if err := collection.Fields.AddMarshaledJSONAt(13, []byte(`{
				"cascadeDelete": false,
				"collectionId": "_pb_users_auth_",
				"hidden": false,
				"id": "relation1777381896",
				"maxSelect": 1,
				"minSelect": 0,
				"name": "creator",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "relation"
			}`)); err != nil {
				return err
			}
		}

		if err := json.Unmarshal([]byte(`{
			"createRule": `+quoteJSONString(expensesCreateRuleWithCreator)+`,
			"listRule": `+quoteJSONString(expensesVisibilityRuleWithCreator)+`,
			"updateRule": `+quoteJSONString(expensesUpdateRuleWithCreator)+`,
			"viewRule": `+quoteJSONString(expensesVisibilityRuleWithCreator)+`,
			"deleteRule": `+quoteJSONString(expensesDeleteRuleWithCreator)+`,
			"indexes": [
				"CREATE UNIQUE INDEX `+"`"+`idx_KqwTULTh3p`+"`"+` ON `+"`"+`expenses`+"`"+` (`+"`"+`attachment_hash`+"`"+`) WHERE `+"`"+`attachment_hash`+"`"+` != ''",
				"CREATE INDEX `+"`"+`idx_8LRpecUoxd`+"`"+` ON `+"`"+`expenses`+"`"+` (\n  `+"`"+`purchase_order`+"`"+`,\n  `+"`"+`committed`+"`"+`\n)",
				"CREATE INDEX `+"`"+`idx_slBmqtw6SZ`+"`"+` ON `+"`"+`expenses`+"`"+` (`+"`"+`date`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_3TRP1AbuJv`+"`"+` ON `+"`"+`expenses`+"`"+` (\n  `+"`"+`branch`+"`"+`,\n  `+"`"+`job`+"`"+`\n)",
				"CREATE INDEX `+"`"+`idx_expenses_uid_date`+"`"+` ON `+"`"+`expenses`+"`"+` (`+"`"+`uid`+"`"+`, `+"`"+`date`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_expenses_approver_submitted_date`+"`"+` ON `+"`"+`expenses`+"`"+` (`+"`"+`approver`+"`"+`, `+"`"+`submitted`+"`"+`, `+"`"+`date`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_expenses_po_date`+"`"+` ON `+"`"+`expenses`+"`"+` (`+"`"+`purchase_order`+"`"+`, `+"`"+`date`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_expenses_approved_nonempty`+"`"+` ON `+"`"+`expenses`+"`"+` (`+"`"+`approved`+"`"+`) WHERE `+"`"+`approved`+"`"+` != ''",
				"CREATE INDEX `+"`"+`idx_expenses_committed_nonempty`+"`"+` ON `+"`"+`expenses`+"`"+` (`+"`"+`committed`+"`"+`) WHERE `+"`"+`committed`+"`"+` != ''",
				"CREATE INDEX `+"`"+`idx_Y3uLpJvqvc`+"`"+` ON `+"`"+`expenses`+"`"+` (`+"`"+`committed_week_ending`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_expenses_creator_date`+"`"+` ON `+"`"+`expenses`+"`"+` (`+"`"+`creator`+"`"+`, `+"`"+`date`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_expenses_creator_submitted_date`+"`"+` ON `+"`"+`expenses`+"`"+` (`+"`"+`creator`+"`"+`, `+"`"+`submitted`+"`"+`, `+"`"+`date`+"`"+`)"
			]
		}`), &collection); err != nil {
			return err
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		_, err = app.DB().NewQuery(`
			UPDATE expenses
			SET creator = uid
			WHERE COALESCE(creator, '') = ''
		`).Execute()
		return err
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("expenses")
		if err != nil {
			return err
		}

		collection.Fields.RemoveById("relation1777381896")
		if err := json.Unmarshal([]byte(`{
			"createRule": `+quoteJSONString(expensesCreateRuleBeforeCreator)+`,
			"listRule": `+quoteJSONString(expensesVisibilityRuleBeforeCreator)+`,
			"updateRule": `+quoteJSONString(expensesUpdateRuleBeforeCreator)+`,
			"viewRule": `+quoteJSONString(expensesVisibilityRuleBeforeCreator)+`,
			"deleteRule": `+quoteJSONString(expensesDeleteRuleBeforeCreator)+`,
			"indexes": [
				"CREATE UNIQUE INDEX `+"`"+`idx_KqwTULTh3p`+"`"+` ON `+"`"+`expenses`+"`"+` (`+"`"+`attachment_hash`+"`"+`) WHERE `+"`"+`attachment_hash`+"`"+` != ''",
				"CREATE INDEX `+"`"+`idx_8LRpecUoxd`+"`"+` ON `+"`"+`expenses`+"`"+` (\n  `+"`"+`purchase_order`+"`"+`,\n  `+"`"+`committed`+"`"+`\n)",
				"CREATE INDEX `+"`"+`idx_slBmqtw6SZ`+"`"+` ON `+"`"+`expenses`+"`"+` (`+"`"+`date`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_3TRP1AbuJv`+"`"+` ON `+"`"+`expenses`+"`"+` (\n  `+"`"+`branch`+"`"+`,\n  `+"`"+`job`+"`"+`\n)",
				"CREATE INDEX `+"`"+`idx_expenses_uid_date`+"`"+` ON `+"`"+`expenses`+"`"+` (`+"`"+`uid`+"`"+`, `+"`"+`date`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_expenses_approver_submitted_date`+"`"+` ON `+"`"+`expenses`+"`"+` (`+"`"+`approver`+"`"+`, `+"`"+`submitted`+"`"+`, `+"`"+`date`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_expenses_po_date`+"`"+` ON `+"`"+`expenses`+"`"+` (`+"`"+`purchase_order`+"`"+`, `+"`"+`date`+"`"+`)",
				"CREATE INDEX `+"`"+`idx_expenses_approved_nonempty`+"`"+` ON `+"`"+`expenses`+"`"+` (`+"`"+`approved`+"`"+`) WHERE `+"`"+`approved`+"`"+` != ''",
				"CREATE INDEX `+"`"+`idx_expenses_committed_nonempty`+"`"+` ON `+"`"+`expenses`+"`"+` (`+"`"+`committed`+"`"+`) WHERE `+"`"+`committed`+"`"+` != ''",
				"CREATE INDEX `+"`"+`idx_Y3uLpJvqvc`+"`"+` ON `+"`"+`expenses`+"`"+` (`+"`"+`committed_week_ending`+"`"+`)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}

func quoteJSONString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func ensureBookKeeperClaim(app core.App) error {
	existing, err := app.FindFirstRecordByFilter("claims", "name={:name}", dbx.Params{"name": bookKeeperClaimName})
	if err == nil && existing != nil {
		return nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	collection, err := app.FindCollectionByNameOrId("claims")
	if err != nil {
		return err
	}
	record := core.NewRecord(collection)
	record.Set("id", "bookkeeperclm01")
	record.Set("name", bookKeeperClaimName)
	record.Set("description", bookKeeperClaimDesc)
	return app.Save(record)
}
