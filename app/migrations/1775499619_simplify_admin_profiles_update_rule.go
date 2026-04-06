package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const adminProfilesAdminOnlyUpdateRuleWithImmutableFields = "@request.auth.id != \"\" &&\n@request.body.id:changed = false &&\n@request.body.uid:changed = false &&\n@request.body.legacy_uid:changed = false &&\n@request.auth.user_claims_via_uid.cid.name ?= 'admin'"

func init() {
	m.Register(func(app core.App) error {
		return updateRule(app, "admin_profiles", "updateRule", adminProfilesAdminOnlyUpdateRuleWithImmutableFields)
	}, func(app core.App) error {
		return updateRule(app, "admin_profiles", "updateRule", adminProfilesTimeOffManagerUpdateRule)
	})
}
