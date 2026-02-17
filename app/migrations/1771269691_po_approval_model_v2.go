package migrations

import (
	"database/sql"
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	userClaimsSummaryCollectionID   = "pbc_1771200001"
	userPOApproverProfileCollection = "pbc_1771200002"
	secondApprovalThresholdFieldID  = "number1771200001"
)

const purchaseOrderVisibilityRuleV2 = `(status = "Active" && @request.auth.id != "") ||
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
      approved < @yesterday &&
      @request.auth.user_claims_via_uid.cid.name ?= "po_approver" &&
      (
        @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.divisions:length = 0 ||
        @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.divisions:each ?= division
      ) &&
      @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.max_amount >= approval_total
    )
  )
)`

const purchaseOrdersAugmentedQueryV2 = `SELECT
  po.id,
  po.po_number,
  po.status,
  po.uid,
  po.type,
  po.date,
  po.end_date,
  po.frequency,
  po.division,
  po.description,
  po.total,
  po.payment_type,
  po.attachment,
  po.rejector,
  po.rejected,
  po.rejection_reason,
  po.approver,
  po.approved,
  po.second_approver,
  po.second_approval,
  po.canceller,
  po.cancelled,
  po.job,
  po.category,
  po.kind,
  po.vendor,
  po.parent_po,
  po.created,
  po.updated,
  po.closer,
  po.closed,
  po.closed_by_system,
  po.priority_second_approver,
  po.approval_total,
  (SELECT COUNT(*) FROM expenses WHERE expenses.purchase_order = po.id AND expenses.committed != "") AS committed_expenses_count,
  (p0.given_name || " " || p0.surname) AS uid_name,
  (p1.given_name || " " || p1.surname) AS approver_name,
  (p2.given_name || " " || p2.surname) AS second_approver_name,
  (p3.given_name || " " || p3.surname) AS priority_second_approver_name,
  (p4.given_name || " " || p4.surname) AS rejector_name,
  po2.po_number AS parent_po_number,
  v.name AS vendor_name,
  v.alias AS vendor_alias,
  j.number AS job_number,
  cl.name AS client_name,
  cl.id AS client_id,
  j.description AS job_description,
  d.code AS division_code,
  d.name AS division_name,
  c.name AS category_name
FROM purchase_orders AS po
LEFT JOIN profiles AS p0 ON po.uid = p0.uid
LEFT JOIN profiles AS p1 ON po.approver = p1.uid
LEFT JOIN profiles AS p2 ON po.second_approver = p2.uid
LEFT JOIN profiles AS p3 ON po.priority_second_approver = p3.uid
LEFT JOIN profiles AS p4 ON po.rejector = p4.uid
LEFT JOIN purchase_orders AS po2 ON po.parent_po = po2.id
LEFT JOIN vendors AS v ON po.vendor = v.id
LEFT JOIN jobs AS j ON po.job = j.id
LEFT JOIN divisions AS d ON po.division = d.id
LEFT JOIN categories AS c ON po.category = c.id
LEFT JOIN clients AS cl ON j.client = cl.id`

const pendingItemsForQualifiedPOSecondApproversQueryV2 = `WITH timeout_config AS (
  SELECT
    CASE
      WHEN json_valid(value) = 1
       AND json_type(value, '$.second_stage_timeout_hours') IN ('real', 'integer')
       AND json_extract(value, '$.second_stage_timeout_hours') > 0
      THEN json_extract(value, '$.second_stage_timeout_hours')
      ELSE 24
    END AS timeout_hours
  FROM app_config
  WHERE key = 'purchase_orders'
  LIMIT 1
),
cfg AS (
  SELECT COALESCE((SELECT timeout_hours FROM timeout_config), 24) AS timeout_hours
),
qualified_users AS (
  SELECT
    u.id AS user_id,
    pap.divisions,
    pap.max_amount,
    pap.project_max,
    pap.sponsorship_max,
    pap.staff_and_social_max,
    pap.media_and_event_max,
    pap.computer_max
  FROM users u
  JOIN admin_profiles ap ON ap.uid = u.id AND ap.active = 1
  JOIN user_claims uc ON u.id = uc.uid
  JOIN claims c ON uc.cid = c.id AND c.name = 'po_approver'
  JOIN po_approver_props pap ON uc.id = pap.user_claim
),
pos_needing_second_approval AS (
  SELECT
    po.id AS po_id,
    po.approval_total,
    po.division,
    po.job,
    COALESCE(ek.name, 'standard') AS kind_name,
    COALESCE(ek.second_approval_threshold, 0) AS second_approval_threshold
  FROM purchase_orders po
  LEFT JOIN expenditure_kinds ek ON po.kind = ek.id
  CROSS JOIN cfg
  WHERE
    po.approved != ''
    AND po.rejected = ''
    AND po.status = 'Unapproved'
    AND po.second_approval = ''
    AND po.approved < strftime('%Y-%m-%d %H:%M:%fZ', 'now', '-' || CAST(cfg.timeout_hours AS TEXT) || ' hours')
    AND COALESCE(ek.second_approval_threshold, 0) > 0
    AND po.approval_total > COALESCE(ek.second_approval_threshold, 0)
),
qualified_pairs AS (
  SELECT
    qu.user_id,
    po.po_id
  FROM qualified_users qu
  JOIN pos_needing_second_approval po
    ON (
      json_valid(qu.divisions)
      AND (
        json_array_length(qu.divisions) = 0
        OR EXISTS (SELECT 1 FROM json_each(qu.divisions) WHERE value = po.division)
      )
    )
   AND (
      CASE
        WHEN po.kind_name = 'standard' AND po.job != '' THEN COALESCE(qu.project_max, 0)
        WHEN po.kind_name = 'standard' THEN COALESCE(qu.max_amount, 0)
        WHEN po.kind_name = 'sponsorship' THEN COALESCE(qu.sponsorship_max, 0)
        WHEN po.kind_name = 'staff_and_social' THEN COALESCE(qu.staff_and_social_max, 0)
        WHEN po.kind_name = 'media_and_event' THEN COALESCE(qu.media_and_event_max, 0)
        WHEN po.kind_name = 'computer' THEN COALESCE(qu.computer_max, 0)
        ELSE 0
      END
   ) > po.second_approval_threshold
   AND (
      CASE
        WHEN po.kind_name = 'standard' AND po.job != '' THEN COALESCE(qu.project_max, 0)
        WHEN po.kind_name = 'standard' THEN COALESCE(qu.max_amount, 0)
        WHEN po.kind_name = 'sponsorship' THEN COALESCE(qu.sponsorship_max, 0)
        WHEN po.kind_name = 'staff_and_social' THEN COALESCE(qu.staff_and_social_max, 0)
        WHEN po.kind_name = 'media_and_event' THEN COALESCE(qu.media_and_event_max, 0)
        WHEN po.kind_name = 'computer' THEN COALESCE(qu.computer_max, 0)
        ELSE 0
      END
   ) >= po.approval_total
)
SELECT
  qp.user_id AS id,
  COUNT(qp.po_id) AS num_pos_qualified
FROM qualified_pairs qp
GROUP BY qp.user_id`

const userClaimsSummaryViewQuery = `SELECT
  u.id AS id,
  COALESCE(
    (
      SELECT json_group_array(c.name)
      FROM user_claims uc
      JOIN claims c ON c.id = uc.cid
      WHERE uc.uid = u.id
    ),
    '[]'
  ) AS claims
FROM users u`

const userPOApproverProfileViewQuery = `WITH po_props AS (
  SELECT
    uc.uid,
    pap.max_amount,
    pap.project_max,
    pap.sponsorship_max,
    pap.staff_and_social_max,
    pap.media_and_event_max,
    pap.computer_max,
    pap.divisions
  FROM user_claims uc
  JOIN claims c ON c.id = uc.cid AND c.name = 'po_approver'
  LEFT JOIN po_approver_props pap ON pap.user_claim = uc.id
)
SELECT
  u.id AS id,
  COALESCE(pp.max_amount, 0) AS max_amount,
  COALESCE(pp.project_max, 0) AS project_max,
  COALESCE(pp.sponsorship_max, 0) AS sponsorship_max,
  COALESCE(pp.staff_and_social_max, 0) AS staff_and_social_max,
  COALESCE(pp.media_and_event_max, 0) AS media_and_event_max,
  COALESCE(pp.computer_max, 0) AS computer_max,
  COALESCE(pp.divisions, '[]') AS divisions,
  COALESCE(
    (
      SELECT json_group_array(c.name)
      FROM user_claims uc
      JOIN claims c ON c.id = uc.cid
      WHERE uc.uid = u.id
    ),
    '[]'
  ) AS claims
FROM users u
LEFT JOIN po_props pp ON pp.uid = u.id`

func init() {
	m.Register(func(app core.App) error {
		if err := ensureSecondApprovalThresholdField(app); err != nil {
			return err
		}
		if err := ensurePurchaseOrdersConfig(app); err != nil {
			return err
		}
		if err := updatePurchaseOrderRules(app); err != nil {
			return err
		}
		if err := updatePurchaseOrdersAugmented(app); err != nil {
			return err
		}
		if err := updatePendingSecondApproverView(app); err != nil {
			return err
		}
		if err := deleteCollectionIfExists(app, "user_po_permission_data"); err != nil {
			return err
		}
		if err := deleteCollectionIfExists(app, "po_approval_thresholds"); err != nil {
			return err
		}
		if err := upsertUserClaimsSummary(app); err != nil {
			return err
		}
		if err := upsertUserPOApproverProfile(app); err != nil {
			return err
		}
		return nil
	}, func(app core.App) error {
		// Intentional no-op rollback for this migration.
		return nil
	})
}

func ensureSecondApprovalThresholdField(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("expenditure_kinds")
	if err != nil {
		return err
	}

	if !hasMigrationFieldByName(collection, "second_approval_threshold") {
		if err := collection.Fields.AddMarshaledJSONAt(len(collection.Fields), []byte(`{
			"hidden": false,
			"id": "`+secondApprovalThresholdFieldID+`",
			"max": null,
			"min": 0,
			"name": "second_approval_threshold",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}
		if err := app.Save(collection); err != nil {
			return err
		}
	}

	_, err = app.DB().NewQuery(`
		UPDATE expenditure_kinds
		SET second_approval_threshold = 0
		WHERE second_approval_threshold IS NULL
	`).Execute()
	if err != nil {
		return err
	}

	// Preserve historical behavior on cutover: if standard has no threshold set,
	// seed it from the legacy lowest global threshold (or 500 fallback).
	seedThreshold := float64(500)
	legacyTableExists, tableExistsErr := tableExists(app, "po_approval_thresholds")
	if tableExistsErr != nil {
		return tableExistsErr
	}
	if legacyTableExists {
		var legacy struct {
			Threshold sql.NullFloat64 `db:"threshold"`
		}
		if legacyErr := app.DB().NewQuery(`
			SELECT MIN(threshold) AS threshold
			FROM po_approval_thresholds
		`).One(&legacy); legacyErr != nil {
			return legacyErr
		}
		if legacy.Threshold.Valid && legacy.Threshold.Float64 > 0 {
			seedThreshold = legacy.Threshold.Float64
		}
	}

	_, err = app.DB().NewQuery(`
		UPDATE expenditure_kinds
		SET second_approval_threshold = {:threshold}
		WHERE name = 'standard'
		  AND COALESCE(second_approval_threshold, 0) <= 0
	`).Bind(map[string]any{
		"threshold": seedThreshold,
	}).Execute()
	return err
}

func ensurePurchaseOrdersConfig(app core.App) error {
	record, err := app.FindFirstRecordByData("app_config", "key", "purchase_orders")
	if err != nil {
		collection, collectionErr := app.FindCollectionByNameOrId("app_config")
		if collectionErr != nil {
			return collectionErr
		}
		record = core.NewRecord(collection)
		record.Set("key", "purchase_orders")
		record.Set("value", `{"second_stage_timeout_hours":24}`)
		return app.Save(record)
	}

	config := map[string]any{}
	if raw := record.GetString("value"); raw != "" {
		_ = json.Unmarshal([]byte(raw), &config)
	}
	timeoutRaw, ok := config["second_stage_timeout_hours"]
	valid := false
	if ok {
		switch v := timeoutRaw.(type) {
		case float64:
			valid = v > 0
		case int:
			valid = v > 0
		}
	}
	if !valid {
		config["second_stage_timeout_hours"] = 24
	}

	encoded, marshalErr := json.Marshal(config)
	if marshalErr != nil {
		return marshalErr
	}
	record.Set("value", string(encoded))
	return app.Save(record)
}

func updatePurchaseOrderRules(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("purchase_orders")
	if err != nil {
		return err
	}
	collection.ListRule = stringPtr(purchaseOrderVisibilityRuleV2)
	collection.ViewRule = stringPtr(purchaseOrderVisibilityRuleV2)
	return app.Save(collection)
}

func updatePurchaseOrdersAugmented(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("purchase_orders_augmented")
	if err != nil {
		return err
	}
	collection.ViewQuery = purchaseOrdersAugmentedQueryV2
	collection.ListRule = stringPtr(purchaseOrderVisibilityRuleV2)
	collection.ViewRule = stringPtr(purchaseOrderVisibilityRuleV2)
	return app.Save(collection)
}

func updatePendingSecondApproverView(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("pending_items_for_qualified_po_second_approvers")
	if err != nil {
		return err
	}
	collection.ViewQuery = pendingItemsForQualifiedPOSecondApproversQueryV2
	return app.Save(collection)
}

func tableExists(app core.App, tableName string) (bool, error) {
	var row struct {
		Exists int `db:"exists_flag"`
	}
	if err := app.DB().NewQuery(`
		SELECT CASE
			WHEN EXISTS (
				SELECT 1
				FROM sqlite_master
				WHERE type = 'table' AND name = {:tableName}
			) THEN 1
			ELSE 0
		END AS exists_flag
	`).Bind(map[string]any{
		"tableName": tableName,
	}).One(&row); err != nil {
		return false, err
	}
	return row.Exists == 1, nil
}

func collectionExistsByName(app core.App, name string) (bool, error) {
	var row struct {
		Exists int `db:"exists_flag"`
	}
	if err := app.DB().NewQuery(`
		SELECT CASE
			WHEN EXISTS (
				SELECT 1
				FROM _collections
				WHERE name = {:name}
			) THEN 1
			ELSE 0
		END AS exists_flag
	`).Bind(map[string]any{
		"name": name,
	}).One(&row); err != nil {
		return false, err
	}
	return row.Exists == 1, nil
}

func deleteCollectionIfExists(app core.App, name string) error {
	exists, existsErr := collectionExistsByName(app, name)
	if existsErr != nil {
		return existsErr
	}
	if !exists {
		return nil
	}

	collection, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		return err
	}
	return app.Delete(collection)
}

func upsertUserClaimsSummary(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("user_claims_summary")
	if err == nil {
		collection.ViewQuery = userClaimsSummaryViewQuery
		collection.ListRule = stringPtr("@request.auth.id = id")
		collection.ViewRule = stringPtr("@request.auth.id = id")
		return app.Save(collection)
	}

	jsonData := `{
		"createRule": null,
		"deleteRule": null,
		"fields": [
			{
				"autogeneratePattern": "",
				"hidden": false,
				"id": "text3208210256",
				"max": 0,
				"min": 0,
				"name": "id",
				"pattern": "^[a-z0-9]+$",
				"presentable": false,
				"primaryKey": true,
				"required": true,
				"system": true,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "json1771200001",
				"maxSize": 1,
				"name": "claims",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "json"
			}
		],
		"id": "` + userClaimsSummaryCollectionID + `",
		"indexes": [],
		"listRule": "@request.auth.id = id",
		"name": "user_claims_summary",
		"system": false,
		"type": "view",
		"updateRule": null,
		"viewQuery": "",
		"viewRule": "@request.auth.id = id"
	}`
	collection = &core.Collection{}
	if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
		return err
	}
	collection.ViewQuery = userClaimsSummaryViewQuery
	return app.Save(collection)
}

func upsertUserPOApproverProfile(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("user_po_approver_profile")
	if err == nil {
		collection.ViewQuery = userPOApproverProfileViewQuery
		collection.ListRule = stringPtr("@request.auth.id = id")
		collection.ViewRule = stringPtr("@request.auth.id = id")
		return app.Save(collection)
	}

	jsonData := `{
		"createRule": null,
		"deleteRule": null,
		"fields": [
			{
				"autogeneratePattern": "",
				"hidden": false,
				"id": "text3208210256",
				"max": 0,
				"min": 0,
				"name": "id",
				"pattern": "^[a-z0-9]+$",
				"presentable": false,
				"primaryKey": true,
				"required": true,
				"system": true,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "number1771200002",
				"max": null,
				"min": null,
				"name": "max_amount",
				"onlyInt": false,
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "number1771200003",
				"max": null,
				"min": null,
				"name": "project_max",
				"onlyInt": false,
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "number1771200004",
				"max": null,
				"min": null,
				"name": "sponsorship_max",
				"onlyInt": false,
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "number1771200005",
				"max": null,
				"min": null,
				"name": "staff_and_social_max",
				"onlyInt": false,
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "number1771200006",
				"max": null,
				"min": null,
				"name": "media_and_event_max",
				"onlyInt": false,
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "number1771200007",
				"max": null,
				"min": null,
				"name": "computer_max",
				"onlyInt": false,
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "json1771200002",
				"maxSize": 1,
				"name": "divisions",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "json"
			},
			{
				"hidden": false,
				"id": "json1771200003",
				"maxSize": 1,
				"name": "claims",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "json"
			}
		],
		"id": "` + userPOApproverProfileCollection + `",
		"indexes": [],
		"listRule": "@request.auth.id = id",
		"name": "user_po_approver_profile",
		"system": false,
		"type": "view",
		"updateRule": null,
		"viewQuery": "",
		"viewRule": "@request.auth.id = id"
	}`
	collection = &core.Collection{}
	if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
		return err
	}
	collection.ViewQuery = userPOApproverProfileViewQuery
	return app.Save(collection)
}

func hasMigrationFieldByName(collection *core.Collection, fieldName string) bool {
	for _, field := range collection.Fields {
		if field.GetName() == fieldName {
			return true
		}
	}
	return false
}

func stringPtr(v string) *string {
	return &v
}
