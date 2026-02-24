package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
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
	t.Cleanup(app.Cleanup)
	tier1, tier2 := testutils.GetApprovalTiers(app)
	computerKind, err := app.FindFirstRecordByFilter("expenditure_kinds", "name = {:name}", dbx.Params{
		"name": "computer",
	})
	if err != nil {
		t.Fatalf("failed to load computer expenditure kind: %v", err)
	}
	computerKindID := computerKind.Id
	capitalKind, err := app.FindFirstRecordByFilter("expenditure_kinds", "name = {:name}", dbx.Params{
		"name": "capital",
	})
	if err != nil {
		t.Fatalf("failed to load capital expenditure kind: %v", err)
	}
	capitalKindID := capitalKind.Id
	projectKind, err := app.FindFirstRecordByFilter("expenditure_kinds", "name = {:name}", dbx.Params{
		"name": "project",
	})
	if err != nil {
		t.Fatalf("failed to load project expenditure kind: %v", err)
	}
	projectKindID := projectKind.Id

	// Municipal division ID for testing
	municipalDivision := "2rrfy6m2c8hazjy"
	drillingServicesDivision := "fy4i9poneukvq9u"
	makeApproversURLWithKindAndJob := func(path string, division string, amount string, kindID string, hasJob bool) string {
		params := url.Values{}
		params.Set("division", division)
		params.Set("amount", amount)
		params.Set("kind", kindID)
		params.Set("has_job", fmt.Sprintf("%t", hasJob))
		return fmt.Sprintf("%s?%s", path, params.Encode())
	}
	makeApproversURL := func(path string, division string, amount string) string {
		return makeApproversURLWithKindAndJob(path, division, amount, capitalKindID, false)
	}

	scenarios := []tests.ApiScenario{
		// Tests for GET /api/purchase_orders/approvers
		{
			Name:   "regular user can retrieve a list of approvers for their division",
			Method: http.MethodGet,
			URL:    makeApproversURL("/api/purchase_orders/approvers", municipalDivision, "500"),
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"wegviunlyr2jjjv"`, // Fakesy Manjor
				//`"id":"66ct66w380ob6w8"`, // Shallow Hal, removed since the max_amount is too high
				`"id":"4r70mfovf22m9uh"`, // Orphaned POApprover
				`"given_name"`,
				`"surname"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "po_approver receives first-stage pool (including self) when qualified for first approval",
			Method: http.MethodGet,
			URL:    makeApproversURL("/api/purchase_orders/approvers", drillingServicesDivision, "500"),
			Headers: map[string]string{
				"Authorization": poApproverToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"etysnrlup2f6bak"`, // Fatty Maclean
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "first approvers returns empty list when no first-stage approver can approve the amount",
			Method: http.MethodGet,
			URL: makeApproversURLWithKindAndJob(
				"/api/purchase_orders/approvers",
				municipalDivision,
				"100",
				computerKindID,
				false,
			),
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`[]`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "invalid amount returns error",
			Method: http.MethodGet,
			URL:    makeApproversURL("/api/purchase_orders/approvers", municipalDivision, "invalid"),
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
		// Tests for GET /api/purchase_orders/second_approvers
		{
			Name:   "amount below tier1 returns empty list for second approvers call",
			Method: http.MethodGet,
			URL:    makeApproversURL("/api/purchase_orders/second_approvers", municipalDivision, fmt.Sprintf("%d", int(tier1)-1)),
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"approvers":[]`,
				`"second_approval_required":false`,
				`"requester_qualifies":false`,
				`"status":"not_required"`,
				`"reason_code":"second_approval_not_required"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "amount exceeding first threshold returns only approvers with max_amount less than or equal to second threshold for second approvers call",
			Method: http.MethodGet,
			URL:    makeApproversURL("/api/purchase_orders/second_approvers", municipalDivision, fmt.Sprintf("%d", int(tier1)+1)),
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"status":"candidates_available"`,
				`"reason_code":"eligible_second_approvers_available"`,
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
			URL:    makeApproversURL("/api/purchase_orders/second_approvers", municipalDivision, fmt.Sprintf("%d", int(tier2)+1)),
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"status":"candidates_available"`,
				`"reason_code":"eligible_second_approvers_available"`,
				`"id":"66ct66w380ob6w8"`, // Shallow Hal has tier3 claim
				`"given_name":"Shallow"`,
				`"surname":"Hal"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user with max_amount between first and second thresholds receives empty list for amount in within their max_amount and division restrictions for second approvers call",
			Method: http.MethodGet,
			URL:    makeApproversURL("/api/purchase_orders/second_approvers", drillingServicesDivision, fmt.Sprintf("%d", int(tier1)+1)),
			Headers: map[string]string{
				"Authorization": authorSoupToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"approvers":[]`,
				`"second_approval_required":true`,
				`"requester_qualifies":true`,
				`"status":"requester_qualifies"`,
				`"reason_code":"requester_is_eligible_second_approver"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user with max_amount between first and second thresholds receives non-empty list for amount in within their max_amount but outside of their division restrictions for second approvers call",
			Method: http.MethodGet,
			URL:    makeApproversURL("/api/purchase_orders/second_approvers", municipalDivision, fmt.Sprintf("%d", int(tier1)+1)),
			Headers: map[string]string{
				"Authorization": authorSoupToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"status":"candidates_available"`,
				`"reason_code":"eligible_second_approvers_available"`,
				`"id":"6bq4j0eb26631dy"`, // Tier Two has max_amount of 2500 with no restrictions
				`"id":"t4g84hfvkt1v9j3"`, // Tier TwoB has max_amount of 2500 with no restrictions
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		{
			Name:   "user with tier3 claim receives empty list for tier2 amount for second approvers call",
			Method: http.MethodGet,
			URL:    makeApproversURL("/api/purchase_orders/second_approvers", municipalDivision, fmt.Sprintf("%d", int(tier2)+1)),
			Headers: map[string]string{
				"Authorization": tier3Token,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"approvers":[]`,
				`"second_approval_required":true`,
				`"requester_qualifies":true`,
				`"status":"requester_qualifies"`,
				`"reason_code":"requester_is_eligible_second_approver"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "super high amount returns second_pool_empty when no second approver can final-approve",
			Method: http.MethodGet,
			URL:    makeApproversURL("/api/purchase_orders/second_approvers", municipalDivision, fmt.Sprintf("%d", int(tier2)+1000000)),
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"second_pool_empty"`,
				`"message":"no second-stage approvers can final-approve this amount; contact an administrator"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "second approvers metadata uses project_max for project kind",
			Method: http.MethodGet,
			URL: makeApproversURLWithKindAndJob(
				"/api/purchase_orders/second_approvers",
				municipalDivision,
				fmt.Sprintf("%d", int(tier2)+1),
				projectKindID,
				true,
			),
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"limit_column":"project_max"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "second approvers metadata uses computer_max for computer kind with job",
			Method: http.MethodGet,
			URL: makeApproversURLWithKindAndJob(
				"/api/purchase_orders/second_approvers",
				municipalDivision,
				"100",
				computerKindID,
				true,
			),
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"limit_column":"computer_max"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "invalid amount returns error for second approvers call",
			Method: http.MethodGet,
			URL:    makeApproversURL("/api/purchase_orders/second_approvers", municipalDivision, "invalid"),
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

// po_approver_props.divisions: prevent including inactive divisions on update
func TestPoApproverPropsUpdate_InactiveDivisionFails(t *testing.T) {
	// Use superuser token to bypass admin-only restriction
	recordToken, err := testutils.GenerateAdminToken("test@example.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "updating po_approver_props divisions to include inactive fails",
			Method: http.MethodPatch,
			URL:    "/api/collections/po_approver_props/records/1zj39f66eq5qxc4",
			Body: strings.NewReader(`{
                "divisions": ["hcd86z57zjty6jo", "apkev2ow1zjtm7w"]
            }`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"divisions":{"code":"not_active"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// po_approver_props.user_claim: prevent duplicate rows for same user_claim on create
func TestPoApproverPropsCreate_DuplicateUserClaimFails(t *testing.T) {
	recordToken, err := testutils.GenerateAdminToken("test@example.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "creating po_approver_props with duplicate user_claim fails",
			Method: http.MethodPost,
			URL:    "/api/collections/po_approver_props/records",
			Body: strings.NewReader(`{
                "user_claim": "dupucclaim00001",
                "max_amount": 1500,
                "project_max": 0,
                "sponsorship_max": 0,
                "staff_and_social_max": 0,
                "media_and_event_max": 0,
                "computer_max": 0,
                "divisions": []
            }`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"duplicate_user_claim"`,
				`"po_approver_props already exists for this user_claim"`,
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
