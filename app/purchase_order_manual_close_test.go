package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"
)

func TestPurchaseOrderManualClosePendingExpenses(t *testing.T) {
	closeToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}
	const approval = "This PO has one or more expenses awaiting approval by "
	const commitment = "This PO has one or more approved expenses awaiting commitment."
	const unknown = "an approver whose name is unavailable"
	for _, tc := range []struct {
		name        string
		poID        string
		message     string
		blankName   bool
		breakLookup bool
	}{
		{
			name: "approval names are sorted and each assigned approver appears once",
			poID: "closepending001", message: approval + "Fatty Maclean, Horace Silver.",
		},
		{
			name: "approved expenses still block without naming their approver",
			poID: "closepending002", message: commitment,
		},
		{
			name: "mixed states show both messages and only pending approval names",
			poID: "closepending003", message: approval + "Horace Silver. " + commitment,
		},
		{
			name: "missing profile does not remove a blocking expense",
			poID: "closepending004", message: approval + "Horace Silver, " + unknown + ".",
		},
		{
			name: "missing approver still blocks with a fallback message",
			poID: "closepending005", message: approval + unknown + ".",
		},
		{
			name: "blank profile name uses one fallback for repeated expenses",
			poID: "closepending001", message: approval + "Horace Silver, " + unknown + ".", blankName: true,
		},
		{
			name: "rejected expenses with and without approval and drafts do not block",
			poID: "closepending006",
		},
		{
			name: "committed expenses and pending expenses on other POs do not block",
			poID: "closepending007",
		},
		{
			name: "recurring PO blocks pending approval",
			poID: "closepending008", message: approval + "Horace Silver.",
		},
		{
			name: "legacy cumulative PO uses the same guard",
			poID: "closepending009", message: approval + "Horace Silver.",
		},
		{
			name: "approved foreign expense awaiting settlement still blocks",
			poID: "closepending010", message: commitment,
		},
		{
			name: "failed pending expense lookup prevents closure",
			poID: "closepending007", breakLookup: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := testutils.SetupTestApp(t)
			t.Cleanup(app.Cleanup)
			if tc.blankName {
				// Simulate invalid display data only in this test database. The
				// expense and PO rows remain the dedicated CSV fixtures.
				if _, err := app.DB().NewQuery(`UPDATE profiles SET given_name = ' ', surname = '' WHERE uid = 'etysnrlup2f6bak'`).Execute(); err != nil {
					t.Fatal(err)
				}
			}
			if tc.breakLookup {
				// Fault injection in the disposable database verifies that a
				// failed profile join cannot be treated as no pending expenses.
				if _, err := app.DB().NewQuery(`ALTER TABLE profiles RENAME TO unavailable_profiles`).Execute(); err != nil {
					t.Fatal(err)
				}
			}
			po, err := app.FindRecordById("purchase_orders", tc.poID)
			if err != nil {
				t.Fatal(err)
			}
			before, err := json.Marshal(po)
			if err != nil {
				t.Fatal(err)
			}
			response := performTestAPIRequest(t, app, http.MethodPost,
				"/api/purchase_orders/"+tc.poID+"/close", nil,
				map[string]string{"Authorization": closeToken})
			wantStatus := http.StatusOK
			wantCode := ""
			if tc.message != "" {
				wantStatus, wantCode = http.StatusBadRequest, "pending_expenses"
			}
			if tc.breakLookup {
				wantStatus, wantCode = http.StatusInternalServerError, "error_fetching_expenses"
			}
			if response.Code != wantStatus {
				t.Fatalf("status=%d, want %d; body=%s", response.Code, wantStatus, response.Body.String())
			}
			po, err = app.FindRecordById("purchase_orders", tc.poID)
			if err != nil {
				t.Fatal(err)
			}
			if wantStatus == http.StatusOK {
				if po.GetString("status") != "Closed" || po.GetDateTime("closed").IsZero() ||
					po.GetString("closer") != "tqqf7q0f3378rvp" || po.GetBool("closed_by_system") {
					t.Fatalf("expected a manual closure; got %v", po.PublicExport())
				}
				return
			}
			var body struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Code != wantCode || (tc.message != "" && body.Message != tc.message) {
				t.Fatalf("unexpected closure error: %+v; want code=%s, message=%q", body, wantCode, tc.message)
			}
			after, err := json.Marshal(po)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Fatalf("blocked closure changed the PO\nbefore: %s\nafter: %s", before, after)
			}
		})
	}
}

func TestPurchaseOrderManualCloseAfterPendingExpenseProcessed(t *testing.T) {
	// The recurring fixture has room for more committed expenses. Committing this
	// expense must clear the manual block without triggering automatic closure.
	for _, action := range []string{"commit", "recall", "reject"} {
		t.Run(action, func(t *testing.T) {
			app := testutils.SetupTestApp(t)
			t.Cleanup(app.Cleanup)
			tokens := make(map[string]string)
			for _, email := range []string{"book@keeper.com", "author@soup.com", "fakemanager@fakesite.xyz"} {
				token, err := testutils.GenerateRecordToken("users", email)
				if err != nil {
					t.Fatal(err)
				}
				tokens[email] = token
			}
			request := func(path, email string, wantStatus int, wantMessage string) {
				t.Helper()
				response := performTestAPIRequest(t, app, http.MethodPost, path, nil,
					map[string]string{"Authorization": tokens[email]})
				if response.Code != wantStatus {
					t.Fatalf("%s: status=%d, want %d; body=%s", path, response.Code, wantStatus, response.Body.String())
				}
				if wantMessage != "" {
					var body struct{ Message string }
					if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Message != wantMessage {
						t.Fatalf("%s: expected %q; body=%s, error=%v", path, wantMessage, response.Body.String(), err)
					}
				}
			}
			const closePath = "/api/purchase_orders/closepending008/close"
			const expensePath = "/api/expenses/closeexp008pend"
			request(closePath, "book@keeper.com", http.StatusBadRequest,
				"This PO has one or more expenses awaiting approval by Horace Silver.")
			switch action {
			case "commit":
				request(expensePath+"/approve", "author@soup.com", http.StatusOK, "")
				request(closePath, "book@keeper.com", http.StatusBadRequest,
					"This PO has one or more approved expenses awaiting commitment.")
				request(expensePath+"/commit", "fakemanager@fakesite.xyz", http.StatusOK, "")
			case "recall":
				request(expensePath+"/recall", "author@soup.com", http.StatusOK, "")
			case "reject":
				response := performTestAPIRequest(t, app, http.MethodPost, expensePath+"/reject",
					strings.NewReader(`{"rejection_reason":"Receipt requires correction"}`),
					map[string]string{"Authorization": tokens["author@soup.com"], "Content-Type": "application/json"})
				if response.Code != http.StatusOK {
					t.Fatalf("reject: status=%d; body=%s", response.Code, response.Body.String())
				}
			}
			request(closePath, "book@keeper.com", http.StatusOK, "")
			po, err := app.FindRecordById("purchase_orders", "closepending008")
			if err != nil {
				t.Fatal(err)
			}
			if po.GetString("status") != "Closed" || po.GetBool("closed_by_system") {
				t.Fatalf("expected manual closure after %s; got %v", action, po.PublicExport())
			}
		})
	}
}
