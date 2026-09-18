package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
)

func TestRecurringPurchaseOrderClosesOnFinalCommit(t *testing.T) {
	for _, tc := range []struct {
		frequency string
		startDate string
		endDate   string
		limit     int
	}{
		{"Monthly", "2025-01-01", "2025-12-31", 12},
		{"Weekly", "2025-02-17", "2025-03-03", 2},
		{"Biweekly", "2025-02-17", "2025-03-31", 3},
	} {
		t.Run(tc.frequency, func(t *testing.T) {
			app := testutils.SetupTestApp(t)
			t.Cleanup(app.Cleanup)

			commitToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
			if err != nil {
				t.Fatal(err)
			}
			po, err := app.FindRecordById("purchase_orders", "recurclosepo001")
			if err != nil {
				t.Fatal(err)
			}
			// Change the fixture schedule and approval values to test each frequency
			// with a different count. All expense rows come from the CSV fixture.
			po.Set("frequency", tc.frequency)
			po.Set("date", tc.startDate)
			po.Set("end_date", tc.endDate)
			po.Set("approval_total", tc.limit*150)
			po.Set("approval_total_home", tc.limit*150)
			if err := app.Save(po); err != nil {
				t.Fatal(err)
			}
			occurrences, _, err := utilities.CalculateRecurringPurchaseOrderTotalValue(app, po)
			if err != nil || occurrences != tc.limit {
				t.Fatalf("fixture must permit %d expenses: count=%d, error=%v", tc.limit, occurrences, err)
			}

			assertState := func(wantCount int, wantStatus string) {
				t.Helper()
				po, err := app.FindRecordById("purchase_orders", "recurclosepo001")
				if err != nil {
					t.Fatal(err)
				}
				var result struct {
					Count int `db:"count"`
				}
				if err := app.DB().NewQuery(`SELECT COUNT(*) AS count FROM expenses
			WHERE purchase_order = 'recurclosepo001' AND committed != ''`).One(&result); err != nil {
					t.Fatal(err)
				}
				if result.Count != wantCount || po.GetString("status") != wantStatus {
					t.Fatalf("committed count=%d, PO status=%s; want %d, %s", result.Count, po.GetString("status"), wantCount, wantStatus)
				}
			}
			commit := func(index int) (int, string) {
				t.Helper()
				response := performTestAPIRequest(t, app, http.MethodPost,
					fmt.Sprintf("/api/expenses/recurcloseexp%02d/commit", index), nil,
					map[string]string{"Authorization": commitToken})
				return response.Code, response.Body.String()
			}

			// Each expense is $100 against a $150 per-expense limit. Closure must use
			// the entry count even though the combined total is below the PO value.
			for i := 1; i < tc.limit; i++ {
				if status, body := commit(i); status != http.StatusOK {
					t.Fatalf("commit %d: status=%d, body=%s", i, status, body)
				}
				assertState(i, "Active")
			}

			// Fail the final expense save after the PO has been closed in the same
			// transaction. Neither change must remain after the failed request.
			finalID := fmt.Sprintf("recurcloseexp%02d", tc.limit)
			saveFailureReached := false
			hookID := app.OnRecordUpdate("expenses").BindFunc(func(e *core.RecordEvent) error {
				if e.Record.Id != finalID || e.Record.GetDateTime("committed").IsZero() {
					return e.Next()
				}
				savedPO, err := e.App.FindRecordById("purchase_orders", po.Id)
				if err != nil {
					return err
				}
				if savedPO.GetString("status") != "Closed" {
					return errors.New("PO was not closed before the final expense save")
				}
				saveFailureReached = true
				return errors.New("test final expense save failure")
			})
			if status, body := commit(tc.limit); status != http.StatusInternalServerError || !strings.Contains(body, "error_saving_record") {
				t.Fatalf("failed final save: status=%d, body=%s", status, body)
			}
			app.OnRecordUpdate("expenses").Unbind(hookID)
			if !saveFailureReached {
				t.Fatal("final save failure was not reached after PO closure")
			}
			assertState(tc.limit-1, "Active")
			failedExpense, err := app.FindRecordById("expenses", finalID)
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{"committed", "committer", "committed_week_ending", "pay_period_ending"} {
				if got := failedExpense.GetString(field); got != "" {
					t.Fatalf("failed commit left %s=%q", field, got)
				}
			}
			if status, body := commit(tc.limit); status != http.StatusOK {
				t.Fatalf("final commit: status=%d, body=%s", status, body)
			}
			assertState(tc.limit, "Closed")

			if status, body := commit(tc.limit + 1); status != http.StatusBadRequest || !strings.Contains(body, "purchase_order_not_active") {
				t.Fatalf("extra commit: status=%d, body=%s", status, body)
			}
			assertState(tc.limit, "Closed")
			if status, body := commit(tc.limit); status != http.StatusBadRequest || !strings.Contains(body, "record_already_committed") {
				t.Fatalf("duplicate commit: status=%d, body=%s", status, body)
			}
			assertState(tc.limit, "Closed")

			adminToken, err := testutils.GenerateRecordToken("users", "admin.only@example.com")
			if err != nil {
				t.Fatal(err)
			}
			uncommit := func(index int) {
				t.Helper()
				response := performTestAPIRequest(t, app, http.MethodPost,
					fmt.Sprintf("/api/expenses/recurcloseexp%02d/uncommit", index), nil,
					map[string]string{"Authorization": adminToken})
				mustStatus(t, response, http.StatusOK)
			}

			// The old closure rule could leave one extra committed expense. Modify
			// an existing fixture row to reproduce that saved state. Removing the
			// extra commit must keep the PO closed while the count equals the limit.
			extraExpense, err := app.FindRecordById("expenses", fmt.Sprintf("recurcloseexp%02d", tc.limit+1))
			if err != nil {
				t.Fatal(err)
			}
			finalExpense, err := app.FindRecordById("expenses", finalID)
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{"committed", "committer", "committed_week_ending", "pay_period_ending"} {
				extraExpense.Set(field, finalExpense.Get(field))
			}
			if err := app.Save(extraExpense); err != nil {
				t.Fatal(err)
			}
			assertState(tc.limit+1, "Closed")
			uncommit(tc.limit + 1)
			assertState(tc.limit, "Closed")
			uncommit(tc.limit)
			assertState(tc.limit-1, "Active")
			if status, body := commit(tc.limit); status != http.StatusOK {
				t.Fatalf("recommit final expense: status=%d, body=%s", status, body)
			}
			assertState(tc.limit, "Closed")
		})
	}
}

func TestOtherPurchaseOrderTypesKeepCommitClosureRules(t *testing.T) {
	for _, tc := range []struct {
		poType     string
		total      int
		finalEntry int
	}{
		{"One-Time", 150, 1},
		{"Cumulative", 200, 2},
	} {
		t.Run(tc.poType, func(t *testing.T) {
			app := testutils.SetupTestApp(t)
			t.Cleanup(app.Cleanup)
			token, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
			if err != nil {
				t.Fatal(err)
			}
			po, err := app.FindRecordById("purchase_orders", "recurclosepo001")
			if err != nil {
				t.Fatal(err)
			}
			// Reuse the CSV fixture with a different PO type and its required
			// values. Each linked expense remains $100. A one-time PO must close
			// on its first commit; a cumulative PO must close at its total value.
			po.Set("type", tc.poType)
			po.Set("frequency", "")
			po.Set("end_date", "")
			po.Set("total", tc.total)
			po.Set("approval_total", tc.total)
			po.Set("approval_total_home", tc.total)
			if err := app.Save(po); err != nil {
				t.Fatal(err)
			}
			for i := 1; i <= tc.finalEntry+1; i++ {
				response := performTestAPIRequest(t, app, http.MethodPost,
					fmt.Sprintf("/api/expenses/recurcloseexp%02d/commit", i), nil,
					map[string]string{"Authorization": token})
				if i <= tc.finalEntry {
					mustStatus(t, response, http.StatusOK)
				} else {
					mustStatus(t, response, http.StatusBadRequest)
					if !strings.Contains(response.Body.String(), "purchase_order_not_active") {
						t.Fatalf("extra commit: %s", response.Body.String())
					}
				}
				po, err := app.FindRecordById("purchase_orders", po.Id)
				if err != nil {
					t.Fatal(err)
				}
				wantStatus := "Active"
				if i >= tc.finalEntry {
					wantStatus = "Closed"
				}
				if got := po.GetString("status"); got != wantStatus {
					t.Fatalf("commit %d: PO status=%s, want %s", i, got, wantStatus)
				}
			}
		})
	}
}
