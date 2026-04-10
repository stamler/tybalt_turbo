package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func setupExpensesExportMissingLegacyUIDApp(tb testing.TB) *tests.TestApp {
	app := testutils.SetupTestApp(tb)

	// This helper relies on the seeded user_claims row `uc_missing_legacy_uid`.
	// The export query joins po_approver_props -> user_claims before checking for
	// a missing legacy UID, so omitting that seeded user_claim would cause the
	// inserted po_approver_props row to be skipped instead of triggering the
	// intended error path.
	if _, err := app.DB().NewQuery(`
		INSERT INTO po_approver_props (
			id,
			user_claim,
			max_amount,
			project_max,
			sponsorship_max,
			staff_and_social_max,
			media_and_event_max,
			computer_max,
			divisions,
			created,
			updated
		) VALUES (
			'pap_missing_legacy_uid',
			'uc_missing_legacy_uid',
			12000,
			12000,
			12000,
			12000,
			12000,
			12000,
			'[]',
			'2025-01-01 00:00:00.000Z',
			'2025-01-01 00:00:00.000Z'
		)
	`).Execute(); err != nil {
		tb.Fatal(err)
	}

	return app
}

func TestExpensesExportLegacyIncludesPoApproverProps(t *testing.T) {
	validToken := "test-secret-123"

	scenarios := []tests.ApiScenario{
		{
			Name:   "valid machine token returns structured response including poApproverProps",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/expenses/2000-01-01",
			Headers: map[string]string{
				"Authorization": "Bearer " + validToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"expenses":[`,
				`"vendors":[`,
				`"purchaseOrders":[`,
				`"poApproverProps":[`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "fails fast when a po_approver_props row has no legacy uid mapping",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/expenses/2000-01-01",
			Headers: map[string]string{
				"Authorization": "Bearer " + validToken,
			},
			ExpectedStatus: http.StatusInternalServerError,
			ExpectedContent: []string{
				`legacy uid for po_approver_props id pap_missing_legacy_uid`,
			},
			TestAppFactory: setupExpensesExportMissingLegacyUIDApp,
		},
		{
			Name:   "uses fixed fallback timestamps when po_approver_props timestamps are blank",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/expenses/2000-01-01",
			Headers: map[string]string{
				"Authorization": "Bearer " + validToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"pap_missing_timestamps"`,
				`"created":"1970-01-01 00:00:00.000Z"`,
				`"updated":"1970-01-01 00:00:00.000Z"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "exports unimported purchase orders even when updatedAfter is in the future and no expenses reference them",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/expenses/2099-01-01",
			Headers: map[string]string{
				"Authorization": "Bearer " + validToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"po_export_no_exp_1"`,
				`"poNumber":"2504-5004"`,
				`"vendorId":"yxhycv2ycpvsbt4"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "does not export unapproved purchase orders with blank po numbers",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/expenses/2000-01-01",
			Headers: map[string]string{
				"Authorization": "Bearer " + validToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"purchaseOrders":[`,
			},
			NotExpectedContent: []string{
				`"id":"po_export_blank_unapproved"`,
				`"description":"Draft PO should not be exported"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "includes currency code and settled_total only when expense currency is explicitly set",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/expenses/2000-01-01",
			Headers: map[string]string{
				"Authorization": "Bearer " + validToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"currency":"USD"`,
				`"settled_total":91.11`,
			},
			TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
				defer res.Body.Close()

				var payload struct {
					Expenses []struct {
						ImmutableID  string   `json:"immutableID"`
						Currency     string   `json:"currency,omitempty"`
						SettledTotal *float64 `json:"settled_total,omitempty"`
					} `json:"expenses"`
				}
				if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
					tb.Fatalf("failed to decode expenses export response: %v", err)
				}

				var withCurrency *struct {
					ImmutableID  string   `json:"immutableID"`
					Currency     string   `json:"currency,omitempty"`
					SettledTotal *float64 `json:"settled_total,omitempty"`
				}
				var blankCurrency *struct {
					ImmutableID  string   `json:"immutableID"`
					Currency     string   `json:"currency,omitempty"`
					SettledTotal *float64 `json:"settled_total,omitempty"`
				}

				for i := range payload.Expenses {
					switch payload.Expenses[i].ImmutableID {
					case "rptcurusd000001":
						withCurrency = &payload.Expenses[i]
					case "rptcurcad000001":
						blankCurrency = &payload.Expenses[i]
					}
				}

				if withCurrency == nil {
					tb.Fatal("expected exported expense with explicit currency")
				}
				if withCurrency.Currency != "USD" {
					tb.Fatalf("expected explicit currency code USD, got %q", withCurrency.Currency)
				}
				if withCurrency.SettledTotal == nil || *withCurrency.SettledTotal != 91.11 {
					tb.Fatalf("expected settled_total 91.11 when currency is set, got %+v", withCurrency.SettledTotal)
				}

				if blankCurrency == nil {
					tb.Fatal("expected exported expense with blank legacy currency")
				}
				if blankCurrency.Currency != "" {
					tb.Fatalf("expected blank legacy currency to omit currency code, got %q", blankCurrency.Currency)
				}
				if blankCurrency.SettledTotal != nil {
					tb.Fatalf("expected blank legacy currency to omit settled_total, got %+v", *blankCurrency.SettledTotal)
				}
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
