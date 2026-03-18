package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	// This migration broadens purchase_order list/view participant visibility so
	// PocketBase collection access better matches the app's practical read model.
	//
	// Specifically:
	// - assigned approvers can still read an Unapproved PO after first approval
	// - rejectors can still read rejected Unapproved POs
	//
	// This intentionally does NOT fully align PocketBase collection rules with
	// the broader /api/purchase_orders/visible* routes. Remaining discrepancies to
	// evaluate later include:
	// - policy-based visibility for eligible non-priority second-stage approvers
	// - any rejected/non-rejected subtleties beyond the explicit participant
	//   broadening done here
	purchaseOrdersParticipantVisibilityOldRule = "(status = \"Active\" && @request.auth.id != \"\") ||\n(\n  (status = \"Cancelled\" || status = \"Closed\") &&\n  (\n    @request.auth.id = uid ||\n    @request.auth.id = approver ||\n    @request.auth.id = second_approver ||\n    @request.auth.user_claims_via_uid.cid.name ?= \"report\"\n  )\n) ||\n(\n  status = \"Unapproved\" &&\n  (\n    @request.auth.id = uid ||\n    (approved = \"\" && @request.auth.id = approver) ||\n    (approved != \"\" && second_approval = \"\" && @request.auth.id = priority_second_approver)\n  )\n)"
	purchaseOrdersParticipantVisibilityNewRule = "(status = \"Active\" && @request.auth.id != \"\") ||\n(\n  (status = \"Cancelled\" || status = \"Closed\") &&\n  (\n    @request.auth.id = uid ||\n    @request.auth.id = approver ||\n    @request.auth.id = second_approver ||\n    @request.auth.user_claims_via_uid.cid.name ?= \"report\"\n  )\n) ||\n(\n  status = \"Unapproved\" &&\n  (\n    @request.auth.id = uid ||\n    @request.auth.id = approver ||\n    @request.auth.id = rejector ||\n    (approved != \"\" && second_approval = \"\" && @request.auth.id = priority_second_approver)\n  )\n)"
)

func init() {
	m.Register(func(app core.App) error {
		if err := updateRule(app, "purchase_orders", "listRule", purchaseOrdersParticipantVisibilityNewRule); err != nil {
			return err
		}
		if err := updateRule(app, "purchase_orders", "viewRule", purchaseOrdersParticipantVisibilityNewRule); err != nil {
			return err
		}
		if err := updateRule(app, "purchase_orders_augmented", "listRule", purchaseOrdersParticipantVisibilityNewRule); err != nil {
			return err
		}
		return updateRule(app, "purchase_orders_augmented", "viewRule", purchaseOrdersParticipantVisibilityNewRule)
	}, func(app core.App) error {
		if err := updateRule(app, "purchase_orders", "listRule", purchaseOrdersParticipantVisibilityOldRule); err != nil {
			return err
		}
		if err := updateRule(app, "purchase_orders", "viewRule", purchaseOrdersParticipantVisibilityOldRule); err != nil {
			return err
		}
		if err := updateRule(app, "purchase_orders_augmented", "listRule", purchaseOrdersParticipantVisibilityOldRule); err != nil {
			return err
		}
		return updateRule(app, "purchase_orders_augmented", "viewRule", purchaseOrdersParticipantVisibilityOldRule)
	})
}
