package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const purchaseOrderVisibilityRuleV3ActionQueueGatedPrevious = `(status = "Active" && @request.auth.id != "") ||
(
  (status = "Cancelled" || status = "Closed") &&
  (
    @request.auth.id = uid ||
    @request.auth.id = approver ||
    @request.auth.id = second_approver ||
    @request.auth.user_claims_via_uid.cid.name ?= "report"
  )
) ||
(
  status = "Unapproved" &&
  (
    @request.auth.id = uid ||
    (approved = "" && @request.auth.id = approver) ||
    (approved != "" && second_approval = "" && @request.auth.id = priority_second_approver) ||
    (
      approved != "" &&
      second_approval = "" &&
      @request.auth.user_claims_via_uid.cid.name ?= "po_approver" &&
      (
        @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.divisions:length = 0 ||
        @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.divisions:each ?= division
      ) &&
      @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.max_amount >= approval_total
    )
  )
)`

const purchaseOrderVisibilityRuleV4DirectOnly = `(status = "Active" && @request.auth.id != "") ||
(
  (status = "Cancelled" || status = "Closed") &&
  (
    @request.auth.id = uid ||
    @request.auth.id = approver ||
    @request.auth.id = second_approver ||
    @request.auth.user_claims_via_uid.cid.name ?= "report"
  )
) ||
(
  status = "Unapproved" &&
  (
    @request.auth.id = uid ||
    (approved = "" && @request.auth.id = approver) ||
    (approved != "" && second_approval = "" && @request.auth.id = priority_second_approver)
  )
)`

func init() {
	m.Register(func(app core.App) error {
		return setPurchaseOrderDirectVisibilityRule(app, purchaseOrderVisibilityRuleV4DirectOnly)
	}, func(app core.App) error {
		return setPurchaseOrderDirectVisibilityRule(app, purchaseOrderVisibilityRuleV3ActionQueueGatedPrevious)
	})
}

func setPurchaseOrderDirectVisibilityRule(app core.App, rule string) error {
	if err := setCollectionDirectVisibilityRule(app, "purchase_orders", rule); err != nil {
		return err
	}

	if err := setCollectionDirectVisibilityRule(app, "purchase_orders_augmented", rule); err != nil {
		return err
	}

	return nil
}

func setCollectionDirectVisibilityRule(app core.App, name string, rule string) error {
	collection, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		return err
	}

	collection.ListRule = stringPtrDirect(rule)
	collection.ViewRule = stringPtrDirect(rule)
	return app.SaveNoValidate(collection)
}

func stringPtrDirect(v string) *string {
	return &v
}
