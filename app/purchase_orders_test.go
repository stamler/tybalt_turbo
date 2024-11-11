package main

import (
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestPurchaseOrdersCreate(t *testing.T) {
	/*
		recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
		if err != nil {
			t.Fatal(err)
		}
	*/
	scenarios := []tests.ApiScenario{
		// TODO: Add test scenarios for purchase order creation
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestPurchaseOrdersUpdate(t *testing.T) {
	/*
		recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
		if err != nil {
			t.Fatal(err)
		}
	*/

	scenarios := []tests.ApiScenario{
		// TODO: Add test scenarios for purchase order updates
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
	nonCloseToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	closeToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}

	/*
		creatorToken, err := testutils.GenerateRecordToken("users", "time@test.com")
		if err != nil {
			t.Fatal(err)
		}

		approverToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
		if err != nil {
			t.Fatal(err)
		}
	*/

	scenarios := []tests.ApiScenario{
		// TODO: Add test scenarios for custom routes:
		// - /api/purchase_orders/{id}/approve
		// - /api/purchase_orders/{id}/reject
		// - /api/purchase_orders/{id}/cancel
		{
			Name:            "caller without the payables_admin claim cannot close Active Cumulative purchase_orders records",
			Method:          http.MethodPost,
			Url:             "/api/purchase_orders/ly8xyzpuj79upq1/close",
			RequestHeaders:  map[string]string{"Authorization": nonCloseToken},
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
			Url:             "/api/purchase_orders/ly8xyzpuj79upq1/close",
			RequestHeaders:  map[string]string{"Authorization": closeToken},
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
			Url:             "/api/purchase_orders/2plsetqdxht7esg/close",
			RequestHeaders:  map[string]string{"Authorization": closeToken},
			ExpectedStatus:  400,
			ExpectedContent: []string{`"code":"invalid_po_type","message":"only cumulative purchase orders can be closed manually"`},
			ExpectedEvents: map[string]int{
				"OnBeforeApiError": 0,
				"OnAfterApiError":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		// TODO: Non-Active Cumulative purchase_orders records cannot be closed
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
