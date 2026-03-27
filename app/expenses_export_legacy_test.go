package main

import (
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

func setupExpensesExportBlankUnapprovedPOApp(tb testing.TB) *tests.TestApp {
	app := testutils.SetupTestApp(tb)

	if _, err := app.DB().NewQuery(`
		INSERT INTO purchase_orders (
			id,
			po_number,
			status,
			_imported,
			date,
			type,
			payment_type,
			total,
			description
		) VALUES (
			'po_export_blank_unapproved',
			'',
			'Unapproved',
			0,
			'2026-03-01',
			'One-Time',
			'CC',
			123.45,
			'Draft PO should not be exported'
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
			TestAppFactory: setupExpensesExportBlankUnapprovedPOApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
