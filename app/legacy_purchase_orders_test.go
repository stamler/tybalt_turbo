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

const legacyPOClaimHolderUID = "wegviunlyr2jjjv"

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

func enableLegacyPOCreateUpdate(tb testing.TB, app *tests.TestApp) {
	tb.Helper()
	setLegacyPOCreateUpdate(tb, app, true)
}

func assignLegacyPOClaim(tb testing.TB, app *tests.TestApp, userClaimID string) {
	tb.Helper()

	var claimID string
	if err := app.DB().NewQuery(`SELECT id FROM claims WHERE name = 'legacy_po_create_update' LIMIT 1`).Row(&claimID); err != nil {
		tb.Fatalf("failed to find legacy_po_create_update claim id: %v", err)
	}

	if _, err := app.DB().NewQuery(`
		INSERT OR IGNORE INTO user_claims (id, uid, cid, created, updated)
		VALUES ({:id}, {:uid}, {:cid}, datetime('now'), datetime('now'))
	`).Bind(dbx.Params{
		"id":  userClaimID,
		"uid": legacyPOClaimHolderUID,
		"cid": claimID,
	}).Execute(); err != nil {
		tb.Fatalf("failed to insert legacy PO claim: %v", err)
	}
}

func insertLegacyPO(tb testing.TB, app *tests.TestApp, id string, poNumber string, status string, imported bool) {
	tb.Helper()

	importedInt := 0
	if imported {
		importedInt = 1
	}

	if _, err := app.DB().NewQuery(`
		INSERT INTO purchase_orders (
			id, uid, approver, date, division, description, payment_type, total, approval_total,
			vendor, status, type, po_number, _imported, branch, kind, legacy_manual_entry,
			created, updated
		) VALUES (
			{:id}, 'f2j5a8vk006baub', 'wegviunlyr2jjjv', '2025-01-15', 'vccd5fo56ctbigh', 'Legacy fixture purchase order',
			'OnAccount', 100.00, 100.00, 'yxhycv2ycpvsbt4', {:status}, 'One-Time', {:poNumber},
			{:imported}, '80875lm27v8wgi4', 'l3vtlbqg529m52j', 1, datetime('now'), datetime('now')
		)
	`).Bind(dbx.Params{
		"id":       id,
		"status":   status,
		"poNumber": poNumber,
		"imported": importedInt,
	}).Execute(); err != nil {
		tb.Fatalf("failed to insert legacy PO fixture: %v", err)
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

func insertStandardPO(tb testing.TB, app *tests.TestApp, id string, uid string, status string) {
	tb.Helper()

	if _, err := app.DB().NewQuery(`
		INSERT INTO purchase_orders (
			id, uid, approver, date, division, description, payment_type, total, approval_total,
			vendor, status, type, po_number, _imported, branch, kind, legacy_manual_entry,
			created, updated
		) VALUES (
			{:id}, {:uid}, {:uid}, '2025-01-15', 'vccd5fo56ctbigh', 'Standard fixture purchase order',
			'OnAccount', 100.00, 100.00, 'yxhycv2ycpvsbt4', {:status}, 'One-Time', '',
			0, '80875lm27v8wgi4', 'l3vtlbqg529m52j', 0, datetime('now'), datetime('now')
		)
	`).Bind(dbx.Params{
		"id":     id,
		"uid":    uid,
		"status": status,
	}).Execute(); err != nil {
		tb.Fatalf("failed to insert standard PO fixture: %v", err)
	}
}

func TestLegacyPurchaseOrdersCustomAPI(t *testing.T) {
	claimHolderToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				setLegacyPOCreateUpdate(tb, app, false)
				assignLegacyPOClaim(tb, app, "uclgeditoff001")
				insertLegacyPO(tb, app, "legacyeditoff01", "2501-5000", "Active", false)
			},
			TestAppFactory: testutils.SetupTestApp,
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				setLegacyPOCreateUpdate(tb, app, false)
				assignLegacyPOClaim(tb, app, "uclgdisabled001")
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
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
				"Authorization": claimHolderToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"legacy_purchase_order_claim_required"`,
			},
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
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
				"Authorization": claimHolderToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"legacy_purchase_order_claim_required"`,
			},
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				insertLegacyPO(tb, app, "legacynoclaim1", "2501-5000", "Active", false)
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclgcreate00001")
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclginvalid0001")
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclgmissingbranch")
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclgcategorydeny1")
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclginvalidtype1")
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclgdisallow001")
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclgeditload01")
				insertLegacyPO(tb, app, "legacyedit0001", "2502-5001", "Active", false)
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclgclosedview01")
				insertLegacyPO(tb, app, "legacyclosedview1", "2502-5009", "Closed", false)
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclgnonlegacy01")
				insertStandardPO(tb, app, "standardupd001", legacyPOClaimHolderUID, "Unapproved")
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
				"po_number": "2502-5001"
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclgupdate00001")
				insertLegacyPO(tb, app, "legacyupd00001", "2502-5001", "Active", true)
				if _, err := app.DB().NewQuery(`
					UPDATE purchase_orders
					SET
						priority_second_approver = 'wegviunlyr2jjjv',
						second_approver = 'wegviunlyr2jjjv',
						second_approval = datetime('now'),
						end_date = '2025-02-15',
						frequency = 'Monthly',
						parent_po = 'someparentpoid',
						attachment = 'legacy.pdf',
						rejector = 'wegviunlyr2jjjv',
						rejected = datetime('now'),
						rejection_reason = 'old rejection'
					WHERE id = 'legacyupd00001'
				`).Execute(); err != nil {
					tb.Fatalf("failed to seed legacy PO stale workflow state: %v", err)
				}
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclgcancelled0001")
				insertLegacyPO(tb, app, "legacycancelled01", "2503-5006", "Cancelled", false)
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclgclosed00001")
				insertLegacyPO(tb, app, "legacyclosed01", "2503-5002", "Closed", false)
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclginactiveuid01")
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclgcollection01")
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclgcollection02")
				insertLegacyPO(tb, app, "legacyviactl01", "2504-5003", "Unapproved", false)
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclgcollection03")
				insertLegacyPO(tb, app, "legacyviactl02", "2508-5007", "Active", false)
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclgvisible001")
				insertLegacyPO(tb, app, "legacyvis00001", "2505-5004", "Unapproved", false)
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclgvisible002")
				insertLegacyPO(tb, app, "legacyvislist1", "2506-5005", "Unapproved", false)
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				enableLegacyPOCreateUpdate(tb, app)
				assignLegacyPOClaim(tb, app, "uclgvisible003")
				insertLegacyPO(tb, app, "legacyvis00002", "2507-5006", "Unapproved", false)
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
