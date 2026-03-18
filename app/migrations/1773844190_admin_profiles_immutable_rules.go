package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	adminProfilesSharedUpdateRuleWithImmutableFields = "@request.auth.id != \"\" &&\n@request.body.id:changed = false &&\n@request.body.uid:changed = false &&\n@request.body.legacy_uid:changed = false &&\n(\n  @request.auth.user_claims_via_uid.cid.name ?= 'admin' ||\n  (\n    @request.auth.user_claims_via_uid.cid.name ?= 'hr' &&\n    @request.body.work_week_hours:changed = false &&\n    @request.body.untracked_time_off:changed = false &&\n    @request.body.opening_date:changed = false &&\n    @request.body.opening_op:changed = false &&\n    @request.body.opening_ov:changed = false &&\n    @request.body._imported:changed = false\n  )\n)"
)

func init() {
	m.Register(func(app core.App) error {
		return updateRule(app, "admin_profiles", "updateRule", adminProfilesSharedUpdateRuleWithImmutableFields)
	}, func(app core.App) error {
		return updateRule(app, "admin_profiles", "updateRule", adminProfilesHrUpdateRule)
	})
}
