package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	adminProfilesOldUpdateRule = "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'admin'"
	adminProfilesHrUpdateRule  = "@request.auth.id != \"\" &&\n(\n  @request.auth.user_claims_via_uid.cid.name ?= 'admin' ||\n  (\n    @request.auth.user_claims_via_uid.cid.name ?= 'hr' &&\n    @request.body.uid:changed = false &&\n    @request.body.work_week_hours:changed = false &&\n    @request.body.untracked_time_off:changed = false &&\n    @request.body.mobile_phone:changed = false &&\n    @request.body.job_title:changed = false &&\n    @request.body.opening_date:changed = false &&\n    @request.body.opening_op:changed = false &&\n    @request.body.opening_ov:changed = false &&\n    @request.body.default_branch:changed = false &&\n    @request.body.legacy_uid:changed = false &&\n    @request.body.active:changed = false &&\n    @request.body._imported:changed = false\n  )\n)"

	adminProfilesAugmentedOldRule = "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'admin'"
	adminProfilesAugmentedHrRule  = "@request.auth.id != \"\" &&\n(\n  @request.auth.user_claims_via_uid.cid.name ?= 'admin' ||\n  @request.auth.user_claims_via_uid.cid.name ?= 'hr'\n)"
)

func init() {
	m.Register(func(app core.App) error {
		if err := updateRule(app, "admin_profiles", "updateRule", adminProfilesHrUpdateRule); err != nil {
			return err
		}
		if err := updateRule(app, "admin_profiles_augmented", "listRule", adminProfilesAugmentedHrRule); err != nil {
			return err
		}
		return updateRule(app, "admin_profiles_augmented", "viewRule", adminProfilesAugmentedHrRule)
	}, func(app core.App) error {
		if err := updateRule(app, "admin_profiles", "updateRule", adminProfilesOldUpdateRule); err != nil {
			return err
		}
		if err := updateRule(app, "admin_profiles_augmented", "listRule", adminProfilesAugmentedOldRule); err != nil {
			return err
		}
		return updateRule(app, "admin_profiles_augmented", "viewRule", adminProfilesAugmentedOldRule)
	})
}

func updateRule(app core.App, collectionName string, ruleName string, rule string) error {
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return err
	}

	payload := map[string]string{ruleName: rule}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(raw, &collection); err != nil {
		return err
	}

	return app.Save(collection)
}
