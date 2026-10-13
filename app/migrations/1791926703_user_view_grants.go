package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The users viewRule used to reach expenses and purchase_orders through
// @collection references. PocketBase joins each @collection reference to the
// main query without a join condition, so checking one users row read every
// expense paired with every purchase order. Replacing them with back-relations
// (expenses_via_uid, purchase_orders_via_approver, ...) would still multiply the
// matched rows of four joins for users with long histories.
//
// user_view_grants lists each grant once instead, so the rule needs one
// back-relation join on uid. A row means either:
//   - reason "expense_approver": viewer approved at least one of uid's
//     committed expenses, or
//   - reason "active_po": uid is the uid, approver, or second_approver of at
//     least one Active purchase order. Any signed-in caller may view uid.
//
// PocketBase infers view field types from the outer SELECT only. Selecting
// users ids makes uid and viewer relations to users, which the back-relation
// needs. The grants are a parenthesized subquery so the parser does not read
// the UNION branches as columns. viewer is NULL on active_po rows.
const userViewGrantsQuery = `SELECT
  g.id AS id,
  u.id AS uid,
  v.id AS viewer,
  CAST(g.reason AS TEXT) AS reason
FROM (
  SELECT po.uid AS id, po.uid AS uid, '' AS viewer, 'active_po' AS reason
  FROM purchase_orders po
  WHERE po.status = 'Active' AND po.uid != ''
  UNION
  SELECT po.approver, po.approver, '', 'active_po'
  FROM purchase_orders po
  WHERE po.status = 'Active' AND po.approver != ''
  UNION
  SELECT po.second_approver, po.second_approver, '', 'active_po'
  FROM purchase_orders po
  WHERE po.status = 'Active' AND po.second_approver != ''
  UNION
  SELECT e.uid || e.approver, e.uid, e.approver, 'expense_approver'
  FROM expenses e
  WHERE e.committed != '' AND e.uid != '' AND e.approver != ''
) g
JOIN users u ON u.id = g.uid
LEFT JOIN users v ON v.id = g.viewer`

const usersViewRuleWithCollectionJoins = "// The caller is logged in and either\n@request.auth.id != \"\" &&\n(\n  // 1. The caller id is equal to the record id or\n  @request.auth.id = id ||\n  // 2. The record has the po_approver claim or\n  user_claims_via_uid.cid.name ?= 'po_approver' ||\n  // 3. The record has the tapr claim or\n  user_claims_via_uid.cid.name ?= 'tapr' ||\n  // 4. The caller has the tapr claim\n  @request.auth.user_claims_via_uid.cid.name ?= 'tapr' ||\n  // 5. The caller is the approver of at least one of users committed expenses\n  (\n    @collection.expenses.approver ?= @request.auth.id &&\n    @collection.expenses.uid ?= id &&\n    @collection.expenses.committed ?!= ''\n  ) ||\n  // 6. The record is the uid, approver, or second_approver of 1+ active purchase order\n  (\n    (\n      @collection.purchase_orders.uid ?= id ||\n      @collection.purchase_orders.approver ?= id ||\n      @collection.purchase_orders.second_approver ?= id\n    ) &&\n    @collection.purchase_orders.status ?= \"Active\"\n  )\n)"

const usersViewRuleWithViewGrants = "// The caller is logged in and either\n@request.auth.id != \"\" &&\n(\n  // 1. The caller id is equal to the record id or\n  @request.auth.id = id ||\n  // 2. The record has the po_approver claim or\n  user_claims_via_uid.cid.name ?= 'po_approver' ||\n  // 3. The record has the tapr claim or\n  user_claims_via_uid.cid.name ?= 'tapr' ||\n  // 4. The caller is the approver of at least one of users committed expenses or\n  user_view_grants_via_uid.viewer ?= @request.auth.id ||\n  // 5. The record is the uid, approver, or second_approver of 1+ active purchase order or\n  user_view_grants_via_uid.reason ?= 'active_po' ||\n  // 6. The caller has the tapr claim. Keep this last: PocketBase joins tables\n  // in the order the rule names them, and each claim the caller holds repeats\n  // every later join.\n  @request.auth.user_claims_via_uid.cid.name ?= 'tapr'\n)"

const userViewGrantsCollectionId = "uviewgrants0001"

func init() {
	m.Register(func(app core.App) error {
		grants := core.NewViewCollection("user_view_grants")
		grants.Id = userViewGrantsCollectionId
		// Superusers only. Rules can still read it, but the API must not reveal
		// who approved whose expenses.
		grants.ListRule = nil
		grants.ViewRule = nil
		grants.ViewQuery = userViewGrantsQuery
		if err := app.Save(grants); err != nil {
			return err
		}

		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		users.ViewRule = pointerString(usersViewRuleWithViewGrants)
		return app.Save(users)
	}, func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		users.ViewRule = pointerString(usersViewRuleWithCollectionJoins)
		if err := app.Save(users); err != nil {
			return err
		}

		grants, err := app.FindCollectionByNameOrId(userViewGrantsCollectionId)
		if err != nil {
			return err
		}
		return app.Delete(grants)
	})
}
