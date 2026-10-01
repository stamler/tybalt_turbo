package main

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestPurchaseOrderPrintBalance(t *testing.T) {
	token, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}
	snapshotPO := func(t testing.TB, app *tests.TestApp, id string) []byte {
		t.Helper()
		record, err := app.FindRecordById("purchase_orders", id)
		if err != nil {
			t.Fatal(err)
		}
		// Capture all stored fields, including approval state and timestamps.
		snapshot, err := json.Marshal(record.FieldsData())
		if err != nil {
			t.Fatal(err)
		}
		return snapshot
	}
	for _, tc := range []struct {
		name, id string
		want     float64
	}{
		{"cumulative deducts only committed expenses", "poprint00000001", 7000},
		{"exhausted cumulative has zero balance", "poprint00000002", 0},
		{"overdrawn cumulative retains negative balance", "poprint00000003", -10},
		{"foreign cumulative uses PO currency, not settled CAD", "poprint00000004", 100},
		{"unused cumulative retains approved amount", "poprint00000005", 100},
		{"one-time uses approved amount", "2plsetqdxht7esg", 132.10},
		{"recurring uses amount per period", "d8463q483f3da28", 144},
	} {
		var before []byte
		scenario := tests.ApiScenario{
			Name: tc.name, Method: http.MethodGet,
			URL:             "/api/purchase_orders/visible/" + tc.id,
			Headers:         map[string]string{"Authorization": token},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"print_max_amount":`},
			TestAppFactory:  testutils.SetupTestApp,
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				before = snapshotPO(t, app, tc.id)
			},
			AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
				defer res.Body.Close()
				var row struct {
					PrintMaxAmount  *float64 `json:"print_max_amount"`
					RemainingAmount float64  `json:"remaining_amount"`
				}
				if err := json.NewDecoder(res.Body).Decode(&row); err != nil {
					t.Fatal(err)
				}
				if row.PrintMaxAmount == nil || math.Abs(*row.PrintMaxAmount-tc.want) > 0.00001 {
					t.Fatalf("want print limit %.2f, got %v", tc.want, row.PrintMaxAmount)
				}
				if tc.id == "poprint00000001" && row.RemainingAmount != 6600 {
					t.Fatalf("provisional balance must still include uncommitted expenses, got %.2f", row.RemainingAmount)
				}
				if after := snapshotPO(t, app, tc.id); !bytes.Equal(before, after) {
					t.Fatalf("reading the print balance changed the PO\nbefore: %s\nafter: %s", before, after)
				}
			},
		}
		scenario.Test(t)
	}
	scenario := tests.ApiScenario{
		Name: "print balance requires authentication", Method: http.MethodGet,
		URL: "/api/purchase_orders/visible/poprint00000001", ExpectedStatus: http.StatusUnauthorized,
		ExpectedContent: []string{`"status":401`}, TestAppFactory: testutils.SetupTestApp,
	}
	scenario.Test(t)
}
