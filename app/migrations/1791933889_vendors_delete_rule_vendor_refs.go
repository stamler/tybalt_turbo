package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The original vendors deleteRule compared purchase_orders.job and
// expenses.job with the vendor id, so it never blocked a delete. PocketBase
// then deleted referenced vendors and blanked the vendor field on their
// purchase orders and expenses. Compare the vendor relations instead.
const (
	vendorsOldDeleteRule = "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'payables_admin' &&\n\n// prevent deletion of vendors if there are referencing purchase orders\n@collection.purchase_orders.job != id &&\n\n// prevent deletion of vendors if there are referencing expenses\n@collection.expenses.job != id"
	vendorsNewDeleteRule = "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'payables_admin' &&\n\n// prevent deletion of vendors if there are referencing purchase orders\n@collection.purchase_orders.vendor != id &&\n\n// prevent deletion of vendors if there are referencing expenses\n@collection.expenses.vendor != id"
)

func init() {
	m.Register(func(app core.App) error {
		return updateRule(app, "vendors", "deleteRule", vendorsNewDeleteRule)
	}, func(app core.App) error {
		return updateRule(app, "vendors", "deleteRule", vendorsOldDeleteRule)
	})
}
