package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
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
				return r
			}(),
			setup: func(t *testing.T, app *tests.TestApp) {
				// Create and save the first child PO
				firstChild := core.NewRecord(poCollection)
				firstChild.Set("parent_po", "2plsetqdxht7esg")
				firstChild.Set("po_number", "2024-0008-01")
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
				return r
			}(),
			setup: func(t *testing.T, app *tests.TestApp) {
				// Create 99 child POs
				for i := 1; i <= 99; i++ {
					child := core.NewRecord(poCollection)
					child.Set("parent_po", "2plsetqdxht7esg")
					child.Set("po_number", fmt.Sprintf("2024-0008-%02d", i))
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
