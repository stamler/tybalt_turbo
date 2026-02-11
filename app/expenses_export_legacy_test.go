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
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				tb.Helper()

				var claimID string
				if err := app.DB().NewQuery(`SELECT id FROM claims WHERE name = 'po_approver' LIMIT 1`).Row(&claimID); err != nil {
					tb.Fatalf("failed to find po_approver claim id: %v", err)
				}

				var uid string
				if err := app.DB().NewQuery(`SELECT uid FROM admin_profiles WHERE COALESCE(legacy_uid, '') != '' LIMIT 1`).Row(&uid); err != nil {
					tb.Fatalf("failed to find admin_profile with legacy uid: %v", err)
				}

				if _, err := app.DB().NewQuery(`
					INSERT OR IGNORE INTO user_claims (uid, cid, created, updated)
					VALUES ({:uid}, {:cid}, datetime('now'), datetime('now'))
				`).Bind(dbx.Params{
					"uid": uid,
					"cid": claimID,
				}).Execute(); err != nil {
					tb.Fatalf("failed to ensure user_claim fixture: %v", err)
				}

				var userClaimID string
				if err := app.DB().NewQuery(`
					SELECT id FROM user_claims
					WHERE uid = {:uid} AND cid = {:cid}
					LIMIT 1
				`).Bind(dbx.Params{
					"uid": uid,
					"cid": claimID,
				}).Row(&userClaimID); err != nil {
					tb.Fatalf("failed to resolve user_claim id: %v", err)
				}

				if _, err := app.DB().NewQuery(`
					INSERT INTO po_approver_props (
						id, user_claim, max_amount, project_max, sponsorship_max,
						staff_and_social_max, media_and_event_max, computer_max,
						divisions, created, updated
					) VALUES (
						{:id}, {:user_claim}, 500, 500, 0,
						0, 0, 0,
						'[]', '', ''
					)
				`).Bind(dbx.Params{
					"id":         "pap_missing_timestamps",
					"user_claim": userClaimID,
				}).Execute(); err != nil {
					tb.Fatalf("failed to insert po_approver_props blank timestamp fixture: %v", err)
				}
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
