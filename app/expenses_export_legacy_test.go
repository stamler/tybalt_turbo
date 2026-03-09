package main

import (
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				tb.Helper()

				var claimID string
				if err := app.DB().NewQuery(`SELECT id FROM claims WHERE name = 'po_approver' LIMIT 1`).Row(&claimID); err != nil {
					tb.Fatalf("failed to find po_approver claim id: %v", err)
				}

				if _, err := app.DB().NewQuery(`
					INSERT INTO user_claims (id, uid, cid, created, updated)
					VALUES ({:id}, {:uid}, {:cid}, datetime('now'), datetime('now'))
				`).Bind(dbx.Params{
					"id":  "uc_missing_legacy_uid",
					"uid": "uid_missing_legacy_profile",
					"cid": claimID,
				}).Execute(); err != nil {
					tb.Fatalf("failed to insert user_claim fixture: %v", err)
				}

				if _, err := app.DB().NewQuery(`
					INSERT INTO po_approver_props (
						id, user_claim, max_amount, project_max, sponsorship_max,
						staff_and_social_max, media_and_event_max, computer_max,
						divisions, created, updated
					) VALUES (
						{:id}, {:user_claim}, 500, 500, 0,
						0, 0, 0,
						'[]', datetime('now'), datetime('now')
					)
				`).Bind(dbx.Params{
					"id":         "pap_missing_legacy_uid",
					"user_claim": "uc_missing_legacy_uid",
				}).Execute(); err != nil {
					tb.Fatalf("failed to insert po_approver_props fixture: %v", err)
				}
			},
			TestAppFactory: testutils.SetupTestApp,
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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				tb.Helper()

				if _, err := app.DB().NewQuery(`
					INSERT INTO purchase_orders (
						id, uid, date, division, description, payment_type, total, approval_total,
						vendor, status, type, po_number, _imported, branch, kind, legacy_manual_entry,
						created, updated
					) VALUES (
						'po_export_no_exp_1', 'f2j5a8vk006baub', '2025-04-01', 'vccd5fo56ctbigh',
						'PO exported without expenses', 'OnAccount', 150.00, 150.00, 'yxhycv2ycpvsbt4',
						'Active', 'One-Time', '2504-5004', 0, '80875lm27v8wgi4', 'l3vtlbqg529m52j', 1,
						'2020-01-01 00:00:00.000Z', '2020-01-01 00:00:00.000Z'
					)
				`).Execute(); err != nil {
					tb.Fatalf("failed to insert standalone exported PO fixture: %v", err)
				}
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
