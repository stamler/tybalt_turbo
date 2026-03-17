package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("o1vpz1mm7qsfoyy")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"createRule": "// the caller is authenticated\n@request.auth.id != \"\" &&\n\n// the pay_period_ending is not set or changed\n@request.body.pay_period_ending:changed = false &&\n\n// the uid is equal to the authenticated user's id\n@request.body.uid = @request.auth.id &&\n\n// no rejection properties are submitted\n@request.body.rejector:isset = false &&\n@request.body.rejected:isset = false &&\n@request.body.rejection_reason:isset = false &&\n\n// no approval properties are submitted\n@request.body.approved:isset = false &&\n@request.body.approver:isset = false &&\n\n// no committed properties are submitted\n@request.body.committed:isset = false &&\n@request.body.committer:isset = false &&\n@request.body.committed_week_ending:isset = false &&\n\n// if present, vendor is active\n(@request.body.vendor = \"\" || @request.body.vendor.status = \"Active\") &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // compare the new category to the new job\n  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||\n  @request.body.category = \"\"\n)",
			"updateRule": "// only the creator can update the record\nuid = @request.auth.id &&\n\n// the pay_period_ending is not set or changed\n@request.body.pay_period_ending:changed = false &&\n\n// the uid must not change\n@request.body.uid:changed = false &&\n\n// no rejection properties are submitted\n(@request.body.rejector:isset = false || rejector = @request.body.rejector) &&\n(@request.body.rejected:isset = false || rejected = @request.body.rejected) &&\n(@request.body.rejection_reason:isset = false || rejection_reason = @request.body.rejection_reason) &&\n\n// submitted is not changed\n(@request.body.submitted:isset = false || submitted = @request.body.submitted) &&\n\n// no approval properties are submitted\n(@request.body.approved:isset = false || approved = @request.body.approved) &&\n(@request.body.approver:isset = false || approver = @request.body.approver) &&\n\n// no committed properties are submitted\n(@request.body.committed:isset = false || committed = @request.body.committed) &&\n(@request.body.committer:isset = false || committer = @request.body.committer) &&\n(@request.body.committed_week_ending:isset = false || committed_week_ending = @request.body.committed_week_ending) &&\n\n// if present, vendor is active\n(@request.body.vendor = \"\" || @request.body.vendor.status = \"Active\") &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // the job is unchanged, compare the new category to job\n  ( @request.body.job:isset = false && @request.body.category.job = job ) ||\n  // the job has changed, compare the new category to the new job\n  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||\n  @request.body.category = \"\"\n)"
		}`), &collection); err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(15, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "6ocqzyet",
			"max": 0,
			"min": 0,
			"name": "pay_period_ending",
			"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("o1vpz1mm7qsfoyy")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"createRule": "// the caller is authenticated\n@request.auth.id != \"\" &&\n\n// the uid is equal to the authenticated user's id\n@request.body.uid = @request.auth.id &&\n\n// no rejection properties are submitted\n@request.body.rejector:isset = false &&\n@request.body.rejected:isset = false &&\n@request.body.rejection_reason:isset = false &&\n\n// no approval properties are submitted\n@request.body.approved:isset = false &&\n@request.body.approver:isset = false &&\n\n// no committed properties are submitted\n@request.body.committed:isset = false &&\n@request.body.committer:isset = false &&\n@request.body.committed_week_ending:isset = false &&\n\n// if present, vendor is active\n(@request.body.vendor = \"\" || @request.body.vendor.status = \"Active\") &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // compare the new category to the new job\n  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||\n  @request.body.category = \"\"\n)",
			"updateRule": "// only the creator can update the record\nuid = @request.auth.id &&\n\n// the uid must not change\n@request.body.uid:changed = false &&\n\n// no rejection properties are submitted\n(@request.body.rejector:isset = false || rejector = @request.body.rejector) &&\n(@request.body.rejected:isset = false || rejected = @request.body.rejected) &&\n(@request.body.rejection_reason:isset = false || rejection_reason = @request.body.rejection_reason) &&\n\n// submitted is not changed\n(@request.body.submitted:isset = false || submitted = @request.body.submitted) &&\n\n// no approval properties are submitted\n(@request.body.approved:isset = false || approved = @request.body.approved) &&\n(@request.body.approver:isset = false || approver = @request.body.approver) &&\n\n// no committed properties are submitted\n(@request.body.committed:isset = false || committed = @request.body.committed) &&\n(@request.body.committer:isset = false || committer = @request.body.committer) &&\n(@request.body.committed_week_ending:isset = false || committed_week_ending = @request.body.committed_week_ending) &&\n\n// if present, vendor is active\n(@request.body.vendor = \"\" || @request.body.vendor.status = \"Active\") &&\n\n// if present, the category belongs to the job, otherwise is blank\n(\n  // the job is unchanged, compare the new category to job\n  ( @request.body.job:isset = false && @request.body.category.job = job ) ||\n  // the job has changed, compare the new category to the new job\n  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||\n  @request.body.category = \"\"\n)"
		}`), &collection); err != nil {
			return err
		}

		// update field
		if err := collection.Fields.AddMarshaledJSONAt(15, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "6ocqzyet",
			"max": 0,
			"min": 0,
			"name": "pay_period_ending",
			"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
			"presentable": false,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
