package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	timeEntriesCreateRuleBeforeTimeClaim = `@request.auth.id != "" &&
// if present, the category belongs to the job, otherwise is blank
(
  // compare the new category to the new job
  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||
  @request.body.category = ""
)`

	timeEntriesCreateRuleWithTimeClaim = `@request.auth.id != "" &&
@request.auth.user_claims_via_uid.cid.name ?= 'time' &&
// if present, the category belongs to the job, otherwise is blank
(
  // compare the new category to the new job
  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||
  @request.body.category = ""
)`

	timeEntriesDeleteRuleBeforeTimeClaim = `@request.auth.id = uid && tsid = ""`

	timeEntriesDeleteRuleWithTimeClaim = `@request.auth.id = uid &&
@request.auth.user_claims_via_uid.cid.name ?= 'time' &&
tsid = ""`

	timeEntriesUpdateRuleBeforeTimeClaim = `// the creating user can edit if the entry is not yet part of a timesheet
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

	timeEntriesUpdateRuleWithTimeClaim = `// the creating user can edit if the entry is not yet part of a timesheet
uid = @request.auth.id &&
@request.auth.user_claims_via_uid.cid.name ?= 'time' &&
tsid = "" &&

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
)

func init() {
	m.Register(func(app core.App) error {
		if err := updateRule(app, "time_entries", "createRule", timeEntriesCreateRuleWithTimeClaim); err != nil {
			return err
		}
		if err := updateRule(app, "time_entries", "deleteRule", timeEntriesDeleteRuleWithTimeClaim); err != nil {
			return err
		}
		return updateRule(app, "time_entries", "updateRule", timeEntriesUpdateRuleWithTimeClaim)
	}, func(app core.App) error {
		if err := updateRule(app, "time_entries", "createRule", timeEntriesCreateRuleBeforeTimeClaim); err != nil {
			return err
		}
		if err := updateRule(app, "time_entries", "deleteRule", timeEntriesDeleteRuleBeforeTimeClaim); err != nil {
			return err
		}
		return updateRule(app, "time_entries", "updateRule", timeEntriesUpdateRuleBeforeTimeClaim)
	})
}
