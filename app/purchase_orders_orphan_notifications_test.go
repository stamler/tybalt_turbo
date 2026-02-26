package main

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func countNotificationsForPO(tb testing.TB, app *tests.TestApp, poID string) int {
	tb.Helper()

	var row struct {
		Count int `db:"count"`
	}
	if err := app.DB().NewQuery(`
		SELECT COUNT(*) AS count
		FROM notifications
		WHERE json_extract(data, '$.POId') = {:poID}
	`).Bind(dbx.Params{"poID": poID}).One(&row); err != nil {
		tb.Fatalf("failed counting notifications for PO %s: %v", poID, err)
	}

	return row.Count
}

func countPurchaseOrdersByID(tb testing.TB, app *tests.TestApp, poID string) int {
	tb.Helper()

	var row struct {
		Count int `db:"count"`
	}
	if err := app.DB().NewQuery(`
		SELECT COUNT(*) AS count
		FROM purchase_orders
		WHERE id = {:id}
	`).Bind(dbx.Params{"id": poID}).One(&row); err != nil {
		tb.Fatalf("failed counting purchase_orders record %s: %v", poID, err)
	}

	return row.Count
}

func TestPurchaseOrderRequestHooks_DeleteOrphanedNotificationOnNextFailure(t *testing.T) {
	createToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	updateToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	kindLookupApp := testutils.SetupTestApp(t)
	capitalKind, err := kindLookupApp.FindFirstRecordByFilter("expenditure_kinds", "name = 'capital'")
	if err != nil {
		kindLookupApp.Cleanup()
		t.Fatalf("failed to load capital expenditure kind: %v", err)
	}
	capitalKindID := capitalKind.Id
	kindLookupApp.Cleanup()

	const updatePOID = "2blv18f40i2q373"

	var createdPOID string
	createScenario := tests.ApiScenario{
		Name:   "PO create request cleans up notification when downstream hook returns error",
		Method: http.MethodPost,
		URL:    "/api/collections/purchase_orders/records",
		Body: bytes.NewBufferString(fmt.Sprintf(`{
			"uid": "rzr98oadsp9qc11",
			"date": "2024-09-01",
			"division": "vccd5fo56ctbigh",
			"description": "forced create e.Next failure cleanup",
			"payment_type": "Expense",
			"total": 123.45,
			"vendor": "2zqxtsmymf670ha",
			"approver": "etysnrlup2f6bak",
			"priority_second_approver": "6bq4j0eb26631dy",
			"status": "Unapproved",
			"type": "One-Time",
			"kind": %q
		}`, capitalKindID)),
		Headers: map[string]string{
			"Authorization": createToken,
			"Content-Type":  "application/json",
		},
		ExpectedStatus: 400,
		ExpectedContent: []string{
			`"message":"Forced create request failure."`,
		},
		AfterTestFunc: func(tb testing.TB, app *tests.TestApp, res *http.Response) {
			if createdPOID == "" {
				tb.Fatal("expected failing downstream create hook to capture generated PO id")
			}

			if notificationsCount := countNotificationsForPO(tb, app, createdPOID); notificationsCount != 0 {
				tb.Fatalf("expected no orphan notifications for failed create request, got %d", notificationsCount)
			}

			if poCount := countPurchaseOrdersByID(tb, app, createdPOID); poCount != 0 {
				tb.Fatalf("expected failed create request to leave no purchase_orders row, got %d", poCount)
			}
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			createdPOID = ""
			app := testutils.SetupTestApp(tb)
			app.OnRecordCreateRequest("purchase_orders").BindFunc(func(e *core.RecordRequestEvent) error {
				createdPOID = e.Record.Id
				return apis.NewBadRequestError("forced create request failure", nil)
			})
			return app
		},
	}

	var updateBeforeNotificationCount int
	var updateBeforeDescription string
	var updateHookRecordID string
	updateScenario := tests.ApiScenario{
		Name:   "PO update request cleans up notification when downstream hook returns error",
		Method: http.MethodPatch,
		URL:    "/api/collections/purchase_orders/records/" + updatePOID,
		Body: bytes.NewBufferString(`{
			"description": "forced update e.Next failure cleanup"
		}`),
		Headers: map[string]string{
			"Authorization": updateToken,
			"Content-Type":  "application/json",
		},
		ExpectedStatus: 400,
		ExpectedContent: []string{
			`"message":"Forced update request failure."`,
		},
		BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			updateBeforeNotificationCount = countNotificationsForPO(tb, app, updatePOID)

			po, err := app.FindRecordById("purchase_orders", updatePOID)
			if err != nil {
				tb.Fatalf("failed loading purchase order fixture %s: %v", updatePOID, err)
			}
			updateBeforeDescription = po.GetString("description")
		},
		AfterTestFunc: func(tb testing.TB, app *tests.TestApp, res *http.Response) {
			if updateHookRecordID != updatePOID {
				tb.Fatalf("expected failing downstream update hook to run for %s, got %s", updatePOID, updateHookRecordID)
			}

			updateAfterNotificationCount := countNotificationsForPO(tb, app, updatePOID)
			if updateAfterNotificationCount != updateBeforeNotificationCount {
				tb.Fatalf("expected notification count to remain unchanged after failed update request, before=%d after=%d", updateBeforeNotificationCount, updateAfterNotificationCount)
			}

			po, err := app.FindRecordById("purchase_orders", updatePOID)
			if err != nil {
				tb.Fatalf("failed loading purchase order fixture %s after failed update: %v", updatePOID, err)
			}
			if po.GetString("description") != updateBeforeDescription {
				tb.Fatalf("expected purchase order description to remain %q after failed update, got %q", updateBeforeDescription, po.GetString("description"))
			}
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			updateHookRecordID = ""
			app := testutils.SetupTestApp(tb)
			app.OnRecordUpdateRequest("purchase_orders").BindFunc(func(e *core.RecordRequestEvent) error {
				updateHookRecordID = e.Record.Id
				return apis.NewBadRequestError("forced update request failure", nil)
			})
			return app
		},
	}

	createScenario.Test(t)
	updateScenario.Test(t)
}
