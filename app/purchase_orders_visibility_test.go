package main

import (
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestPurchaseOrdersVisibilityRules(t *testing.T) {
	// Generate token for a regular user
	regularUserToken, err := testutils.GenerateRecordToken("users", "time@test.com")
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
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
