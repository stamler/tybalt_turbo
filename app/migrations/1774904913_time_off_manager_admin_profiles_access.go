package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	adminProfilesTimeOffManagerUpdateRule    = "@request.auth.id != \"\" &&\n@request.body.id:changed = false &&\n@request.body.uid:changed = false &&\n@request.body.legacy_uid:changed = false &&\n(\n  @request.auth.user_claims_via_uid.cid.name ?= 'admin' ||\n  (\n    (\n      @request.auth.user_claims_via_uid.cid.name ?= 'hr' ||\n      @request.auth.user_claims_via_uid.cid.name ?= 'time_off_manager'\n    ) &&\n    @request.body.work_week_hours:changed = false &&\n    @request.body.untracked_time_off:changed = false &&\n    @request.body._imported:changed = false &&\n    (\n      (\n        @request.auth.user_claims_via_uid.cid.name ?= 'hr' &&\n        @request.auth.user_claims_via_uid.cid.name ?= 'time_off_manager'\n      ) ||\n      (\n        @request.auth.user_claims_via_uid.cid.name ?= 'hr' &&\n        @request.body.opening_date:changed = false &&\n        @request.body.opening_op:changed = false &&\n        @request.body.opening_ov:changed = false\n      ) ||\n      (\n        @request.auth.user_claims_via_uid.cid.name ?= 'time_off_manager' &&\n        @request.body.active:changed = false &&\n        @request.body.allow_personal_reimbursement:changed = false &&\n        @request.body.default_branch:changed = false &&\n        @request.body.default_charge_out_rate:changed = false &&\n        @request.body.job_title:changed = false &&\n        @request.body.mobile_phone:changed = false &&\n        @request.body.off_rotation_permitted:changed = false &&\n        @request.body.payroll_id:changed = false &&\n        @request.body.personal_vehicle_insurance_expiry:changed = false &&\n        @request.body.salary:changed = false &&\n        @request.body.skip_min_time_check:changed = false &&\n        @request.body.time_sheet_expected:changed = false\n      )\n    )\n  )\n)"
	adminProfilesAugmentedTimeOffManagerRule = "@request.auth.id != \"\" &&\n(\n  @request.auth.user_claims_via_uid.cid.name ?= 'admin' ||\n  @request.auth.user_claims_via_uid.cid.name ?= 'hr' ||\n  @request.auth.user_claims_via_uid.cid.name ?= 'time_off_manager'\n)"
)

func init() {
	m.Register(func(app core.App) error {
		if err := updateRule(app, "admin_profiles", "updateRule", adminProfilesTimeOffManagerUpdateRule); err != nil {
			return err
		}
		if err := updateRule(app, "admin_profiles_augmented", "listRule", adminProfilesAugmentedTimeOffManagerRule); err != nil {
			return err
		}
		return updateRule(app, "admin_profiles_augmented", "viewRule", adminProfilesAugmentedTimeOffManagerRule)
	}, func(app core.App) error {
		if err := updateRule(app, "admin_profiles", "updateRule", adminProfilesSharedUpdateRuleWithImmutableFields); err != nil {
			return err
		}
		if err := updateRule(app, "admin_profiles_augmented", "listRule", adminProfilesAugmentedHrRule); err != nil {
			return err
		}
		return updateRule(app, "admin_profiles_augmented", "viewRule", adminProfilesAugmentedHrRule)
	})
}
