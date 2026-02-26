package main

import (
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestPurchaseOrdersVisibilityRules(t *testing.T) {
	// Generate token for a regular user with no po-related claims
	regularUserToken, err := testutils.GenerateRecordToken("users", "time@test.com") // Tester Time
	if err != nil {
		t.Fatal(err)
	}

	// Generate token for Fakesy Manjor (approver of several Unapproved POs) This
	// user only has the po_approver claim empty divisions property on the
	// po_approver_props record, and has no second tier claims
	approverToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz") // Fakesy Manjor
	if err != nil {
		t.Fatal(err)
	}

	// Generate token for Horace Silver (creator of several Unapproved POs) This
	// user has the po_approver claim with a divisions property on the
	// po_approver_props record that includes restrictions.
	creatorToken, err := testutils.GenerateRecordToken("users", "author@soup.com") // Horace Silver
	if err != nil {
		t.Fatal(err)
	}

	// A user with the po_approver claim but no purchase_orders records assigned to them as approver
	approverTokenNoPOs, err := testutils.GenerateRecordToken("users", "orphan@poapprover.com")
	if err != nil {
		t.Fatal(err)
	}

	// Generate token for Tier Two (priority_second_approver of n9ev1x7a00c1iy6)
	prioritySecondApproverToken, err := testutils.GenerateRecordToken("users", "tier2@poapprover.com") // Tier Two
	if err != nil {
		t.Fatal(err)
	}

	// Generate token for Tier TwoB (has max_amount 2500, unrestricted divisions)
	tier2bToken, err := testutils.GenerateRecordToken("users", "tier2b@poapprover.com") // Tier TwoB
	if err != nil {
		t.Fatal(err)
	}

	// Generate token for inactive Tier TwoB-like user (active=0 in fixture)
	inactiveTier2bToken, err := testutils.GenerateRecordToken("users", "inactive2@poapprover.com")
	if err != nil {
		t.Fatal(err)
	}

	// Generate token for Hal (project_max and max_amount both 1000000)
	halToken, err := testutils.GenerateRecordToken("users", "hal@2005.com")
	if err != nil {
		t.Fatal(err)
	}

	// Generate token for Fatt (max_amount=500 but computer_max=2500)
	fattToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	// Generate token for a user with no claims (creator of cancelled PO 1cqrvp4mna33k2b)
	noclaimsToken, err := testutils.GenerateRecordToken("users", "noclaims@example.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		// Any authenticated user without special permissions can see all Active purchase_orders records
		{
			Name:   "any authenticated user without special permissions can see all active purchase_orders",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records",
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"totalItems":10`, // 10 purchase_orders visible in the fixture set
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		// User without special claims cannot see a cancelled PO they didn't create
		{
			Name:   "user without purchase_order claims cannot see a cancelled PO they didn't create",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/338568325487lo2", // Cancelled PO created by Horace Silver
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusNotFound, // Should get 404 as they don't have permission
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
				`"status":404`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		// Approver can see an Unapproved PO they're assigned to.
		{
			Name:   "approver can see an Unapproved PO they're assigned to",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/46efdq319b22480", // Unapproved PO with Fakesy Manjor as approver
			Headers: map[string]string{
				"Authorization": approverToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"46efdq319b22480"`,
				`"status":"Unapproved"`,
				`"approver":"wegviunlyr2jjjv"`, // Fakesy Manjor's ID
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "holder of the po_approver claim cannot see Unapproved POs they're not assigned to",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/46efdq319b22480", // Unapproved PO with Fakesy Manjor as approver
			Headers: map[string]string{
				"Authorization": approverTokenNoPOs,
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
				`"status":404`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		// Creator can see their own Unapproved PO
		{
			Name:   "creator can see their own Unapproved PO",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/46efdq319b22480", // Unapproved PO created by Horace Silver
			Headers: map[string]string{
				"Authorization": creatorToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"46efdq319b22480"`,
				`"status":"Unapproved"`,
				`"uid":"f2j5a8vk006baub"`, // Horace Silver's ID
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "creator can see their own rejected unapproved PO via visible API",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible/l9w1z13mm3srtoo",
			Headers: map[string]string{
				"Authorization": creatorToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"l9w1z13mm3srtoo"`,
				`"status":"Unapproved"`,
				`"rejected":"`,
				`"rejection_reason":"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "visible rejected scope returns creator rejected purchase orders",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible?scope=rejected",
			Headers: map[string]string{
				"Authorization": creatorToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"l9w1z13mm3srtoo"`,
				`"rejected":"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		// Priority second approver cannot see an Unapproved PO before first approval occurs
		{
			Name:   "priority second approver cannot see an Unapproved PO before first approval",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/n9ev1x7a00c1iy6", // Unapproved PO with Tier Two as priority_second_approver
			Headers: map[string]string{
				"Authorization": prioritySecondApproverToken,
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
				`"status":404`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		// Non-direct second-stage approver is no longer visible via collection API
		{
			Name:   "non-direct second-stage approver cannot see first-approved PO via collection API",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/2blv18f40i2q373",
			Headers: map[string]string{
				"Authorization": tier2bToken, // Not the priority second approver
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
				`"status":404`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "eligible second-stage approver with project limit can see first-approved PO via visible API",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible/2blv18f40i2q373",
			Headers: map[string]string{
				"Authorization": halToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"2blv18f40i2q373"`,
				`"status":"Unapproved"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		// Holder of po_approver claim cannot see unapproved PO they are not assigned to in stage 1
		{
			Name:   "holder of po_approver claim cannot see unrelated unapproved PO before first approval",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/l9w1z13mm3srtoo", // Unapproved PO with approval_total less than 500
			Headers: map[string]string{
				"Authorization": tier2bToken, // Tier TwoB has is not the priority_second_approver
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
				`"status":404`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		// Non-direct second-stage approver is no longer visible via collection API
		{
			Name:   "non-priority second-stage approver cannot see first-approved PO via collection API",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/stg2within24h01",
			Headers: map[string]string{
				"Authorization": tier2bToken,
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
				`"status":404`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "non-priority second-stage approver can see first-approved PO via visible API",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible/stg2within24h01",
			Headers: map[string]string{
				"Authorization": tier2bToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"stg2within24h01"`,
				`"status":"Unapproved"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		// Creator can see their own Cancelled PO
		{
			Name:   "creator can see their own Cancelled PO",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/1cqrvp4mna33k2b", // Cancelled PO created by noclaims@example.com
			Headers: map[string]string{
				"Authorization": noclaimsToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"1cqrvp4mna33k2b"`,
				`"status":"Cancelled"`,
				`"uid":"4ssj9f1yg250o9y"`, // noclaims@example.com's ID
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		// Approver can see a Cancelled PO they approved
		{
			Name:   "approver can see a Cancelled PO they approved",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/1cqrvp4mna33k2b", // Cancelled PO with orphan@poapprover.com as approver
			Headers: map[string]string{
				"Authorization": approverTokenNoPOs, // orphan@poapprover.com
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"1cqrvp4mna33k2b"`,
				`"status":"Cancelled"`,
				`"approver":"4r70mfovf22m9uh"`, // orphan@poapprover.com's ID
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		// Second approver can see a Cancelled PO they second-approved
		{
			Name:   "second approver can see a Cancelled PO they second-approved",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/1cqrvp4mna33k2b", // Cancelled PO with tier2@poapprover.com as second_approver
			Headers: map[string]string{
				"Authorization": prioritySecondApproverToken, // tier2@poapprover.com
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"1cqrvp4mna33k2b"`,
				`"status":"Cancelled"`,
				`"second_approver":"6bq4j0eb26631dy"`, // tier2@poapprover.com's ID
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		// User without special claims cannot see a cancelled PO they didn't create (for 1cqrvp4mna33k2b)
		{
			Name:   "user without special claims cannot see a cancelled PO they didn't create (new PO)",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/1cqrvp4mna33k2b", // Cancelled PO created by noclaims@example.com
			Headers: map[string]string{
				"Authorization": regularUserToken, // Tester Time has no special claims
			},
			ExpectedStatus: http.StatusNotFound, // Should get 404 as they don't have permission
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
				`"status":404`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		// Approver cannot see a Cancelled PO they didn't approve
		{
			Name:   "approver cannot see a Cancelled PO they didn't approve",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/1cqrvp4mna33k2b", // Cancelled PO with orphan@poapprover.com as approver
			Headers: map[string]string{
				"Authorization": approverToken, // Fakesy Manjor is not the approver of this PO
			},
			ExpectedStatus: http.StatusNotFound, // Should get 404 as they don't have permission
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
				`"status":404`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		// Creator can see their own Closed PO
		{
			Name:   "creator can see their own Closed PO",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/0pia83nnprdlzf8", // Closed PO created by noclaims@example.com
			Headers: map[string]string{
				"Authorization": noclaimsToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"0pia83nnprdlzf8"`,
				`"status":"Closed"`,
				`"uid":"4ssj9f1yg250o9y"`, // noclaims@example.com's ID
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		// Approver can see a Closed PO they approved
		{
			Name:   "approver can see a Closed PO they approved",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/0pia83nnprdlzf8", // Closed PO with orphan@poapprover.com as approver
			Headers: map[string]string{
				"Authorization": approverTokenNoPOs, // orphan@poapprover.com
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"0pia83nnprdlzf8"`,
				`"status":"Closed"`,
				`"approver":"4r70mfovf22m9uh"`, // orphan@poapprover.com's ID
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		// User without special claims cannot see a Closed PO they didn't create
		{
			Name:   "user without special claims cannot see a Closed PO they didn't create",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/0pia83nnprdlzf8", // Closed PO created by noclaims@example.com
			Headers: map[string]string{
				"Authorization": regularUserToken, // Tester Time has no special claims
			},
			ExpectedStatus: http.StatusNotFound, // Should get 404 as they don't have permission
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
				`"status":404`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "visible endpoint requires stale_before when scope is stale",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible?scope=stale",
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"missing_stale_before"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "visible stale scope returns stale active purchase orders",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible?scope=stale&stale_before=2025-01-01%2000:00:00.000Z",
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"2plsetqdxht7esg"`,
				`"status":"Active"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "visible endpoint requires expiring_before when scope is expiring",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible?scope=expiring",
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"missing_expiring_before"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "visible expiring scope returns active recurring purchase orders before cutoff",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible?scope=expiring&expiring_before=2025-10-01",
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"d8463q483f3da28"`,
				`"type":"Recurring"`,
				`"status":"Active"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "pending endpoint behavior remains unchanged for priority second approver",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/pending",
			Headers: map[string]string{
				"Authorization": prioritySecondApproverToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"stg2within24h01"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "pending endpoint includes assigned approver self-bypass for dual-stage records",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/pending",
			Headers: map[string]string{
				"Authorization": creatorToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"01897j210v01f69"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "visible endpoint uses kind-specific approval limit columns",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible/pocompkindvis01",
			Headers: map[string]string{
				"Authorization": fattToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"pocompkindvis01"`,
				`"status":"Unapproved"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "inactive admin profile cannot qualify second-stage visibility",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible/stg2within24h01",
			Headers: map[string]string{
				"Authorization": inactiveTier2bToken,
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"code":"po_not_found_or_not_visible"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
