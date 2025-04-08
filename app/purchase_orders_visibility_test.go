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
				`"totalItems":7`, // 7 active purchase_orders in the test database
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

		// Priority second approver can see an Unapproved PO they're assigned to
		{
			Name:   "priority second approver can see an Unapproved PO they're assigned to",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/n9ev1x7a00c1iy6", // Unapproved PO with Tier Two as priority_second_approver
			Headers: map[string]string{
				"Authorization": prioritySecondApproverToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"n9ev1x7a00c1iy6"`,
				`"status":"Unapproved"`,
				`"priority_second_approver":"6bq4j0eb26631dy"`, // Tier Two's ID
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		// User with second_approver_claim CAN see an Unapproved PO with a different priority_second_approver after 24h
		{
			Name:   "user with second_approver_claim CAN see an Unapproved PO with different priority_second_approver AFTER 24 hours if the PO is in the same tier as the user",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/n9ev1x7a00c1iy6", // Unapproved PO with Tier Two as priority_second_approver
			Headers: map[string]string{
				"Authorization": tier2bToken, // Tier TwoB has is not the priority_second_approver
			},
			ExpectedStatus: http.StatusOK, // Should be able to see it after 24 hours
			ExpectedContent: []string{
				`"id":"n9ev1x7a00c1iy6"`,
				`"status":"Unapproved"`,
				`"priority_second_approver":"6bq4j0eb26631dy"`, // Tier Two's ID
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		// User with second_approver_claim CANNOT see an Unapproved PO with a different priority_second_approver after 24h if the PO is in a lower tier
		{
			Name:   "user with second_approver_claim CANNOT see an Unapproved PO with different priority_second_approver AFTER 24 hours if the PO is in a lower tier than the user",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/l9w1z13mm3srtoo", // Unapproved PO with approval_total less than 500
			Headers: map[string]string{
				"Authorization": tier2bToken, // Tier TwoB has is not the priority_second_approver
			},
			ExpectedStatus: http.StatusNotFound, // Should NOT be able to see it even after 24 hours because it's in a lower tier
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
				`"status":404`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		// User with second_approver_claim CANNOT see an Unapproved PO with a different priority_second_approver within 24h
		{
			Name:   "user with second_approver_claim CANNOT see an Unapproved PO with different priority_second_approver WITHIN 24 hours",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/n9ev1x7a00c1iy6", // Unapproved PO with Tier Two as priority_second_approver
			Headers: map[string]string{
				"Authorization": tier2bToken, // Tier TwoB  is not the priority_second_approver
			},
			ExpectedStatus: http.StatusNotFound, // Should NOT be able to see it within 24 hours
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
				`"status":404`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := testutils.SetupTestApp(t)

				// Update the PO's timestamp to be within the past 24 hours (2 hours ago)
				_, err := app.NonconcurrentDB().NewQuery("UPDATE purchase_orders SET updated = datetime('now', '-2 hours') WHERE id = 'n9ev1x7a00c1iy6'").Execute()
				if err != nil {
					t.Fatalf("Failed to update timestamp: %v", err)
				}

				return app
			},
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
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
