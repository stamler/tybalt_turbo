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

	authorSoupToken, err := testutils.GenerateRecordToken("users", "author@soup.com") // Horace Silver has {"divisions":["hcd86z57zjty6jo","fy4i9poneukvq9u"],"max_amount":2500}
	if err != nil {
		t.Fatal(err)
	}

	tier3Token, err := testutils.GenerateRecordToken("users", "hal@2005.com") // Shallow Hal has {"max_amount":1000000}
	if err != nil {
		t.Fatal(err)
	}

	// Get approval tier amounts from the database for test validation
	app := testutils.SetupTestApp(t)
	tier1, tier2 := testutils.GetApprovalTiers(app)

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
				//`"id":"66ct66w380ob6w8"`, // Shallow Hal (has null payload), removed since the max_amount is too high
				`"id":"4r70mfovf22m9uh"`, // Orphaned POApprover (has null payload)
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
			Name:   "amount below tier1 returns empty list for second approvers call",
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
			Name:   "amount exceeding first threshold returns only approvers with max_amount less than or equal to second threshold for second approvers call",
			Method: http.MethodGet,
			URL:    fmt.Sprintf("/api/purchase_orders/second_approvers/%s/%d", municipalDivision, int(tier1)+1),
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"6bq4j0eb26631dy"`, // Tier Two has max_amount of 2500 with no restrictions
				`"given_name":"Tier"`,
				`"surname":"Two"`,
				`"id":"t4g84hfvkt1v9j3"`, // Tier TwoB has max_amount of 2500 with no restrictions
				`"given_name":"Tier"`,
				`"surname":"TwoB"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "tier2 amount returns tier3 approvers for second approvers call",
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
			Name:   "user with max_amount between first and second thresholds receives empty list for amount in within their max_amount and division restrictions for second approvers call",
			Method: http.MethodGet,
			URL:    fmt.Sprintf("/api/purchase_orders/second_approvers/%s/%d", drillingServicesDivision, int(tier1)+1),
			Headers: map[string]string{
				"Authorization": authorSoupToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`[]`, // Empty array (will auto-set to self in UI)
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user with max_amount between first and second thresholds receives non-empty list for amount in within their max_amount but outside of their division restrictions for second approvers call",
			Method: http.MethodGet,
			URL:    fmt.Sprintf("/api/purchase_orders/second_approvers/%s/%d", municipalDivision, int(tier1)+1),
			Headers: map[string]string{
				"Authorization": authorSoupToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"6bq4j0eb26631dy"`, // Tier Two has max_amount of 2500 with no restrictions
				`"id":"t4g84hfvkt1v9j3"`, // Tier TwoB has max_amount of 2500 with no restrictions
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		{
			Name:   "user with tier3 claim receives empty list for tier2 amount for second approvers call",
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
			Name:   "super high amount returns empty list for second approvers call",
			Method: http.MethodGet,
			URL:    fmt.Sprintf("/api/purchase_orders/second_approvers/%s/%d", municipalDivision, int(tier2)+1000000),
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
			Name:   "invalid amount returns error for second approvers call",
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
