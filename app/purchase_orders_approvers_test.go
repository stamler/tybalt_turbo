package main

import (
	"fmt"
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestPurchaseOrdersApproversRoutes(t *testing.T) {
	// Generate tokens for different user types to test different scenarios
	regularUserToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	poApproverToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com") // Fatty Maclean has po_approver claim
	if err != nil {
		t.Fatal(err)
	}

	tier2Token, err := testutils.GenerateRecordToken("users", "author@soup.com") // Horace Silver has po_approver_tier2 claim
	if err != nil {
		t.Fatal(err)
	}

	tier3Token, err := testutils.GenerateRecordToken("users", "hal@2005.com") // Shallow Hal has po_approver_tier3 claim
	if err != nil {
		t.Fatal(err)
	}

	// Get approval tier amounts from the database for test validation
	app := testutils.SetupTestApp(t)
	tier1, tier2, tier3 := testutils.GetApprovalTiers(app)

	// Municipal division ID for testing
	municipalDivision := "2rrfy6m2c8hazjy"
	drillingServicesDivision := "fy4i9poneukvq9u"

	scenarios := []tests.ApiScenario{
		// Tests for GET /api/purchase_orders/approvers/{division}/{amount}
		{
			Name:   "regular user can retrieve a list of approvers for their division",
			Method: http.MethodGet,
			URL:    fmt.Sprintf("/api/purchase_orders/approvers/%s/500", municipalDivision),
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"wegviunlyr2jjjv"`, // Fakesy Manjor (has null payload)
				`"id":"66ct66w380ob6w8"`, // Shallow Hal (has null payload)
				`"given_name"`,
				`"surname"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "po_approver receives empty list of approvers (will auto-set to self in UI) if they have no division restriction",
			Method: http.MethodGet,
			URL:    fmt.Sprintf("/api/purchase_orders/approvers/%s/500", drillingServicesDivision),
			Headers: map[string]string{
				"Authorization": poApproverToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`[]`, // Empty array
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "invalid amount returns error",
			Method: http.MethodGet,
			URL:    fmt.Sprintf("/api/purchase_orders/approvers/%s/invalid", municipalDivision),
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"invalid_amount"`,
				`"message":"Amount must be a valid number"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		// Tests for GET /api/purchase_orders/second_approvers/{division}/{amount}
		{
			Name:   "amount_below_tier_1_returns_empty_list",
			Method: http.MethodGet,
			URL:    fmt.Sprintf("/api/purchase_orders/second_approvers/%s/%d", municipalDivision, int(tier1)-1),
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`[]`, // Empty array for amounts below tier 1
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "tier1_amount_returns_tier2_approvers",
			Method: http.MethodGet,
			URL:    fmt.Sprintf("/api/purchase_orders/second_approvers/%s/%d", municipalDivision, int(tier1)+1),
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"f2j5a8vk006baub"`, // Horace Silver has tier2 claim
				`"given_name":"Horace"`,
				`"surname":"Silver"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "tier2_amount_returns_tier3_approvers",
			Method: http.MethodGet,
			URL:    fmt.Sprintf("/api/purchase_orders/second_approvers/%s/%d", municipalDivision, int(tier2)+1),
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"66ct66w380ob6w8"`, // Shallow Hal has tier3 claim
				`"given_name":"Shallow"`,
				`"surname":"Hal"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user_with_tier2_claim_receives_empty_list_for_tier1_amount",
			Method: http.MethodGet,
			URL:    fmt.Sprintf("/api/purchase_orders/second_approvers/%s/%d", municipalDivision, int(tier1)+1),
			Headers: map[string]string{
				"Authorization": tier2Token,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`[]`, // Empty array (will auto-set to self in UI)
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user_with_tier3_claim_receives_empty_list_for_tier2_amount",
			Method: http.MethodGet,
			URL:    fmt.Sprintf("/api/purchase_orders/second_approvers/%s/%d", municipalDivision, int(tier2)+1),
			Headers: map[string]string{
				"Authorization": tier3Token,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`[]`, // Empty array (will auto-set to self in UI)
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "super_high_amount_returns_empty_list",
			Method: http.MethodGet,
			URL:    fmt.Sprintf("/api/purchase_orders/second_approvers/%s/%d", municipalDivision, int(tier3)+10000),
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`[]`, // Empty array as no one is qualified to approve amounts above the highest tier
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "invalid_amount_returns_error",
			Method: http.MethodGet,
			URL:    fmt.Sprintf("/api/purchase_orders/second_approvers/%s/invalid", municipalDivision),
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"invalid_amount"`,
				`"message":"Amount must be a valid number"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
