package main

import (
	"encoding/json"
	"math"
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestPurchaseOrdersVisibilityRules(t *testing.T) {
	type visiblePORow struct {
		ID              string   `json:"id"`
		Type            string   `json:"type"`
		Total           float64  `json:"total"`
		ApprovalTotal   float64  `json:"approval_total"`
		ExpensesTotal   *float64 `json:"expenses_total"`
		RemainingAmount *float64 `json:"remaining_amount"`
	}

	assertFloatApprox := func(t testing.TB, got float64, want float64) {
		t.Helper()
		if math.Abs(got-want) > 0.0001 {
			t.Fatalf("expected %.4f, got %.4f", want, got)
		}
	}

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

	// Generate token for Shallow Hal (owner self-bypass with non-zero limit but
	// insufficient final limit for the full PO amount).
	poApproverSelfBypassToken, err := testutils.GenerateRecordToken("users", "u_po_bypass_001@example.com")
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
			URL:    "/api/collections/purchase_orders/records?filter=%28status%3D%27Active%27%29",
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"totalItems":15`, // Count is asserted against the current seeded active-PO fixture set.
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
		{
			Name:   "visible approved_by_me_awaiting_second scope returns first-approved purchase orders for the first approver",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible?scope=approved_by_me_awaiting_second",
			Headers: map[string]string{
				"Authorization": approverToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"2blv18f40i2q373"`,
				`"status":"Unapproved"`,
				`"approved":"`,
			},
			NotExpectedContent: []string{
				`"id":"46efdq319b22480"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "rejector can still see rejected unapproved PO via visible API",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible/l9w1z13mm3srtoo",
			Headers: map[string]string{
				"Authorization": approverToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"l9w1z13mm3srtoo"`,
				`"status":"Unapproved"`,
				`"rejector":"wegviunlyr2jjjv"`,
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
			Name:   "first approver can see first-approved PO via collection API",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders/records/2blv18f40i2q373",
			Headers: map[string]string{
				"Authorization": approverToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"2blv18f40i2q373"`,
				`"status":"Unapproved"`,
				`"approver":"wegviunlyr2jjjv"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "first approver can see first-approved PO via augmented collection API",
			Method: http.MethodGet,
			URL:    "/api/collections/purchase_orders_augmented/records/2blv18f40i2q373",
			Headers: map[string]string{
				"Authorization": approverToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"2blv18f40i2q373"`,
				`"status":"Unapproved"`,
				`"approver":"wegviunlyr2jjjv"`,
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
		{
			Name:   "first approver can still see first-approved PO via visible API",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible/2blv18f40i2q373",
			Headers: map[string]string{
				"Authorization": approverToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"2blv18f40i2q373"`,
				`"status":"Unapproved"`,
				`"approved":"`,
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
			Name:   "visible endpoint rejects limit=0",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible?scope=approved_by_me_awaiting_second&limit=0",
			Headers: map[string]string{
				"Authorization": approverToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"invalid_limit"`,
				`"message":"limit must be a positive integer"`,
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
			Name:   "pending endpoint includes owner self-bypass when caller has non-zero kind limit but cannot final-approve amount",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/pending",
			Headers: map[string]string{
				"Authorization": poApproverSelfBypassToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"poselfbypass01"`,
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
			Name:   "visible by id returns remaining_amount for one-time purchase orders",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible/2plsetqdxht7esg",
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"2plsetqdxht7esg"`,
				`"type":"One-Time"`,
				`"remaining_amount":`,
			},
			TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
				defer res.Body.Close()

				var row visiblePORow
				if err := json.NewDecoder(res.Body).Decode(&row); err != nil {
					t.Fatalf("failed decoding visible PO response: %v", err)
				}
				if row.RemainingAmount == nil || row.ExpensesTotal == nil {
					t.Fatalf("expected remaining_amount and expenses_total in response, got %#v", row)
				}
				assertFloatApprox(t, *row.RemainingAmount, row.Total-*row.ExpensesTotal)
			},
		},
		{
			Name:   "visible by id falls back to total when legacy one-time purchase order has zero approval_total",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible/0pia83nnprdlzf8",
			Headers: map[string]string{
				"Authorization": noclaimsToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"0pia83nnprdlzf8"`,
				`"type":"One-Time"`,
				`"remaining_amount":`,
			},
			TestAppFactory: testutils.SetupTestApp,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				if _, err := app.DB().NewQuery(`
					UPDATE purchase_orders
					SET approval_total = 0
					WHERE id = {:id}
				`).Bind(map[string]any{
					"id": "0pia83nnprdlzf8",
				}).Execute(); err != nil {
					t.Fatalf("failed updating purchase order fixture: %v", err)
				}
			},
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
				defer res.Body.Close()

				var row visiblePORow
				if err := json.NewDecoder(res.Body).Decode(&row); err != nil {
					t.Fatalf("failed decoding visible PO response: %v", err)
				}
				if row.RemainingAmount == nil || row.ExpensesTotal == nil {
					t.Fatalf("expected remaining_amount and expenses_total in response, got %#v", row)
				}
				assertFloatApprox(t, row.ApprovalTotal, 0)
				assertFloatApprox(t, *row.RemainingAmount, row.Total-*row.ExpensesTotal)
				assertFloatApprox(t, *row.RemainingAmount, 0)
			},
		},
		{
			Name:   "visible by id returns provisional remaining_amount for cumulative purchase orders using all linked expenses",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible/ly8xyzpuj79upq1",
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"ly8xyzpuj79upq1"`,
				`"type":"Cumulative"`,
				`"remaining_amount":`,
			},
			TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
				defer res.Body.Close()

				var row visiblePORow
				if err := json.NewDecoder(res.Body).Decode(&row); err != nil {
					t.Fatalf("failed decoding visible PO response: %v", err)
				}
				if row.RemainingAmount == nil || row.ExpensesTotal == nil {
					t.Fatalf("expected remaining_amount and expenses_total in response, got %#v", row)
				}
				assertFloatApprox(t, *row.ExpensesTotal, 2387.12)
				assertFloatApprox(t, *row.RemainingAmount, row.Total-*row.ExpensesTotal)
				assertFloatApprox(t, *row.RemainingAmount, -1040)
			},
		},
		{
			Name:   "visible by id returns remaining_amount for recurring purchase orders based on approval total",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible/d8463q483f3da28",
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"d8463q483f3da28"`,
				`"type":"Recurring"`,
				`"remaining_amount":`,
			},
			TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
				defer res.Body.Close()

				var row visiblePORow
				if err := json.NewDecoder(res.Body).Decode(&row); err != nil {
					t.Fatalf("failed decoding visible PO response: %v", err)
				}
				if row.RemainingAmount == nil || row.ExpensesTotal == nil {
					t.Fatalf("expected remaining_amount and expenses_total in response, got %#v", row)
				}
				assertFloatApprox(t, *row.RemainingAmount, row.ApprovalTotal-*row.ExpensesTotal)
				assertFloatApprox(t, *row.RemainingAmount, 22)
			},
		},
		{
			Name:   "visible mine scope includes remaining_amount on returned rows",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/visible?scope=mine",
			Headers: map[string]string{
				"Authorization": creatorToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"remaining_amount":`,
			},
			TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
				defer res.Body.Close()

				var rows []visiblePORow
				if err := json.NewDecoder(res.Body).Decode(&rows); err != nil {
					t.Fatalf("failed decoding visible purchase order list: %v", err)
				}
				if len(rows) == 0 {
					t.Fatal("expected at least one purchase order in mine scope")
				}
				if rows[0].RemainingAmount == nil {
					t.Fatalf("expected remaining_amount on returned row, got %#v", rows[0])
				}
			},
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

func TestPurchaseOrdersSearchEndpoint(t *testing.T) {
	regularUserToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	noclaimsToken, err := testutils.GenerateRecordToken("users", "noclaims@example.com")
	if err != nil {
		t.Fatal(err)
	}

	type searchRow struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}

	assertHasID := func(t testing.TB, rows []searchRow, id string) {
		t.Helper()
		for _, row := range rows {
			if row.ID == id {
				return
			}
		}
		t.Fatalf("expected response to contain id %s, got %#v", id, rows)
	}

	assertLacksID := func(t testing.TB, rows []searchRow, id string) {
		t.Helper()
		for _, row := range rows {
			if row.ID == id {
				t.Fatalf("did not expect response to contain id %s, got %#v", id, rows)
			}
		}
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "regular user search only includes visible active statuses and excludes closed cancelled and unapproved records they cannot see",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/search",
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"2plsetqdxht7esg"`,
				`"status":"Active"`,
			},
			NotExpectedContent: []string{
				`"status":"Unapproved"`,
			},
			TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
				defer res.Body.Close()

				var rows []searchRow
				if err := json.NewDecoder(res.Body).Decode(&rows); err != nil {
					t.Fatalf("failed decoding search response: %v", err)
				}

				assertHasID(t, rows, "2plsetqdxht7esg")
				assertLacksID(t, rows, "0pia83nnprdlzf8")
				assertLacksID(t, rows, "1cqrvp4mna33k2b")
				assertLacksID(t, rows, "46efdq319b22480")
			},
		},
		{
			Name:   "creator search includes their visible closed and cancelled records while still excluding unapproved records",
			Method: http.MethodGet,
			URL:    "/api/purchase_orders/search",
			Headers: map[string]string{
				"Authorization": noclaimsToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"0pia83nnprdlzf8"`,
				`"status":"Closed"`,
				`"id":"1cqrvp4mna33k2b"`,
				`"status":"Cancelled"`,
			},
			NotExpectedContent: []string{
				`"status":"Unapproved"`,
			},
			TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
				defer res.Body.Close()

				var rows []searchRow
				if err := json.NewDecoder(res.Body).Decode(&rows); err != nil {
					t.Fatalf("failed decoding search response: %v", err)
				}

				assertHasID(t, rows, "0pia83nnprdlzf8")
				assertHasID(t, rows, "1cqrvp4mna33k2b")
				assertLacksID(t, rows, "46efdq319b22480")
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
