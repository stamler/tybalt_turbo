package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"testing"
	"time"
	"tybalt/internal/testutils"
	"tybalt/routes"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestPurchaseOrdersCreate(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	// Generate token for user with po_approver claim
	poApproverToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	// Generate token for user with division-specific po_approver claim
	divisionApproverToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	// Generate token for user with po_approver_tier3 claim
	po_approver_tier3Token, err := testutils.GenerateRecordToken("users", "hal@2005.com")
	if err != nil {
		t.Fatal(err)
	}

	// Generate token for user with po_approver_tier2 claim
	po_approver_tier2Token, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	// Get approval tier values once and reuse them throughout the tests
	app := testutils.SetupTestApp(t)
	tier1, tier2 := testutils.GetApprovalTiers(app)

	scenarios := []tests.ApiScenario{
		{
			Name:   "valid purchase order is created",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Normal"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"approver":"etysnrlup2f6bak"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "recurring purchase order requires end_date and frequency",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Recurring",
				"end_date": "2024-11-01",
				"frequency": "Monthly"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"approver":"etysnrlup2f6bak"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "recurring purchase order fails without end_date",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Recurring",
				"frequency": "Monthly"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"end_date":{"code":"value_required"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "recurring purchase fails without frequency",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Recurring",
				"end_date": "2024-11-01"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"frequency":{"code":"value_required"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "recurring purchase order fails with less than 2 occurrences",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Recurring",
				"end_date": "2024-10-01",
				"frequency": "Monthly"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"global":{"code":"fewer_than_two_occurrences"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "recurring purchase order fails if end_date is not after start_date",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Recurring",
				"end_date": "2024-09-01",
				"frequency": "Monthly"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"end_date":{"code":"end_date_not_after_start_date"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "recurring purchase order allows other frequencies",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Recurring",
				"end_date": "2024-11-01",
				"frequency": "Weekly"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"approver":"etysnrlup2f6bak"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "recurring purchase order fails when frequency is not valid",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Recurring",
				"end_date": "2024-11-01",
				"frequency": "Invalid"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"frequency":{"code":"invalid_frequency"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "otherwise valid purchase order fails when approver is set non-qualified user",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "tqqf7q0f3378rvp",
				"status": "Unapproved",
				"type": "Normal"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"approver":{"code":"validation_no_claim"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "otherwise valid purchase order fails when approver is set to blank string or missing",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "",
				"status": "Unapproved",
				"type": "Normal"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"approver":{"code":"value_required"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "valid child purchase order is created",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"parent_po": "ly8xyzpuj79upq1",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "this one is cumulative",
				"payment_type": "OnAccount",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Normal",
				"job": "cjf0kt0defhq480",
				"category": "t5nmdl188gtlhz0"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"approver":"etysnrlup2f6bak"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		// We need a test child PO that is status Active
		{
			Name:   "a child purchase order cannot itself be a parent",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"parent_po": "25046ft47x49cc2",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "this one is cumulative",
				"payment_type": "OnAccount",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Normal",
				"job": "cjf0kt0defhq480",
				"category": "t5nmdl188gtlhz0"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"parent_po":{"code":"child_po_cannot_be_parent"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "child purchase order may not be of type Cumulative",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"parent_po": "ly8xyzpuj79upq1",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "this one is cumulative",
				"payment_type": "OnAccount",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Cumulative",
				"job": "cjf0kt0defhq480",
				"category": "t5nmdl188gtlhz0"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"type":{"code":"validation_in_invalid"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "child purchase order may not be of type Recurring",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"parent_po": "ly8xyzpuj79upq1",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "this one is cumulative",
				"payment_type": "OnAccount",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Recurring",
				"end_date": "2024-11-01",
				"frequency": "Monthly",
				"job": "cjf0kt0defhq480",
				"category": "t5nmdl188gtlhz0"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"type":{"code":"validation_in_invalid"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "fails when other child POs with status 'Unapproved' exist",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"parent_po": "y660i6a14ql2355",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "this one is cumulative",
				"payment_type": "OnAccount",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Normal",
				"job": "cjf0kt0defhq480",
				"category": "t5nmdl188gtlhz0"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"parent_po":{"code":"existing_children_with_blocking_status"`,
			},
			ExpectedEvents: map[string]int{
				"*":                     0,
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "fails when parent_po is not cumulative",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"parent_po": "2plsetqdxht7esg",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "this one is cumulative",
				"payment_type": "OnAccount",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Normal",
				"job": "cjf0kt0defhq480",
				"category": "t5nmdl188gtlhz0"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"parent_po":{"code":"invalid_type"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "fails when job of child purchase order does not match job of parent purchase order",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"parent_po": "ly8xyzpuj79upq1",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "this one is cumulative",
				"payment_type": "OnAccount",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Normal",
				"job": "non-matching-job",
				"category": "t5nmdl188gtlhz0"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"job":{"code":"value_mismatch"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "otherwise valid purchase order with Inactive vendor fails",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": 1234.56,
				"vendor": "ctswqva5onxj75q",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Normal"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"message":"Failed to create record.","status":400`,
			},
			ExpectedEvents: map[string]int{},
			TestAppFactory: testutils.SetupTestApp,
		},
		/*
		   This test verifies the basic auto-approval flow for purchase orders.
		   When a user with the po_approver claim (empty divsions property of po_approver_props = all divisions) creates a PO:
		   1. The PO should be auto-approved immediately:
		      - approved timestamp should be set to current date/time
		      - approver should be set to the creator's ID
		   2. Status should become "Active" (since no second approval needed for low value)
		   3. PO number should be generated (format: YYYY-NNNN)

		   Test setup:
		   - Uses user wegviunlyr2jjjv (fakemanager@fakesite.xyz) who has po_approver claim
		   - Sets PO total to random value below tier1 to avoid triggering second approval
		   - Uses correct auth token matching the creator's ID

		   Verification points:
		   - approved: Checks timestamp starts with current date
		   - status: Must be "Active"
		   - po_number: Must start with current year
		   - approver: Must be creator's ID (wegviunlyr2jjjv)
		*/
		{
			Name:   "purchase order is not automatically approved when creator has po_approver claim",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(fmt.Sprintf(`{
				"uid": "wegviunlyr2jjjv",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": %.2f,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Normal"
			}`, rand.Float64()*(tier1-1.0)+1.0)), // Random value between 1 and tier1
			Headers:        map[string]string{"Authorization": poApproverToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"status":"Unapproved"`,
				`"po_number":""`,
				`"approver":"etysnrlup2f6bak"`, // Original approver remains
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		/*
		   These tests verify division-specific auto-approval for purchase orders.
		   User fatt@mac.com (id: etysnrlup2f6bak) has po_approver claim with divisions property:
		   ["hcd86z57zjty6jo", "fy4i9poneukvq9u", "vccd5fo56ctbigh"] on the po_approver_props record

		   Test 1 (Success case):
		   - Creates PO with division "vccd5fo56ctbigh" (in user's po_approver_props divisions property)
		   - Should auto-approve since user has permission for this division
		   - Verifies: approval timestamp, Active status, PO number generation
		   - Creator becomes approver

		   Test 2 (Failure case):
		   - Creates PO with division "ngpjzurmkrfl8fo" (not in user's po_approver_props divisions property)
		   - Uses wegviunlyr2jjjv as approver (has empty po_approver_props divisions property = all divisions)
		   - Should succeed (200) but not auto-approve
		   - Verifies: no approval, Unapproved status, original approver remains

		   Both tests:
		   - Use random total below tier1 to avoid second approval
		   - Use correct auth token for fatt@mac.com
		   - Match uid to authenticated user's ID
		*/
		{
			Name:   "purchase order is not automatically approved when creator has po_approver claim for non-matching division",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(fmt.Sprintf(`{
				"uid": "etysnrlup2f6bak",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": %.2f,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Normal"
			}`, rand.Float64()*(tier1-1.0)+1.0)), // Random value between 1 and tier1
			Headers:        map[string]string{"Authorization": divisionApproverToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"status":"Unapproved"`,
				`"po_number":""`,
				`"approver":"etysnrlup2f6bak"`, // Original approver remains
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "purchase order is not auto-approved when creator has po_approver claim but non-matching division",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(fmt.Sprintf(`{
				"uid": "etysnrlup2f6bak",
				"date": "2024-09-01",
				"division": "ngpjzurmkrfl8fo",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": %.2f,
				"vendor": "2zqxtsmymf670ha",
				"approver": "wegviunlyr2jjjv",
				"status": "Unapproved",
				"type": "Normal"
			}`, rand.Float64()*(tier1-1.0)+1.0)), // Random value between 1 and tier1
			Headers:        map[string]string{"Authorization": divisionApproverToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,                // Should not be approved
				`"status":"Unapproved"`,        // Status should remain Unapproved
				`"approver":"wegviunlyr2jjjv"`, // Original approver should remain
				`"po_number":""`,               // No PO number should be generated
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		/*
		   This test verifies auto-approval of high-value purchase orders by users with elevated claims.
		   User hal@2005.com (id: 66ct66w380ob6w8) has:
		   - po_approver claim with empty divisions property on the po_approver_props record (can approve for any division)
		   - po_approver_tier3 claim (can provide second approval for high-value POs)

		   Test verifies that when this user creates a high-value PO:
		   1. First approval is automatic (due to po_approver claim)
		   2. Second approval is also automatic (due to po_approver_tier3 claim)
		   3. Status becomes Active and PO number is generated
		   4. Creator is set as both approver and second_approver

		   The test:
		   - Uses total above tier2 to trigger second approval requirement
		   - Uses random division (since user has empty po_approver_props divisions property)
		   - Verifies all approval fields and timestamps
		*/
		{
			Name:   "purchase order is not automatically approved when creator has po_approver and po_approver_tier3 claims",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(fmt.Sprintf(`{
				"uid": "66ct66w380ob6w8",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": %.2f,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Normal"
			}`, rand.Float64()*(1000.0)+tier2)), // Random value > tier2
			Headers:        map[string]string{"Authorization": po_approver_tier3Token},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"second_approval":""`,
				`"status":"Unapproved"`,
				`"po_number":""`,
				`"approver":"etysnrlup2f6bak"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		/*
		   This test verifies auto-approval of mid-range value purchase orders by users with po_approver_tier2 claim.
		   User author@soup.com (id: f2j5a8vk006baub) has:
		   - po_approver claim with empty divisions property on the po_approver_props record (can approve for any division)
		   - po_approver_tier2 claim (can provide second approval for POs between tier1 and tier2)

		   Test verifies that when this user creates a mid-range value PO:
		   1. First approval is automatic (due to po_approver claim)
		   2. Second approval is also automatic (due to po_approver_tier2 claim)
		   3. Status becomes Active and PO number is generated
		   4. Creator is set as both approver and second_approver

		   The test:
		   - Uses total between tier1 and tier2
		   - Uses random division (since user has empty po_approver_props divisions property)
		   - Verifies all approval fields and timestamps
		*/
		{
			Name:   "purchase order is not automatically approved when creator has po_approver and po_approver_tier2 claims",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(fmt.Sprintf(`{
				"uid": "f2j5a8vk006baub",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": %.2f,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Normal"
			}`, rand.Float64()*(tier2-tier1)+tier1)), // Random value between tier1 and tier2
			Headers:        map[string]string{"Authorization": po_approver_tier2Token},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"second_approval":""`,
				`"status":"Unapproved"`,
				`"po_number":""`,
				`"approver":"etysnrlup2f6bak"`, // Original approver remains
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 2, // 1 for the PO, 1 for the notification
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		// Add a new test case for priority_second_approver validation
		{
			Name:   "fails when priority_second_approver is not authorized for the PO amount",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Normal",
				"priority_second_approver": "regularUser"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"priority_second_approver":{"code":"invalid_priority_second_approver"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestPurchaseOrdersUpdate(t *testing.T) {

	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "valid purchase order is updated",
			Method: http.MethodPatch,
			URL:    "/api/collections/purchase_orders/records/gal6e5la2fa4rpn",
			Body: strings.NewReader(`{
				"uid": "f2j5a8vk006baub",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": 2234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Cumulative"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"approver":"etysnrlup2f6bak"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "otherwise valid purchase order with Inactive vendor fails",
			Method: http.MethodPatch,
			URL:    "/api/collections/purchase_orders/records/gal6e5la2fa4rpn",
			Body: strings.NewReader(`{
				"uid": "f2j5a8vk006baub",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test purchase order",
				"payment_type": "Expense",
				"total": 2234.56,
				"vendor": "ctswqva5onxj75q",
				"approver": "etysnrlup2f6bak",
				"status": "Unapproved",
				"type": "Cumulative"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found.","status":404`,
			},
			ExpectedEvents: map[string]int{},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestPurchaseOrdersDelete(t *testing.T) {
	/*
		recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
		if err != nil {
			t.Fatal(err)
		}

		nonCreatorToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
		if err != nil {
			t.Fatal(err)
		}
	*/

	scenarios := []tests.ApiScenario{
		// TODO: Add test scenarios for purchase order deletion
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestPurchaseOrdersRead(t *testing.T) {
	/*
		creatorToken, err := testutils.GenerateRecordToken("users", "time@test.com")
		if err != nil {
			t.Fatal(err)
		}

		approverToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
		if err != nil {
			t.Fatal(err)
		}

		reportToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
		if err != nil {
			t.Fatal(err)
		}
	*/

	scenarios := []tests.ApiScenario{
		// TODO: Add test scenarios for purchase order reading
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestGeneratePONumber(t *testing.T) {
	currentYear := time.Now().Year() % 100
	currentMonth := time.Now().Month()
	currentPoPrefix := fmt.Sprintf("%d%02d-", currentYear, currentMonth)
	app := testutils.SetupTestApp(t)
	poCollection, err := app.FindCollectionByNameOrId("purchase_orders")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		year          int // 0 for current year
		month         int // 0 for current month
		record        *core.Record
		setup         func(t *testing.T, app *tests.TestApp)
		cleanup       func(t *testing.T, app *tests.TestApp)
		expected      string
		expectedError string
	}{
		{
			// Test Case 1: First Child PO
			// This test verifies that when creating the first child PO for an existing parent:
			// - The parent PO (2024-0008) is found in the database
			// - No existing child POs are found
			// - The generated number follows the format: parent_number + "-01"
			// - The number is unique in the database
			name: "first child PO for 2024-0008",
			record: func() *core.Record {
				r := core.NewRecord(poCollection)
				r.Set("parent_po", "2plsetqdxht7esg") // ID of PO 2024-0008
				return r
			}(),
			expected: "2024-0008-01",
		},
		{
			// Test Case 2: Second Child PO
			// This test verifies that when creating a second child PO:
			// - The parent PO (2024-0008) is found
			// - An existing child PO (2024-0008-01) is found
			// - The next sequential number (-02) is generated
			// - The number is unique in the database
			//
			// The test:
			// 1. Sets up by creating the first child PO (2024-0008-01)
			// 2. Attempts to create a second child PO
			// 3. Cleans up by removing all child POs created during the test
			name: "second child PO for 2024-0008",
			record: func() *core.Record {
				r := core.NewRecord(poCollection)
				r.Set("parent_po", "2plsetqdxht7esg")
				r.Set("uid", "f2j5a8vk006baub")
				r.Set("type", "Normal")
				r.Set("date", "2024-01-01")
				r.Set("division", "ngpjzurmkrfl8fo")
				r.Set("description", "Test description")
				r.Set("total", 100.0)
				r.Set("payment_type", "OnAccount")
				r.Set("vendor", "2zqxtsmymf670ha")
				r.Set("approver", "wegviunlyr2jjjv")
				r.Set("status", "Unapproved")
				return r
			}(),
			setup: func(t *testing.T, app *tests.TestApp) {
				// Create and save the first child PO
				firstChild := core.NewRecord(poCollection)
				firstChild.Set("parent_po", "2plsetqdxht7esg")
				firstChild.Set("uid", "f2j5a8vk006baub")
				firstChild.Set("type", "Normal")
				firstChild.Set("po_number", "2024-0008-01")
				firstChild.Set("date", "2024-01-01")
				firstChild.Set("division", "ngpjzurmkrfl8fo")
				firstChild.Set("description", "Test description")
				firstChild.Set("total", 100.0)
				firstChild.Set("approval_total", 100.0)
				firstChild.Set("payment_type", "OnAccount")
				firstChild.Set("vendor", "2zqxtsmymf670ha")
				firstChild.Set("approver", "wegviunlyr2jjjv")
				firstChild.Set("status", "Unapproved")
				if err := app.Save(firstChild); err != nil {
					t.Fatalf("failed to save first child PO: %v", err)
				}
			},
			cleanup: func(t *testing.T, app *tests.TestApp) {
				// Delete any child POs we created
				records, err := app.FindRecordsByFilter(
					"purchase_orders",
					"parent_po = {:parentId}",
					"",
					0,
					0,
					dbx.Params{"parentId": "2plsetqdxht7esg"},
				)
				if err != nil {
					t.Fatalf("failed to find child POs to clean up: %v", err)
				}
				for _, record := range records {
					if err := app.Delete(record); err != nil {
						t.Fatalf("failed to delete child PO: %v", err)
					}
				}
			},
			expected: "2024-0008-02",
		},
		{
			// Test Case 3: Parent PO Number Generation (for current YYMM)
			// This test verifies that when creating a new parent PO for the current year/month:
			// - A PO is set up with number YYMM-NNNN.
			// - The next generated sequential number is YYMM-(NNNN+1).
			// - The number is unique in the database.
			name:  "sequential parent PO",
			year:  2024,
			month: 1,
			record: func() *core.Record {
				return core.NewRecord(poCollection)
			}(),
			expected: "2401-0010", // Next after 2024-0009 in test DB
		},
		{
			// Test Case 4: Parent PO Without Number
			// This test verifies that when creating a child PO:
			// - If the parent PO exists but has no PO number
			// - An appropriate error is returned
			name: "parent PO without number",
			record: func() *core.Record {
				r := core.NewRecord(poCollection)
				r.Set("parent_po", "gal6e5la2fa4rpn") // ID of a PO without number
				return r
			}(),
			expectedError: "parent PO does not have a PO number",
		},
		{
			// Test Case 5: Maximum Child POs Reached
			// This test verifies that when creating a child PO:
			// - If 99 child POs already exist for the parent
			// - An appropriate error is returned
			//
			// The test:
			// 1. Sets up by creating 99 child POs
			// 2. Attempts to create the 100th child PO
			// 3. Cleans up all created child POs
			name: "maximum child POs reached",
			record: func() *core.Record {
				r := core.NewRecord(poCollection)
				r.Set("parent_po", "2plsetqdxht7esg")
				r.Set("uid", "f2j5a8vk006baub")
				r.Set("type", "Normal")
				r.Set("date", "2024-01-01")
				r.Set("division", "ngpjzurmkrfl8fo")
				r.Set("description", "Test description")
				r.Set("total", 100.0)
				r.Set("payment_type", "OnAccount")
				r.Set("vendor", "2zqxtsmymf670ha")
				r.Set("approver", "wegviunlyr2jjjv")
				r.Set("status", "Unapproved")
				return r
			}(),
			setup: func(t *testing.T, app *tests.TestApp) {
				// Create 99 child POs
				for i := 1; i <= 99; i++ {
					child := core.NewRecord(poCollection)
					child.Set("parent_po", "2plsetqdxht7esg")
					child.Set("po_number", fmt.Sprintf("2024-0008-%02d", i))
					child.Set("uid", "f2j5a8vk006baub")
					child.Set("type", "Normal")
					child.Set("date", "2024-01-01")
					child.Set("division", "ngpjzurmkrfl8fo")
					child.Set("description", "Test description")
					child.Set("total", 100.0)
					child.Set("approval_total", 100.0)
					child.Set("payment_type", "OnAccount")
					child.Set("vendor", "2zqxtsmymf670ha")
					child.Set("approver", "wegviunlyr2jjjv")
					child.Set("status", "Unapproved")
					if err := app.Save(child); err != nil {
						t.Fatalf("failed to save child PO %d: %v", i, err)
					}
				}
			},
			cleanup: func(t *testing.T, app *tests.TestApp) {
				// Delete all child POs
				records, err := app.FindRecordsByFilter(
					"purchase_orders",
					"parent_po = {:parentId}",
					"",
					0,
					0,
					dbx.Params{"parentId": "2plsetqdxht7esg"},
				)
				if err != nil {
					t.Fatalf("failed to find child POs to clean up: %v", err)
				}
				for _, record := range records {
					if err := app.Delete(record); err != nil {
						t.Fatalf("failed to delete child PO: %v", err)
					}
				}
			},
			expectedError: "maximum number of child POs reached (99) for parent 2024-0008",
		},
		{
			// Test Case 6: First PO of the year
			// This test verifies that when creating the first PO of the year:
			// - No existing POs are found
			// - The generated number follows the format: YYMM + "-0001"
			// - The number is unique in the database
			name: "first PO of the year",
			record: func() *core.Record {
				return core.NewRecord(poCollection)
			}(),
			setup: func(t *testing.T, app *tests.TestApp) {
				// Delete all POs from current period to ensure we're starting fresh
				records, err := app.FindRecordsByFilter(
					"purchase_orders",
					`po_number ~ '{:prefix}-%'`,
					"",
					0,
					0,
					dbx.Params{"prefix": currentPoPrefix},
				)
				if err != nil {
					t.Fatalf("failed to find current period POs: %v", err)
				}
				for _, record := range records {
					if err := app.Delete(record); err != nil {
						t.Fatalf("failed to delete existing PO: %v", err)
					}
				}
			},
			expected: fmt.Sprintf("%s0001", currentPoPrefix),
		},
		{
			// Test Case 7: Parent PO not found
			// This test verifies that when creating a child PO:
			// - If the parent PO does not exist
			// - An appropriate error is returned
			name: "parent PO not found",
			record: func() *core.Record {
				r := core.NewRecord(poCollection)
				r.Set("parent_po", "nonexistent")
				return r
			}(),
			expectedError: "parent PO not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t, app)
			}

			if tt.cleanup != nil {
				defer tt.cleanup(t, app)
			}

			var result string
			var err error
			if tt.year != 0 {
				result, err = routes.GeneratePONumber(app, tt.record, tt.year, tt.month)
			} else {
				// default to current year/month
				result, err = routes.GeneratePONumber(app, tt.record)
			}

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.expectedError)
					return
				}
				if err.Error() != tt.expectedError {
					t.Errorf("expected error %q, got %q", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
