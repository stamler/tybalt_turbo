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
	// user only has the po_approver claim with no payload, and has no second tier
	// claims
	approverToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz") // Fakesy Manjor
	if err != nil {
		t.Fatal(err)
	}

	// Generate token for Horace Silver (creator of several Unapproved POs) This
	// user has the po_approver claim with a payload that includes restrictions
	// and also the po_approver_tier2 claim with no payload.
	creatorToken, err := testutils.GenerateRecordToken("users", "author@soup.com") // Horace Silver
	if err != nil {
		t.Fatal(err)
	}

	// A user with the po_approver claim but no purchase_orders records assigned to them as approver
	approverTokenNoPOs, err := testutils.GenerateRecordToken("users", "orphan@poapprover.com")
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
				`"totalItems":5`, // 5 active purchase_orders in the test database
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
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
