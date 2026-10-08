package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"
)

// Test fixtures:
//   - users: "rzr98oadsp9qc11" (time@test.com)
//   - users: "u_mileage_valid" (insurance expires 2025-12-31)
//   - users: "u_mileage_same_day" (insurance expires 2025-01-10)
//   - expense_rates effective 2025-01-05: Breakfast=20, Dinner=30,
//     mileage {"0": 0.70, "5000": 0.64}
//   - expenses: "exp_dup_attach_create_src_1" (receipt-backed Expense owned by
//     rzr98oadsp9qc11)
//   - purchase_orders: "exp_closed_po_1"

func TestExpensesCopyToTomorrow(t *testing.T) {
	ownerToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}
	mileageValidToken, err := testutils.GenerateRecordToken("users", "u_mileage_valid@example.com")
	if err != nil {
		t.Fatal(err)
	}
	mileageSameDayToken, err := testutils.GenerateRecordToken("users", "u_mileage_same_day@example.com")
	if err != nil {
		t.Fatal(err)
	}

	setup := func(t *testing.T) *tests.TestApp {
		app := testutils.SetupTestApp(t)
		t.Cleanup(app.Cleanup)
		return app
	}

	createExpense := func(t *testing.T, app *tests.TestApp, token string, body string) string {
		t.Helper()
		res := performTestAPIRequest(t, app, http.MethodPost, "/api/collections/expenses/records", strings.NewReader(body), map[string]string{
			"Authorization": token,
		})
		mustStatus(t, res, http.StatusOK)
		var created struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(res.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to decode create response: %v; body=%s", err, res.Body.String())
		}
		return created.ID
	}

	copyExpense := func(t *testing.T, app *tests.TestApp, token string, id string) (int, map[string]any) {
		t.Helper()
		res := performTestAPIRequest(t, app, http.MethodPost, "/api/expenses/"+id+"/copy_to_tomorrow", nil, map[string]string{
			"Authorization": token,
		})
		payload := map[string]any{}
		if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
			t.Fatalf("failed to decode copy response: %v; body=%s", err, res.Body.String())
		}
		return res.Code, payload
	}

	countExpenses := func(t *testing.T, app *tests.TestApp) int {
		t.Helper()
		var count int
		if err := app.DB().NewQuery("SELECT COUNT(*) FROM expenses").Row(&count); err != nil {
			t.Fatalf("failed to count expenses: %v", err)
		}
		return count
	}

	t.Run("allowance copy moves date, recomputes total, and starts as a fresh draft", func(t *testing.T) {
		app := setup(t)
		sourceID := createExpense(t, app, ownerToken, `{
			"uid": "rzr98oadsp9qc11",
			"date": "2025-01-10",
			"division": "vccd5fo56ctbigh",
			"payment_type": "Allowance",
			"allowance_types": ["Breakfast", "Dinner"],
			"total": 0
		}`)

		// Submit the source so the copy demonstrates that workflow state is not
		// carried over and that submitted sources remain eligible.
		submitRes := performTestAPIRequest(t, app, http.MethodPost, "/api/expenses/"+sourceID+"/submit", nil, map[string]string{
			"Authorization": ownerToken,
		})
		mustStatus(t, submitRes, http.StatusOK)

		status, payload := copyExpense(t, app, ownerToken, sourceID)
		if status != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %v", status, payload)
		}
		newID, _ := payload["new_record_id"].(string)
		if newID == "" || newID == sourceID {
			t.Fatalf("expected a new record id, got %q", newID)
		}

		copied, err := app.FindRecordById("expenses", newID)
		if err != nil {
			t.Fatalf("failed to load copied expense: %v", err)
		}
		if got := copied.GetString("date"); got != "2025-01-11" {
			t.Errorf("date = %q, want 2025-01-11", got)
		}
		if got := copied.GetFloat("total"); got != 50 {
			t.Errorf("total = %v, want 50", got)
		}
		if got := copied.GetString("description"); got != "Allowance for Breakfast, Dinner" {
			t.Errorf("description = %q", got)
		}
		if got := copied.GetStringSlice("allowance_types"); strings.Join(got, ",") != "Breakfast,Dinner" {
			t.Errorf("allowance_types = %v", got)
		}
		if copied.GetString("uid") != "rzr98oadsp9qc11" || copied.GetString("creator") != "rzr98oadsp9qc11" {
			t.Errorf("uid/creator = %q/%q, want owner", copied.GetString("uid"), copied.GetString("creator"))
		}
		if copied.GetBool("submitted") || copied.GetString("approved") != "" || copied.GetString("committed") != "" {
			t.Errorf("expected unsubmitted draft, got submitted=%v approved=%q committed=%q",
				copied.GetBool("submitted"), copied.GetString("approved"), copied.GetString("committed"))
		}
		if copied.GetString("approver") == "" {
			t.Error("expected approver to be assigned")
		}
		if copied.GetString("attachment_document") != "" {
			t.Error("expected no attachment on copy")
		}

		source, err := app.FindRecordById("expenses", sourceID)
		if err != nil {
			t.Fatalf("failed to reload source expense: %v", err)
		}
		if source.GetString("date") != "2025-01-10" || !source.GetBool("submitted") {
			t.Errorf("source changed: date=%q submitted=%v", source.GetString("date"), source.GetBool("submitted"))
		}
	})

	t.Run("mileage copy keeps distance and description and recomputes total", func(t *testing.T) {
		app := setup(t)
		sourceID := createExpense(t, app, mileageValidToken, `{
			"uid": "u_mileage_valid",
			"date": "2025-01-10",
			"division": "vccd5fo56ctbigh",
			"description": "client site visit",
			"payment_type": "Mileage",
			"distance": 100,
			"total": 0
		}`)

		status, payload := copyExpense(t, app, mileageValidToken, sourceID)
		if status != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %v", status, payload)
		}
		copied, err := app.FindRecordById("expenses", payload["new_record_id"].(string))
		if err != nil {
			t.Fatalf("failed to load copied expense: %v", err)
		}
		if got := copied.GetString("date"); got != "2025-01-11" {
			t.Errorf("date = %q, want 2025-01-11", got)
		}
		if got := copied.GetFloat("distance"); got != 100 {
			t.Errorf("distance = %v, want 100", got)
		}
		if got := copied.GetString("description"); got != "client site visit" {
			t.Errorf("description = %q", got)
		}
		if got := copied.GetFloat("total"); got != 70 {
			t.Errorf("total = %v, want 70", got)
		}
	})

	t.Run("copy crosses month and year boundary", func(t *testing.T) {
		app := setup(t)
		sourceID := createExpense(t, app, ownerToken, `{
			"uid": "rzr98oadsp9qc11",
			"date": "2025-12-31",
			"division": "vccd5fo56ctbigh",
			"payment_type": "Allowance",
			"allowance_types": ["Breakfast"],
			"total": 0
		}`)

		status, payload := copyExpense(t, app, ownerToken, sourceID)
		if status != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %v", status, payload)
		}
		copied, err := app.FindRecordById("expenses", payload["new_record_id"].(string))
		if err != nil {
			t.Fatalf("failed to load copied expense: %v", err)
		}
		if got := copied.GetString("date"); got != "2026-01-01" {
			t.Errorf("date = %q, want 2026-01-01", got)
		}
	})

	t.Run("mileage copy fails without creating a record when insurance expires before the new date", func(t *testing.T) {
		app := setup(t)
		sourceID := createExpense(t, app, mileageSameDayToken, `{
			"uid": "u_mileage_same_day",
			"date": "2025-01-10",
			"division": "vccd5fo56ctbigh",
			"description": "mileage on expiry day",
			"payment_type": "Mileage",
			"distance": 100,
			"total": 0
		}`)
		before := countExpenses(t, app)

		status, payload := copyExpense(t, app, mileageSameDayToken, sourceID)
		if status != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %v", status, payload)
		}
		body, _ := json.Marshal(payload)
		if !strings.Contains(string(body), `"insurance_expired"`) {
			t.Errorf("expected insurance_expired error, got %s", body)
		}
		if after := countExpenses(t, app); after != before {
			t.Errorf("expense count changed from %d to %d", before, after)
		}
	})

	t.Run("non-owner cannot copy", func(t *testing.T) {
		app := setup(t)
		sourceID := createExpense(t, app, ownerToken, `{
			"uid": "rzr98oadsp9qc11",
			"date": "2025-01-10",
			"division": "vccd5fo56ctbigh",
			"payment_type": "Allowance",
			"allowance_types": ["Breakfast"],
			"total": 0
		}`)
		before := countExpenses(t, app)

		status, payload := copyExpense(t, app, mileageValidToken, sourceID)
		if status != http.StatusForbidden || payload["code"] != "unauthorized" {
			t.Fatalf("expected 403 unauthorized, got %d: %v", status, payload)
		}
		if after := countExpenses(t, app); after != before {
			t.Errorf("expense count changed from %d to %d", before, after)
		}
	})

	t.Run("receipt-backed expense cannot be copied", func(t *testing.T) {
		app := setup(t)
		status, payload := copyExpense(t, app, ownerToken, "exp_dup_attach_create_src_1")
		if status != http.StatusBadRequest || payload["code"] != "unsupported_payment_type" {
			t.Fatalf("expected 400 unsupported_payment_type, got %d: %v", status, payload)
		}
	})

	t.Run("purchase order linked expense cannot be copied", func(t *testing.T) {
		app := setup(t)
		sourceID := createExpense(t, app, ownerToken, `{
			"uid": "rzr98oadsp9qc11",
			"date": "2025-01-10",
			"division": "vccd5fo56ctbigh",
			"payment_type": "Allowance",
			"allowance_types": ["Breakfast"],
			"total": 0
		}`)
		// Link the PO directly; the request hooks are not under test here.
		if _, err := app.DB().Update("expenses", dbx.Params{"purchase_order": "exp_closed_po_1"}, dbx.HashExp{"id": sourceID}).Execute(); err != nil {
			t.Fatalf("failed to link purchase order: %v", err)
		}
		before := countExpenses(t, app)

		status, payload := copyExpense(t, app, ownerToken, sourceID)
		if status != http.StatusBadRequest || payload["code"] != "linked_to_purchase_order" {
			t.Fatalf("expected 400 linked_to_purchase_order, got %d: %v", status, payload)
		}
		if after := countExpenses(t, app); after != before {
			t.Errorf("expense count changed from %d to %d", before, after)
		}
	})

	t.Run("missing expense returns 404", func(t *testing.T) {
		app := setup(t)
		status, payload := copyExpense(t, app, ownerToken, "doesnotexist000")
		if status != http.StatusNotFound || payload["code"] != "record_not_found" {
			t.Fatalf("expected 404 record_not_found, got %d: %v", status, payload)
		}
	})

	t.Run("copy blocked when expenses editing is disabled", func(t *testing.T) {
		app := setupExpensesEditingDisabledApp(t)
		t.Cleanup(app.Cleanup)
		status, payload := copyExpense(t, app, ownerToken, "exp_dup_attach_create_src_1")
		if status != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %v", status, payload)
		}
	})
}
