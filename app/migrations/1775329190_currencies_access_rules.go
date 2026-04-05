package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	currenciesReadRule      = "@request.auth.id != \"\""
	currenciesAdminOnlyRule = "@request.auth.user_claims_via_uid.cid.name ?= \"admin\""
)

func init() {
	m.Register(func(app core.App) error {
		if err := updateRule(app, "currencies", "listRule", currenciesReadRule); err != nil {
			return err
		}
		if err := updateRule(app, "currencies", "viewRule", currenciesReadRule); err != nil {
			return err
		}
		if err := updateRule(app, "currencies", "createRule", currenciesAdminOnlyRule); err != nil {
			return err
		}
		if err := updateRule(app, "currencies", "updateRule", currenciesAdminOnlyRule); err != nil {
			return err
		}
		return updateRule(app, "currencies", "deleteRule", currenciesAdminOnlyRule)
	}, func(app core.App) error {
		if err := updateRule(app, "currencies", "listRule", ""); err != nil {
			return err
		}
		if err := updateRule(app, "currencies", "viewRule", ""); err != nil {
			return err
		}
		if err := updateRule(app, "currencies", "createRule", ""); err != nil {
			return err
		}
		if err := updateRule(app, "currencies", "updateRule", ""); err != nil {
			return err
		}
		return updateRule(app, "currencies", "deleteRule", "")
	})
}
