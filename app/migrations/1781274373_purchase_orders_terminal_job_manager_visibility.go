package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	purchaseOrdersTerminalJobManagerVisibilityOldRule = "(status = \"Active\" && @request.auth.id != \"\") ||\n(\n  (status = \"Cancelled\" || status = \"Closed\") &&\n  (\n    @request.auth.id = uid ||\n    @request.auth.id = approver ||\n    @request.auth.id = second_approver ||\n    @request.auth.user_claims_via_uid.cid.name ?= \"report\"\n  )\n) ||\n(\n  status = \"Unapproved\" &&\n  (\n    @request.auth.id = uid ||\n    @request.auth.id = approver ||\n    @request.auth.id = rejector ||\n    (approved != \"\" && second_approval = \"\" && @request.auth.id = priority_second_approver)\n  )\n)"
	purchaseOrdersTerminalJobManagerVisibilityNewRule = "(status = \"Active\" && @request.auth.id != \"\") ||\n(\n  (status = \"Cancelled\" || status = \"Closed\") &&\n  (\n    @request.auth.id = uid ||\n    @request.auth.id = approver ||\n    @request.auth.id = second_approver ||\n    job.manager = @request.auth.id ||\n    job.alternate_manager = @request.auth.id ||\n    @request.auth.user_claims_via_uid.cid.name ?= \"report\"\n  )\n) ||\n(\n  status = \"Unapproved\" &&\n  (\n    @request.auth.id = uid ||\n    @request.auth.id = approver ||\n    @request.auth.id = rejector ||\n    (approved != \"\" && second_approval = \"\" && @request.auth.id = priority_second_approver)\n  )\n)"
)

func init() {
	m.Register(func(app core.App) error {
		if err := updateRule(app, "purchase_orders", "listRule", purchaseOrdersTerminalJobManagerVisibilityNewRule); err != nil {
			return err
		}
		if err := updateRule(app, "purchase_orders", "viewRule", purchaseOrdersTerminalJobManagerVisibilityNewRule); err != nil {
			return err
		}
		if err := updateRule(app, "purchase_orders_augmented", "listRule", purchaseOrdersTerminalJobManagerVisibilityNewRule); err != nil {
			return err
		}
		return updateRule(app, "purchase_orders_augmented", "viewRule", purchaseOrdersTerminalJobManagerVisibilityNewRule)
	}, func(app core.App) error {
		if err := updateRule(app, "purchase_orders", "listRule", purchaseOrdersTerminalJobManagerVisibilityOldRule); err != nil {
			return err
		}
		if err := updateRule(app, "purchase_orders", "viewRule", purchaseOrdersTerminalJobManagerVisibilityOldRule); err != nil {
			return err
		}
		if err := updateRule(app, "purchase_orders_augmented", "listRule", purchaseOrdersTerminalJobManagerVisibilityOldRule); err != nil {
			return err
		}
		return updateRule(app, "purchase_orders_augmented", "viewRule", purchaseOrdersTerminalJobManagerVisibilityOldRule)
	})
}
