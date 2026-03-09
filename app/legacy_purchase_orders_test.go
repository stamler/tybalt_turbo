package main

import (
	"database/sql"
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func setLegacyPOCreateUpdate(tb testing.TB, app *tests.TestApp, enabled bool) {
	tb.Helper()

	collection, err := app.FindCollectionByNameOrId("app_config")
	if err != nil {
		tb.Fatalf("failed to find app_config collection: %v", err)
	}

	record, err := app.FindFirstRecordByData("app_config", "key", "purchase_orders")
	if err != nil || record == nil {
		record = core.NewRecord(collection)
		record.Set("key", "purchase_orders")
	}
	value := `{"second_stage_timeout_hours":24,"enable_legacy_po_create_update":false}`
	if enabled {
		value = `{"second_stage_timeout_hours":24,"enable_legacy_po_create_update":true}`
	}
	record.Set("value", value)
	if err := app.Save(record); err != nil {
		tb.Fatalf("failed to save purchase_orders config: %v", err)
	}
}

func assertLegacyPOForcedState(tb testing.TB, app *tests.TestApp, id string, requireApproved bool) {
	tb.Helper()

	var status string
	var approved sql.NullString
	var imported int
	var prioritySecondApprover sql.NullString
	var secondApprover sql.NullString
	var secondApproval sql.NullString
	var endDate sql.NullString
	var frequency sql.NullString
	var parentPO sql.NullString
	var category sql.NullString
	var attachment sql.NullString
	var rejector sql.NullString
	var rejected sql.NullString
	var rejectionReason sql.NullString
	var canceller sql.NullString
	var cancelled sql.NullString

	if err := app.DB().NewQuery(`
		SELECT
			status,
			approved,
			_imported,
			priority_second_approver,
			second_approver,
			second_approval,
			end_date,
			frequency,
			parent_po,
			category,
			attachment,
			rejector,
			rejected,
			rejection_reason,
			canceller,
			cancelled
		FROM purchase_orders
		WHERE id = {:id}
	`).Bind(dbx.Params{"id": id}).Row(
		&status,
		&approved,
		&imported,
		&prioritySecondApprover,
		&secondApprover,
		&secondApproval,
		&endDate,
		&frequency,
		&parentPO,
		&category,
		&attachment,
		&rejector,
		&rejected,
		&rejectionReason,
		&canceller,
		&cancelled,
	); err != nil {
		tb.Fatalf("failed to verify legacy PO forced state: %v", err)
	}

	if status != "Active" {
		tb.Fatalf("expected legacy PO status to be Active, got %q", status)
	}
	if imported != 0 {
		tb.Fatalf("expected legacy PO _imported to be forced false, got %d", imported)
	}
	if requireApproved && strings.TrimSpace(approved.String) == "" {
		tb.Fatalf("expected legacy PO approved timestamp to be set on create")
	}

	for field, value := range map[string]sql.NullString{
		"priority_second_approver": prioritySecondApprover,
		"second_approver":          secondApprover,
		"second_approval":          secondApproval,
		"end_date":                 endDate,
		"frequency":                frequency,
		"parent_po":                parentPO,
		"category":                 category,
		"attachment":               attachment,
		"rejector":                 rejector,
		"rejected":                 rejected,
		"rejection_reason":         rejectionReason,
		"canceller":                canceller,
		"cancelled":                cancelled,
	} {
		if strings.TrimSpace(value.String) != "" {
			tb.Fatalf("expected legacy PO %s to be blank, got %q", field, value.String)
		}
	}
}

func setupLegacyPOFeatureDisabledApp(tb testing.TB) *tests.TestApp {
	tb.Helper()

	app := testutils.SetupTestApp(tb)
	setLegacyPOCreateUpdate(tb, app, false)
	return app
}

func TestLegacyPurchaseOrdersCustomAPI(t *testing.T) {
	claimHolderToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}
	noClaimsToken, err := testutils.GenerateRecordToken("users", "u_no_claims@example.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "legacy edit load is denied when feature flag is disabled",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/legacy/legacyeditoff01/edit",
			Headers: map[string]string{
				"Authorization": claimHolderToken,
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"legacy_purchase_order_create_update_disabled"`,
			},
			TestAppFactory: setupLegacyPOFeatureDisabledApp,
		},
		{
			Name:   "legacy create is denied when feature flag is disabled",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/legacy",
			Body: strings.NewReader(`{
				"po_number": "2501-5000",
				"uid": "f2j5a8vk006baub",
				"approver": "wegviunlyr2jjjv",
				"date": "2025-01-15",
				"division": "vccd5fo56ctbigh",
				"branch": "80875lm27v8wgi4",
				"description": "Legacy imported purchase order",
				"payment_type": "OnAccount",
				"total": 321.45,
				"vendor": "yxhycv2ycpvsbt4",
				"type": "One-Time",
				"kind": "l3vtlbqg529m52j"
			}`),
			Headers: map[string]string{
				"Authorization": claimHolderToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"legacy_purchase_order_create_update_disabled"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 0,
			},
			TestAppFactory: setupLegacyPOFeatureDisabledApp,
		},
		{
			Name:   "legacy create is denied without the dedicated claim",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/legacy",
			Body: strings.NewReader(`{
				"po_number": "2501-5000",
				"uid": "f2j5a8vk006baub",
				"approver": "wegviunlyr2jjjv",
				"date": "2025-01-15",
				"division": "vccd5fo56ctbigh",
				"branch": "80875lm27v8wgi4",
				"description": "Legacy imported purchase order",
				"payment_type": "OnAccount",
				"total": 321.45,
				"vendor": "yxhycv2ycpvsbt4",
				"type": "One-Time",
				"kind": "l3vtlbqg529m52j"
			}`),
			Headers: map[string]string{
				"Authorization": noClaimsToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"legacy_purchase_order_claim_required"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "legacy update is denied without the dedicated claim",
			Method: http.MethodPatch,
			URL:    "/api/purchase_orders/legacy/legacynoclaim1",
			Body: strings.NewReader(`{
				"description": "Attempt to update without claim"
			}`),
			Headers: map[string]string{
				"Authorization": noClaimsToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"legacy_purchase_order_claim_required"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdateRequest": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "legacy claim holder can create active legacy PO with manual number",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/legacy",
			Body: strings.NewReader(`{
				"po_number": "2501-5000",
				"uid": "f2j5a8vk006baub",
				"approver": "wegviunlyr2jjjv",
				"date": "2025-01-15",
				"division": "vccd5fo56ctbigh",
				"branch": "80875lm27v8wgi4",
				"description": "Legacy imported purchase order",
				"payment_type": "OnAccount",
				"total": 321.45,
				"vendor": "yxhycv2ycpvsbt4",
				"type": "One-Time",
				"kind": "l3vtlbqg529m52j"
			}`),
			Headers: map[string]string{
				"Authorization": claimHolderToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"legacy_manual_entry":true`,
				`"po_number":"2501-5000"`,
				`"status":"Active"`,
				`"uid":"f2j5a8vk006baub"`,
				`"approver":"wegviunlyr2jjjv"`,
				`"_imported":false`,
			},
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				tb.Helper()
				var id string
				if err := app.DB().NewQuery(`SELECT id FROM purchase_orders WHERE po_number = '2501-5000' LIMIT 1`).Row(&id); err != nil {
					tb.Fatalf("failed to find created legacy PO: %v", err)
				}
				assertLegacyPOForcedState(tb, app, id, true)
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "legacy create rejects invalid manual PO number",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/legacy",
			Body: strings.NewReader(`{
				"po_number": "2401-4000",
				"uid": "f2j5a8vk006baub",
				"approver": "wegviunlyr2jjjv",
				"date": "2025-01-15",
				"division": "vccd5fo56ctbigh",
				"branch": "80875lm27v8wgi4",
				"description": "Legacy invalid number",
				"payment_type": "OnAccount",
				"total": 321.45,
				"vendor": "yxhycv2ycpvsbt4",
				"type": "One-Time",
				"kind": "l3vtlbqg529m52j"
			}`),
			Headers: map[string]string{
				"Authorization": claimHolderToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"po_number":{"code":"invalid_legacy_po_number"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "legacy create requires branch",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/legacy",
			Body: strings.NewReader(`{
				"po_number": "2501-5009",
				"uid": "f2j5a8vk006baub",
				"approver": "wegviunlyr2jjjv",
				"date": "2025-01-15",
				"division": "vccd5fo56ctbigh",
				"description": "Legacy missing branch",
				"payment_type": "OnAccount",
				"total": 321.45,
				"vendor": "yxhycv2ycpvsbt4",
				"type": "One-Time",
				"kind": "l3vtlbqg529m52j"
			}`),
			Headers: map[string]string{
				"Authorization": claimHolderToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"branch":{"code":"required"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "legacy create rejects category field",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/legacy",
			Body: strings.NewReader(`{
				"po_number": "2501-5010",
				"uid": "f2j5a8vk006baub",
				"approver": "wegviunlyr2jjjv",
				"date": "2025-01-15",
				"division": "vccd5fo56ctbigh",
				"branch": "80875lm27v8wgi4",
				"description": "Legacy category should be blank",
				"payment_type": "OnAccount",
				"total": 321.45,
				"vendor": "yxhycv2ycpvsbt4",
				"type": "One-Time",
				"kind": "l3vtlbqg529m52j",
				"category": "somedisallowedcat"
			}`),
			Headers: map[string]string{
				"Authorization": claimHolderToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"category":{"code":"field_not_allowed"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "legacy create rejects recurring type",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/legacy",
			Body: strings.NewReader(`{
				"po_number": "2501-5008",
				"uid": "f2j5a8vk006baub",
				"approver": "wegviunlyr2jjjv",
				"date": "2025-01-15",
				"division": "vccd5fo56ctbigh",
				"branch": "80875lm27v8wgi4",
				"description": "Legacy recurring type",
				"payment_type": "OnAccount",
				"total": 321.45,
				"vendor": "yxhycv2ycpvsbt4",
				"type": "Recurring",
				"kind": "l3vtlbqg529m52j"
			}`),
			Headers: map[string]string{
				"Authorization": claimHolderToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"type":{"code":"invalid_type"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "legacy create rejects disallowed workflow fields",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/legacy",
			Body: strings.NewReader(`{
				"po_number": "2501-5000",
				"uid": "f2j5a8vk006baub",
				"approver": "wegviunlyr2jjjv",
				"date": "2025-01-15",
				"division": "vccd5fo56ctbigh",
				"branch": "80875lm27v8wgi4",
				"description": "Legacy imported purchase order",
				"payment_type": "OnAccount",
				"total": 321.45,
				"vendor": "yxhycv2ycpvsbt4",
				"type": "One-Time",
				"kind": "l3vtlbqg529m52j",
				"_imported": true
			}`),
			Headers: map[string]string{
				"Authorization": claimHolderToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"_imported":{"code":"field_not_allowed"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "legacy edit load succeeds for legacy purchase order",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/legacy/legacyedit0001/edit",
			Headers: map[string]string{
				"Authorization": claimHolderToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"legacyedit0001"`,
				`"legacy_manual_entry":true`,
				`"po_number":"2502-5001"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "legacy edit load succeeds for closed legacy purchase order",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/legacy/legacyclosedview1/edit",
			Headers: map[string]string{
				"Authorization": claimHolderToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"legacyclosedview1"`,
				`"legacy_manual_entry":true`,
				`"status":"Closed"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "legacy update is denied for a non-legacy purchase order",
			Method: http.MethodPatch,
			URL:    "/api/purchase_orders/legacy/standardupd001",
			Body: strings.NewReader(`{
				"description": "Attempt to convert standard PO to legacy"
			}`),
			Headers: map[string]string{
				"Authorization": claimHolderToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"legacy_purchase_order_only"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdateRequest": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "legacy update forces active manual state",
			Method: http.MethodPatch,
			URL:    "/api/purchase_orders/legacy/legacyupd00001",
			Body: strings.NewReader(`{
				"description": "Legacy PO updated in Turbo",
				"po_number": "2502-5011"
			}`),
			Headers: map[string]string{
				"Authorization": claimHolderToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"description":"Legacy PO updated in Turbo"`,
				`"_imported":false`,
			},
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				tb.Helper()
				assertLegacyPOForcedState(tb, app, "legacyupd00001", false)
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdateRequest": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "cancelled legacy PO cannot be edited",
			Method: http.MethodPatch,
			URL:    "/api/purchase_orders/legacy/legacycancelled01",
			Body: strings.NewReader(`{
				"description": "Attempt to edit cancelled legacy PO",
				"po_number": "2503-5006"
			}`),
			Headers: map[string]string{
				"Authorization": claimHolderToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"status":{"code":"cancelled_legacy_purchase_order"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdateRequest": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "closed legacy PO cannot be edited",
			Method: http.MethodPatch,
			URL:    "/api/purchase_orders/legacy/legacyclosed01",
			Body: strings.NewReader(`{
				"description": "Attempt to edit closed legacy PO",
				"po_number": "2503-5002"
			}`),
			Headers: map[string]string{
				"Authorization": claimHolderToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"status":{"code":"closed_legacy_purchase_order"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdateRequest": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "legacy create fails when uid is inactive",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/legacy",
			Body: strings.NewReader(`{
				"po_number": "2503-5007",
				"uid": "u_inactive",
				"approver": "wegviunlyr2jjjv",
				"date": "2025-01-15",
				"division": "vccd5fo56ctbigh",
				"branch": "80875lm27v8wgi4",
				"description": "Legacy imported purchase order",
				"payment_type": "OnAccount",
				"total": 321.45,
				"vendor": "yxhycv2ycpvsbt4",
				"type": "One-Time",
				"kind": "l3vtlbqg529m52j"
			}`),
			Headers: map[string]string{
				"Authorization": claimHolderToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"uid":{"code":"validation_error","message":"the selected staff member is not an active user","data":null}`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestLegacyPurchaseOrdersCollectionRegression(t *testing.T) {
	claimHolderToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "collection create cannot create legacy purchase orders",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"legacy_manual_entry": true,
				"po_number": "2501-5000",
				"uid": "f2j5a8vk006baub",
				"date": "2025-01-15",
				"division": "vccd5fo56ctbigh",
				"description": "Legacy imported purchase order",
				"payment_type": "OnAccount",
				"total": 321.45,
				"vendor": "yxhycv2ycpvsbt4",
				"status": "Unapproved",
				"type": "One-Time",
				"kind": "l3vtlbqg529m52j"
			}`),
			Headers: map[string]string{
				"Authorization": claimHolderToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"message":"Failed to create record."`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "collection update cannot modify legacy purchase orders",
			Method: http.MethodPatch,
			URL:    "/api/collections/purchase_orders/records/legacyviactl01",
			Body: strings.NewReader(`{
				"description": "Should be rejected by collection update rule"
			}`),
			Headers: map[string]string{
				"Authorization": claimHolderToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdateRequest": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "collection delete cannot delete active legacy purchase orders",
			Method: http.MethodDelete,
			URL:    "/api/collections/purchase_orders/records/legacyviactl02",
			Headers: map[string]string{
				"Authorization": claimHolderToken,
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestLegacyPurchaseOrdersVisibleAPI(t *testing.T) {
	claimHolderToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	regularUserToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "legacy claim holder can view legacy purchase order through normal visible API",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible/legacyvis00001",
			Headers: map[string]string{
				"Authorization": claimHolderToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"legacyvis00001"`,
				`"legacy_manual_entry":true`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "legacy claim holder sees legacy purchase order in normal visible list payload",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible?scope=all",
			Headers: map[string]string{
				"Authorization": claimHolderToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"legacyvislist1"`,
				`"legacy_manual_entry":true`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user without legacy claim does not gain visibility to unapproved legacy purchase order",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible/legacyvis00002",
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"po_not_found_or_not_visible"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
