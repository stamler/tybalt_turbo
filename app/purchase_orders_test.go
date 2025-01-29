package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"testing"
	"time"
	"tybalt/hooks"
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

	// Generate token for user with smg claim
	smgApproverToken, err := testutils.GenerateRecordToken("users", "hal@2005.com")
	if err != nil {
		t.Fatal(err)
	}

	// Generate token for user with vp claim
	vpApproverToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	// Get current year for PO number validation
	currentYear := time.Now().Format("2006")
	// Get current date for approval timestamp validation
	currentDate := time.Now().Format("2006-01-02")

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
				"OnRecordCreate": 1,
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
				"OnRecordCreate": 1,
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
		   When a user with the po_approver claim (empty payload = all divisions) creates a PO:
		   1. The PO should be auto-approved immediately:
		      - approved timestamp should be set to current date/time
		      - approver should be set to the creator's ID
		   2. Status should become "Active" (since no second approval needed for low value)
		   3. PO number should be generated (format: YYYY-NNNN)

		   Test setup:
		   - Uses user wegviunlyr2jjjv (fakemanager@fakesite.xyz) who has po_approver claim
		   - Sets PO total to random value below MANAGER_PO_LIMIT to avoid triggering second approval
		   - Uses correct auth token matching the creator's ID

		   Verification points:
		   - approved: Checks timestamp starts with current date
		   - status: Must be "Active"
		   - po_number: Must start with current year
		   - approver: Must be creator's ID (wegviunlyr2jjjv)
		*/
		{
			Name:   "purchase order is auto-approved when creator has po_approver claim",
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
			}`, rand.Float64()*(hooks.MANAGER_PO_LIMIT-1.0)+1.0)),
			Headers:        map[string]string{"Authorization": poApproverToken},
			ExpectedStatus: 200,
			ExpectedContent: func() []string {
				if hooks.POAutoApprove {
					return []string{
						fmt.Sprintf(`"approved":"%s`, currentDate), // Should have an approval timestamp starting with today's date
						`"status":"Active"`,
						fmt.Sprintf(`"po_number":"%s-`, currentYear), // Should have a PO number starting with current year
						`"approver":"wegviunlyr2jjjv"`,               // Creator becomes approver
					}
				} else {
					return []string{
						`"approved":""`,
						`"status":"Unapproved"`,
						`"po_number":""`,
						`"approver":"etysnrlup2f6bak"`, // Original approver remains
					}
				}
			}(),
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		/*
		   These tests verify division-specific auto-approval for purchase orders.
		   User fatt@mac.com (id: etysnrlup2f6bak) has po_approver claim with payload:
		   ["hcd86z57zjty6jo", "fy4i9poneukvq9u", "vccd5fo56ctbigh"]

		   Test 1 (Success case):
		   - Creates PO with division "vccd5fo56ctbigh" (in user's po_approver claim payload)
		   - Should auto-approve since user has permission for this division
		   - Verifies: approval timestamp, Active status, PO number generation
		   - Creator becomes approver

		   Test 2 (Failure case):
		   - Creates PO with division "ngpjzurmkrfl8fo" (not in user's po_approver claim payload)
		   - Uses wegviunlyr2jjjv as approver (has empty po_approver claim payload = all divisions)
		   - Should succeed (200) but not auto-approve
		   - Verifies: no approval, Unapproved status, original approver remains

		   Both tests:
		   - Use random total below MANAGER_PO_LIMIT (500) to avoid second approval
		   - Use correct auth token for fatt@mac.com
		   - Match uid to authenticated user's ID
		*/
		{
			Name:   "purchase order is auto-approved when creator has po_approver claim for matching division",
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
			}`, rand.Float64()*(hooks.MANAGER_PO_LIMIT-1.0)+1.0)),
			Headers:        map[string]string{"Authorization": divisionApproverToken},
			ExpectedStatus: 200,
			ExpectedContent: func() []string {
				if hooks.POAutoApprove {
					return []string{
						fmt.Sprintf(`"approved":"%s`, currentDate), // Should have an approval timestamp starting with today's date
						`"status":"Active"`,
						fmt.Sprintf(`"po_number":"%s-`, currentYear), // Should have a PO number starting with current year
						`"approver":"etysnrlup2f6bak"`,               // Creator becomes approver
					}
				} else {
					return []string{
						`"approved":""`,
						`"status":"Unapproved"`,
						`"po_number":""`,
						`"approver":"etysnrlup2f6bak"`, // Original approver remains
					}
				}
			}(),
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 1,
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
			}`, rand.Float64()*(hooks.MANAGER_PO_LIMIT-1.0)+1.0)),
			Headers:        map[string]string{"Authorization": divisionApproverToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,                // Should not be approved
				`"status":"Unapproved"`,        // Status should remain Unapproved
				`"approver":"wegviunlyr2jjjv"`, // Original approver should remain
				`"po_number":""`,               // No PO number should be generated
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		/*
		   This test verifies auto-approval of high-value purchase orders by users with elevated claims.
		   User hal@2005.com (id: 66ct66w380ob6w8) has:
		   - po_approver claim with empty payload (can approve for any division)
		   - smg claim (can provide second approval for high-value POs)

		   Test verifies that when this user creates a high-value PO:
		   1. First approval is automatic (due to po_approver claim)
		   2. Second approval is also automatic (due to smg claim)
		   3. Status becomes Active and PO number is generated
		   4. Creator is set as both approver and second_approver

		   The test:
		   - Uses total above VP_PO_LIMIT (2500) to trigger second approval requirement
		   - Uses random division (since user has empty po_approver payload)
		   - Verifies all approval fields and timestamps
		*/
		{
			Name:   "purchase order is fully auto-approved when creator has po_approver and smg claims",
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
			}`, rand.Float64()*(1000.0)+hooks.VP_PO_LIMIT)), // Random value > VP_PO_LIMIT
			Headers:        map[string]string{"Authorization": smgApproverToken},
			ExpectedStatus: 200,
			ExpectedContent: func() []string {
				if hooks.POAutoApprove {
					return []string{
						fmt.Sprintf(`"approved":"%s`, currentDate),
						fmt.Sprintf(`"second_approval":"%s`, currentDate),
						`"status":"Active"`,
						fmt.Sprintf(`"po_number":"%s-`, currentYear),
						`"approver":"66ct66w380ob6w8"`,
						`"second_approver":"66ct66w380ob6w8"`,
					}
				} else {
					return []string{
						`"approved":""`,
						`"second_approval":""`,
						`"status":"Unapproved"`,
						`"po_number":""`,
						`"approver":"etysnrlup2f6bak"`,
					}
				}
			}(),
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		/*
		   This test verifies auto-approval of mid-range value purchase orders by users with VP claim.
		   User author@soup.com (id: f2j5a8vk006baub) has:
		   - po_approver claim with empty payload (can approve for any division)
		   - vp claim (can provide second approval for POs between MANAGER_PO_LIMIT and VP_PO_LIMIT)

		   Test verifies that when this user creates a mid-range value PO:
		   1. First approval is automatic (due to po_approver claim)
		   2. Second approval is also automatic (due to vp claim)
		   3. Status becomes Active and PO number is generated
		   4. Creator is set as both approver and second_approver

		   The test:
		   - Uses total between MANAGER_PO_LIMIT (500) and VP_PO_LIMIT (2500)
		   - Uses random division (since user has empty po_approver payload)
		   - Verifies all approval fields and timestamps
		*/
		{
			Name:   "purchase order is fully auto-approved when creator has po_approver and vp claims",
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
			}`, rand.Float64()*(hooks.VP_PO_LIMIT-hooks.MANAGER_PO_LIMIT)+hooks.MANAGER_PO_LIMIT)), // Random value between MANAGER_PO_LIMIT and VP_PO_LIMIT
			Headers:        map[string]string{"Authorization": vpApproverToken},
			ExpectedStatus: 200,
			ExpectedContent: func() []string {
				if hooks.POAutoApprove {
					return []string{
						fmt.Sprintf(`"approved":"%s`, currentDate),        // Should have an approval timestamp
						fmt.Sprintf(`"second_approval":"%s`, currentDate), // Should have a second approval timestamp
						`"status":"Active"`,
						fmt.Sprintf(`"po_number":"%s-`, currentYear), // Should have a PO number
						`"approver":"f2j5a8vk006baub"`,               // Creator becomes approver
						`"second_approver":"f2j5a8vk006baub"`,        // Creator also becomes second approver
					}
				} else {
					return []string{
						`"approved":""`,
						`"second_approval":""`,
						`"status":"Unapproved"`,
						`"po_number":""`,
						`"approver":"etysnrlup2f6bak"`, // Original approver remains
					}
				}
			}(),
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 1,
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

func TestPurchaseOrdersRoutes(t *testing.T) {
	unauthorizedToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	nonCloseToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	closeToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}

	poApproverToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	// Token for VP who can do second approvals
	vpToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	// Token for user with smg claim
	smgToken, err := testutils.GenerateRecordToken("users", "hal@2005.com")
	if err != nil {
		t.Fatal(err)
	}

	currentDate := time.Now().Format("2006-01-02")
	currentYear := time.Now().Format("2006")

	scenarios := []tests.ApiScenario{
		{
			Name:   "authorized approver successfully approves PO below MANAGER_PO_LIMIT",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/gal6e5la2fa4rpn/approve", // Using existing Unapproved PO with total 329.01
			Body:   strings.NewReader(`{}`),                        // No body needed for approval
			Headers: map[string]string{
				"Authorization": poApproverToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				fmt.Sprintf(`"approved":"%s`, currentDate),   // Should have today's date
				fmt.Sprintf(`"po_number":"%s-`, currentYear), // Should start with current year
				`"status":"Active"`,                          // Status should be Active
				`"approver":"etysnrlup2f6bak"`,               // Should be set to the approver's ID
			},
			ExpectedEvents: map[string]int{
				"OnModelAfterUpdateSuccess": 1,
				"OnModelUpdate":             1,
				"OnRecordUpdate":            1,
				"OnRecordValidate":          1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "first approval of high-value PO leaves status as Unapproved",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/46efdq319b22480/approve", // Using existing Unapproved PO with total 862.12
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": poApproverToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				fmt.Sprintf(`"approved":"%s`, currentDate), // Should have today's date
				`"status":"Unapproved"`,                    // Status should remain Unapproved
				`"po_number":""`,                           // No PO number yet
				`"approver":"etysnrlup2f6bak"`,             // Approver changes to match caller (fatt@mac.com)
				`"second_approver":""`,                     // No second approver yet
				`"second_approval":""`,                     // No second approval timestamp yet
			},
			ExpectedEvents: map[string]int{
				"OnModelAfterUpdateSuccess": 1,
				"OnModelUpdate":             1,
				"OnRecordUpdate":            1,
				"OnRecordValidate":          1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "VP completes both approvals of high-value PO in single call",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/46efdq319b22480/approve", // Using existing Unapproved PO with total 862.12
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": vpToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				fmt.Sprintf(`"approved":"%s`, currentDate),        // Should have today's date
				fmt.Sprintf(`"second_approval":"%s`, currentDate), // Should have same timestamp
				`"status":"Active"`,                               // Status should become Active
				fmt.Sprintf(`"po_number":"%s-`, currentYear),      // Should get PO number
				`"approver":"f2j5a8vk006baub"`,                    // VP becomes first approver
				`"second_approver":"f2j5a8vk006baub"`,             // VP also becomes second approver
			},
			ExpectedEvents: map[string]int{
				"OnModelAfterUpdateSuccess": 1,
				"OnModelUpdate":             1,
				"OnRecordUpdate":            1,
				"OnRecordValidate":          1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "second approval of high-value PO completes approval process",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/2blv18f40i2q373/approve", // Using PO with first approval and total 1022.69
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": vpToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"approved":"2025-01-29 14:22:29.563Z"`,           // Should keep original first approval
				fmt.Sprintf(`"second_approval":"%s`, currentDate), // Should have today's date
				`"status":"Active"`,                               // Status should become Active
				fmt.Sprintf(`"po_number":"%s-`, currentYear),      // Should get PO number
				`"approver":"wegviunlyr2jjjv"`,                    // Should keep original approver
				`"second_approver":"f2j5a8vk006baub"`,             // Should be set to VP's ID
			},
			ExpectedEvents: map[string]int{
				"OnModelAfterUpdateSuccess": 1,
				"OnModelUpdate":             1,
				"OnRecordUpdate":            1,
				"OnRecordValidate":          1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "unauthorized user cannot approve purchase order",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/gal6e5la2fa4rpn/approve",
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": unauthorizedToken,
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized_approval"`,
				`"message":"you are not authorized to approve this purchase order"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "po_approver cannot approve PO in unauthorized division",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/j6nn3v7s2s6d6u8/approve", // PO in division 2rrfy6m2c8hazjy
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": poApproverToken, // fatt@mac.com who can't approve purchase orders in this division
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized_approval"`,
				`"message":"you are not authorized to approve this purchase order"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "caller with the payables_admin claim can convert Active Normal purchase_orders to Cumulative",
			Method:          http.MethodPost,
			URL:             "/api/purchase_orders/2plsetqdxht7esg/make_cumulative",
			Headers:         map[string]string{"Authorization": closeToken},
			ExpectedStatus:  204,
			ExpectedContent: []string{},
			ExpectedEvents: map[string]int{
				"OnRecordUpdate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "caller with the payables_admin claim cannot convert non-Active Normal purchase_orders to Cumulative",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/gal6e5la2fa4rpn/make_cumulative",
			Headers:        map[string]string{"Authorization": closeToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"po_not_active"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate": 0,
				"OnModelAfterUpdate":  0,
				"OnBeforeApiError":    0,
				"OnAfterApiError":     0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "caller with the payables_admin claim cannot convert Active non-Normal purchase_orders to Cumulative",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/ly8xyzpuj79upq1/make_cumulative",
			Headers:        map[string]string{"Authorization": closeToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"po_not_normal"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate": 0,
				"OnModelAfterUpdate":  0,
				"OnBeforeApiError":    0,
				"OnAfterApiError":     0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "caller without the payables_admin claim cannot convert Active Normal purchase_orders to Cumulative",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/2plsetqdxht7esg/make_cumulative",
			Headers:        map[string]string{"Authorization": nonCloseToken},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"code":"unauthorized_conversion"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate": 0,
				"OnModelAfterUpdate":  0,
				"OnBeforeApiError":    0,
				"OnAfterApiError":     0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "caller with the payables_admin claim can cancel Active purchase_orders records with no expenses against them",
			Method:          http.MethodPost,
			URL:             "/api/purchase_orders/2plsetqdxht7esg/cancel",
			Headers:         map[string]string{"Authorization": closeToken},
			ExpectedStatus:  204,
			ExpectedContent: []string{},
			ExpectedEvents: map[string]int{
				"OnRecordUpdate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "caller without the payables_admin claim cannot cancel Active purchase_orders records with no expenses against them",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/2plsetqdxht7esg/cancel",
			Headers:        map[string]string{"Authorization": nonCloseToken},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"code":"unauthorized_cancellation"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate": 0,
				"OnModelAfterUpdate":  0,
				"OnBeforeApiError":    0,
				"OnAfterApiError":     0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "caller without the payables_admin claim cannot close Active Cumulative purchase_orders records",
			Method:          http.MethodPost,
			URL:             "/api/purchase_orders/ly8xyzpuj79upq1/close",
			Headers:         map[string]string{"Authorization": nonCloseToken},
			ExpectedStatus:  403,
			ExpectedContent: []string{`"code":"unauthorized_closure","message":"you are not authorized to close purchase orders"`},
			ExpectedEvents: map[string]int{
				"OnBeforeApiError": 0,
				"OnAfterApiError":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "caller with the payables_admin claim can close Active Cumulative purchase_orders records",
			Method:          http.MethodPost,
			URL:             "/api/purchase_orders/ly8xyzpuj79upq1/close",
			Headers:         map[string]string{"Authorization": closeToken},
			ExpectedStatus:  200,
			ExpectedContent: []string{`"message":"Purchase order closed successfully"`},
			ExpectedEvents: map[string]int{
				"OnBeforeApiError": 0,
				"OnAfterApiError":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "Active Non-Cumulative purchase_orders records cannot be closed",
			Method:          http.MethodPost,
			URL:             "/api/purchase_orders/2plsetqdxht7esg/close",
			Headers:         map[string]string{"Authorization": closeToken},
			ExpectedStatus:  400,
			ExpectedContent: []string{`"code":"invalid_po_type","message":"only cumulative purchase orders can be closed manually"`},
			ExpectedEvents: map[string]int{
				"OnBeforeApiError": 0,
				"OnAfterApiError":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		// TODO: Non-Active Cumulative purchase_orders records cannot be closed

		/*
		   This test verifies that a user cannot perform second approval without the required claims (SMG/VP),
		   even if they have other valid permissions. Specifically, it tests that:

		   Test Data:
		   1. Purchase Order (2blv18f40i2q373):
		      - Division: vccd5fo56ctbigh
		      - Total: 1022.69 (above MANAGER_PO_LIMIT, requiring second approval)
		      - Current Status: Unapproved
		      - Has first approval: Yes (timestamp: 2025-01-29 14:22:29.563Z)
		      - First approver: wegviunlyr2jjjv

		   2. User Attempting Second Approval (fatt@mac.com using poApproverToken):
		      - Has po_approver claim: Yes
		      - Authorized divisions: ["hcd86z57zjty6jo", "fy4i9poneukvq9u", "vccd5fo56ctbigh"]
		      - Has division permission: Yes (PO's division matches user's authorized divisions)
		      - Has SMG claim: No
		      - Has VP claim: No

		   Expected Behavior:
		   - Request should fail with 403 Forbidden
		   - Error should indicate lack of required claim (not division permission)
		   - No changes should be made to the PO
		   - No events should be triggered

		   This test isolates the claim requirement by using a user who has all other necessary permissions
		   (division authorization, po_approver claim) but lacks the specific claims required for second approval.
		   This ensures the failure is specifically due to missing SMG/VP claims, not other permission issues.
		*/
		{
			Name:   "user with po_approver claim but without SMG/VP claims cannot perform second approval",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/2blv18f40i2q373/approve",
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": poApproverToken, // fatt@mac.com who has division permission but no SMG/VP claims
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized_approval"`,
				`"message":"you are not authorized to perform second approval on this purchase order"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "VP cannot second-approve PO with value above VP_PO_LIMIT (SMG claim required)",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/q79eyq0uqrk6x2q/approve", // PO with total 3251.12
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": vpToken, // author@soup.com who has VP claim but not SMG
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized_approval"`,
				`"message":"you are not authorized to perform second approval on this purchase order"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "SMG can second-approve PO with value above VP_PO_LIMIT",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/q79eyq0uqrk6x2q/approve", // PO with total 3251.12
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": smgToken, // hal@2005.com who has SMG claim
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				fmt.Sprintf(`"approved":"%s`, currentDate),        // Should have today's date
				fmt.Sprintf(`"second_approval":"%s`, currentDate), // Should have same timestamp
				`"status":"Active"`,                               // Status should become Active
				fmt.Sprintf(`"po_number":"%s-`, currentYear),      // Should get PO number
				`"approver":"f2j5a8vk006baub"`,                    // approver does not change
				`"second_approver":"66ct66w380ob6w8"`,             // smg holder becomes second approver
			},
			ExpectedEvents: map[string]int{
				"OnModelAfterUpdateSuccess": 1,
				"OnModelUpdate":             1,
				"OnRecordUpdate":            1,
				"OnRecordValidate":          1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "approval attempt on already approved PO fails",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/2plsetqdxht7esg/approve", // Already approved PO (2024-0008)
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": poApproverToken, // Using a valid approver token to isolate the already-approved check
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"po_not_unapproved"`,
				`"message":"only unapproved purchase orders can be approved"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "approval attempt on rejected PO fails",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/l9w1z13mm3srtoo/approve", // Rejected PO with rejection reason
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": poApproverToken, // Using a valid approver token to isolate the rejection check
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"po_rejected"`,
				`"message":"rejected purchase orders cannot be approved"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestGeneratePONumber(t *testing.T) {
	app := testutils.SetupTestApp(t)
	poCollection, err := app.FindCollectionByNameOrId("purchase_orders")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		year          int // 0 for default 2024
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
			// Test Case 3: Parent PO Number Generation
			// This test verifies that when creating a new parent PO:
			// - The next sequential number after the highest existing PO is generated
			// - The number is unique in the database
			//
			// The test:
			// 1. Creates a new parent PO record
			// 2. Generates a PO number based on existing POs in the database
			name: "sequential parent PO",
			record: func() *core.Record {
				r := core.NewRecord(poCollection)
				return r
			}(),
			expected: "2024-0010", // Next after 2024-0009 in test DB
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
			// - The generated number follows the format: Year + "-0001"
			// - The number is unique in the database
			name: "first PO of the year",
			year: 2025,
			record: func() *core.Record {
				return core.NewRecord(poCollection)
			}(),
			setup: func(t *testing.T, app *tests.TestApp) {
				// Delete all POs from 2025 to ensure we're starting fresh
				records, err := app.FindRecordsByFilter(
					"purchase_orders",
					`po_number ~ '{:current_year}-%'`,
					"",
					0,
					0,
					dbx.Params{"current_year": 2025},
				)
				if err != nil {
					t.Fatalf("failed to find 2025 POs: %v", err)
				}
				for _, record := range records {
					if err := app.Delete(record); err != nil {
						t.Fatalf("failed to delete existing PO: %v", err)
					}
				}
			},
			expected: "2025-0001",
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
				result, err = routes.GeneratePONumber(app, tt.record, tt.year)
			} else {
				// default to 2024 since the test DB is seeded with 2024 POs
				result, err = routes.GeneratePONumber(app, tt.record, 2024)
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
