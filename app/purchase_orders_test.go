package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestPurchaseOrdersCreate(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}
	scenarios := []tests.ApiScenario{
		{
			Name:   "valid purchase order is created",
			Method: http.MethodPost,
			Url:    "/api/collections/purchase_orders/records",
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
			RequestHeaders: map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"approver":"etysnrlup2f6bak"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeCreate":         1,
				"OnModelAfterCreate":          1,
				"OnRecordBeforeCreateRequest": 1,
				"OnRecordAfterCreateRequest":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "otherwise valid purchase order with Inactive vendor fails",
			Method: http.MethodPost,
			Url:    "/api/collections/purchase_orders/records",
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
			RequestHeaders: map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`{"code":400,"message":"Failed to create record."`,
			},
			ExpectedEvents: map[string]int{},
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
			Url:    "/api/collections/purchase_orders/records/gal6e5la2fa4rpn",
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
			RequestHeaders: map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"approver":"etysnrlup2f6bak"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate":         1,
				"OnModelAfterUpdate":          1,
				"OnRecordBeforeUpdateRequest": 1,
				"OnRecordAfterUpdateRequest":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "otherwise valid purchase order with Inactive vendor fails",
			Method: http.MethodPatch,
			Url:    "/api/collections/purchase_orders/records/gal6e5la2fa4rpn",
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
			RequestHeaders: map[string]string{"Authorization": recordToken},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"code":404,"message":"The requested resource wasn't found."`,
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
		{
			Name:            "caller with the payables_admin claim can cancel Active purchase_orders records with no expenses against them",
			Method:          http.MethodPost,
			Url:             "/api/purchase_orders/2plsetqdxht7esg/cancel",
			RequestHeaders:  map[string]string{"Authorization": closeToken},
			ExpectedStatus:  204,
			ExpectedContent: []string{},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate": 1,
				"OnModelAfterUpdate":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "caller without the payables_admin claim cannot cancel Active purchase_orders records with no expenses against them",
			Method:         http.MethodPost,
			Url:            "/api/purchase_orders/2plsetqdxht7esg/cancel",
			RequestHeaders: map[string]string{"Authorization": nonCloseToken},
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
