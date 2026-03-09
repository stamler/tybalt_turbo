package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const legacyPOCreateUpdateClaimName = "legacy_po_create_update"
const legacyPOCreateUpdateClaimDescription = "Can create and update legacy manually-entered purchase orders while the transition flag is enabled"
const legacyPOCreateUpdateFieldID = "bool1773000000"
const purchaseOrdersConfigDescriptionBeforeLegacyPOCreateUpdate = "For purchase_orders which have a priority_second_approver set, how long to wait before showing the PO to all qualified second approvers."
const purchaseOrdersConfigDescriptionWithLegacyPOCreateUpdate = "Controls purchase order workflow behavior, including second-stage timeout handling and the hidden legacy PO create/update flow."

const purchaseOrdersVisibilityRuleBeforeLegacyCreateUpdate = `(status = "Active" && @request.auth.id != "") ||
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

const purchaseOrdersCreateRuleBeforeLegacyCreateUpdate = `// the caller is authenticated
@request.auth.id != "" &&

// no po_number is submitted
(@request.body.po_number:isset = false || @request.body.po_number = "") &&

// status is Unapproved
@request.body.status = "Unapproved" &&

// the uid is missing or is equal to the authenticated user's id
(@request.body.uid:isset = false || @request.body.uid = @request.auth.id) &&

// no rejection properties are submitted
@request.body.rejector:isset = false &&
@request.body.rejected:isset = false &&
@request.body.rejection_reason:isset = false &&

// approved isn't set. We check that the approver has the appropriate claim and divisions in payload in hooks, commenting out the previous check for po_approver here. 
@request.body.approved:isset = false &&
//@request.body.approver.user_claims_via_uid.cid.name ?= 'po_approver' &&

// no second approver properties are submitted
@request.body.second_approver:isset = false &&
@request.body.second_approval:isset = false &&

// no cancellation properties are submitted
@request.body.cancelled:isset = false &&
@request.body.canceller:isset = false &&

// no closed properties are submitted
@request.body.closed:isset = false &&
@request.body.closer:isset = false &&
@request.body.closed_by_system:isset = false &&

// vendor is active (disabled, perform this in the hook for better error messages)
// @request.body.vendor.status = "Active" &&

// if present, the category belongs to the job, otherwise is blank
(
  // compare the new category to the new job
  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||
  @request.body.category = ""
)`

const purchaseOrdersCreateRuleWithLegacyCreateUpdate = `// the caller is authenticated
@request.auth.id != "" &&

// standard flow cannot set legacy_manual_entry
(@request.body.legacy_manual_entry:isset = false || @request.body.legacy_manual_entry = false) &&

// no po_number is submitted
(@request.body.po_number:isset = false || @request.body.po_number = "") &&

// status is Unapproved
@request.body.status = "Unapproved" &&

// the uid is missing or is equal to the authenticated user's id
(@request.body.uid:isset = false || @request.body.uid = @request.auth.id) &&

// no rejection properties are submitted
@request.body.rejector:isset = false &&
@request.body.rejected:isset = false &&
@request.body.rejection_reason:isset = false &&

// approved isn't set. We check that the approver has the appropriate claim and divisions in payload in hooks, commenting out the previous check for po_approver here. 
@request.body.approved:isset = false &&
//@request.body.approver.user_claims_via_uid.cid.name ?= 'po_approver' &&

// no second approver properties are submitted
@request.body.second_approver:isset = false &&
@request.body.second_approval:isset = false &&

// no cancellation properties are submitted
@request.body.cancelled:isset = false &&
@request.body.canceller:isset = false &&

// no closed properties are submitted
@request.body.closed:isset = false &&
@request.body.closer:isset = false &&
@request.body.closed_by_system:isset = false &&

// vendor is active (disabled, perform this in the hook for better error messages)
// @request.body.vendor.status = "Active" &&

// if present, the category belongs to the job, otherwise is blank
(
  // compare the new category to the new job
  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||
  @request.body.category = ""
)`

const purchaseOrdersUpdateRuleWithLegacyCreateUpdate = `legacy_manual_entry = false &&
(@request.body.legacy_manual_entry:isset = false || @request.body.legacy_manual_entry = false) &&

// only the creator can update the record
uid = @request.auth.id &&

// the uid must not change
@request.body.uid:changed = false &&

// status is Unapproved and second approval has not been performed
status = 'Unapproved' &&
second_approval = "" &&

// no po_number is submitted
(@request.body.po_number:isset = false || po_number = @request.body.po_number) &&

// no rejection properties are submitted
(@request.body.rejector:isset = false || rejector = @request.body.rejector) &&
(@request.body.rejected:isset = false || rejected = @request.body.rejected) &&
(@request.body.rejection_reason:isset = false || rejection_reason = @request.body.rejection_reason) &&

// approved is unchanged
(@request.body.approved:isset = false || approved = @request.body.approved) &&

// no second approver properties are submitted
(@request.body.second_approver:isset = false || second_approver = @request.body.second_approver) &&
(@request.body.second_approval:isset = false || second_approval = @request.body.second_approval) &&

// no cancellation properties are submitted
(@request.body.cancelled:isset = false || cancelled = @request.body.cancelled) &&
(@request.body.canceller:isset = false || canceller = @request.body.canceller) &&

// no closed properties are submitted
(@request.body.closed:isset = false || closed = @request.body.closed) &&
(@request.body.closer:isset = false || closer = @request.body.closer) &&
(@request.body.closed_by_system:isset = false || closed_by_system = @request.body.closed_by_system) &&

// vendor is active (disabled, perform this in the hook for better error messages)
// @request.body.vendor.status = "Active" &&

// if present, the category belongs to the job, otherwise is blank
(
  // the job is unchanged, compare the new category to job
  ( @request.body.job:isset = false && @request.body.category.job = job ) ||
  // the job has changed, compare the new category to the new job
  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||
  @request.body.category = ""
 )`

func init() {
	m.Register(func(app core.App) error {
		if err := addLegacyPOCreateUpdateClaim(app); err != nil {
			return err
		}
		if err := ensurePurchaseOrdersLegacyConfig(app); err != nil {
			return err
		}
		if err := addLegacyPurchaseOrderField(app); err != nil {
			return err
		}
		if err := setLegacyPurchaseOrderRules(
			app,
			purchaseOrdersCreateRuleWithLegacyCreateUpdate,
			purchaseOrdersUpdateRuleWithLegacyCreateUpdate,
			purchaseOrdersVisibilityRuleBeforeLegacyCreateUpdate,
		); err != nil {
			return err
		}
		return nil
	}, func(app core.App) error {
		if err := removeLegacyPurchaseOrderField(app); err != nil {
			return err
		}
		if err := revertPurchaseOrdersLegacyConfig(app); err != nil {
			return err
		}
		if err := setLegacyPurchaseOrderRules(
			app,
			purchaseOrdersCreateRuleBeforeLegacyCreateUpdate,
			purchaseOrdersUpdateRuleAfterOwnerHardening,
			purchaseOrdersVisibilityRuleBeforeLegacyCreateUpdate,
		); err != nil {
			return err
		}
		return removeLegacyPOCreateUpdateClaim(app)
	})
}

func addLegacyPOCreateUpdateClaim(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("claims")
	if err != nil {
		return err
	}

	record, err := app.FindFirstRecordByData("claims", "name", legacyPOCreateUpdateClaimName)
	if err != nil || record == nil {
		record = core.NewRecord(collection)
		record.Set("name", legacyPOCreateUpdateClaimName)
	}
	record.Set("description", legacyPOCreateUpdateClaimDescription)
	return app.Save(record)
}

func removeLegacyPOCreateUpdateClaim(app core.App) error {
	record, err := app.FindFirstRecordByData("claims", "name", legacyPOCreateUpdateClaimName)
	if err != nil || record == nil {
		return nil
	}
	return app.Delete(record)
}

func ensurePurchaseOrdersLegacyConfig(app core.App) error {
	record, err := app.FindFirstRecordByData("app_config", "key", "purchase_orders")
	if err != nil || record == nil {
		collection, collectionErr := app.FindCollectionByNameOrId("app_config")
		if collectionErr != nil {
			return collectionErr
		}
		record = core.NewRecord(collection)
		record.Set("key", "purchase_orders")
	}

	config := map[string]any{}
	if raw := record.GetString("value"); raw != "" {
		_ = json.Unmarshal([]byte(raw), &config)
	}

	timeoutRaw, ok := config["second_stage_timeout_hours"]
	validTimeout := false
	if ok {
		switch v := timeoutRaw.(type) {
		case float64:
			validTimeout = v > 0
		case int:
			validTimeout = v > 0
		}
	}
	if !validTimeout {
		config["second_stage_timeout_hours"] = 24
	}
	config["enable_legacy_po_create_update"] = false

	encoded, marshalErr := json.Marshal(config)
	if marshalErr != nil {
		return marshalErr
	}

	record.Set("value", string(encoded))
	record.Set("description", purchaseOrdersConfigDescriptionWithLegacyPOCreateUpdate)
	return app.Save(record)
}

func revertPurchaseOrdersLegacyConfig(app core.App) error {
	record, err := app.FindFirstRecordByData("app_config", "key", "purchase_orders")
	if err != nil || record == nil {
		return nil
	}

	config := map[string]any{}
	if raw := record.GetString("value"); raw != "" {
		_ = json.Unmarshal([]byte(raw), &config)
	}

	delete(config, "enable_legacy_po_create_update")
	if _, ok := config["second_stage_timeout_hours"]; !ok {
		config["second_stage_timeout_hours"] = 24
	}

	encoded, marshalErr := json.Marshal(config)
	if marshalErr != nil {
		return marshalErr
	}

	record.Set("value", string(encoded))
	record.Set("description", purchaseOrdersConfigDescriptionBeforeLegacyPOCreateUpdate)
	return app.Save(record)
}

func addLegacyPurchaseOrderField(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("purchase_orders")
	if err != nil {
		return err
	}
	if collection.Fields.GetByName("legacy_manual_entry") == nil {
		if err := collection.Fields.AddMarshaledJSONAt(37, []byte(`{
			"hidden": false,
			"id": "`+legacyPOCreateUpdateFieldID+`",
			"name": "legacy_manual_entry",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}
	}
	return app.Save(collection)
}

func removeLegacyPurchaseOrderField(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("purchase_orders")
	if err != nil {
		return err
	}
	collection.Fields.RemoveById(legacyPOCreateUpdateFieldID)
	return app.Save(collection)
}

func setLegacyPurchaseOrderRules(app core.App, createRule string, updateRule string, visibilityRule string) error {
	collection, err := app.FindCollectionByNameOrId("purchase_orders")
	if err != nil {
		return err
	}
	collection.CreateRule = stringPtr(createRule)
	collection.UpdateRule = stringPtr(updateRule)
	collection.ListRule = stringPtr(visibilityRule)
	collection.ViewRule = stringPtr(visibilityRule)
	return app.SaveNoValidate(collection)
}
