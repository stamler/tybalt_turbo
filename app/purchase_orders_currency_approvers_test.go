package main

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
)

func TestPurchaseOrderApproversRoutes_UseHomeCurrencyConversion(t *testing.T) {
	regularUserToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	tier1, _ := testutils.GetApprovalTiers(app)
	capitalKind, err := app.FindFirstRecordByFilter("expenditure_kinds", "name = {:name}", dbx.Params{
		"name": "capital",
	})
	if err != nil {
		t.Fatalf("failed loading capital expenditure kind: %v", err)
	}

	makeURL := func(amount float64, currencyID string) string {
		params := url.Values{}
		params.Set("division", "2rrfy6m2c8hazjy")
		params.Set("amount", fmt.Sprintf("%.2f", amount))
		params.Set("kind", capitalKind.Id)
		params.Set("has_job", "false")
		if currencyID != "" {
			params.Set("currency", currencyID)
		}
		return "/api/purchase_orders/second_approvers?" + params.Encode()
	}

	amountBelowTier1 := tier1 - 10
	if amountBelowTier1 <= 0 {
		t.Fatalf("expected positive amount below tier1, got %v", amountBelowTier1)
	}

	cadRes := performTestAPIRequest(t, app, "GET", makeURL(amountBelowTier1, testCADCurrencyID), nil, map[string]string{
		"Authorization": regularUserToken,
	})
	mustStatus(t, cadRes, 200)
	cadBody := mustReadBody(t, cadRes)
	if !(containsAll(cadBody,
		`"second_approval_required":false`,
		`"status":"not_required"`,
		`"reason_code":"second_approval_not_required"`,
	)) {
		t.Fatalf("expected CAD amount below tier1 to avoid second approval, body=%s", cadBody)
	}

	usdRes := performTestAPIRequest(t, app, "GET", makeURL(amountBelowTier1, testUSDCurrencyID), nil, map[string]string{
		"Authorization": regularUserToken,
	})
	mustStatus(t, usdRes, 200)
	usdBody := mustReadBody(t, usdRes)
	if !(containsAll(usdBody,
		`"second_approval_required":true`,
		`"status":"candidates_available"`,
		`"reason_code":"eligible_second_approvers_available"`,
	)) {
		t.Fatalf("expected USD-converted amount to require second approval, body=%s", usdBody)
	}
}

func containsAll(body string, snippets ...string) bool {
	for _, snippet := range snippets {
		if !strings.Contains(body, snippet) {
			return false
		}
	}
	return true
}

func TestPurchaseOrderApproversRoutes_RecurringPaymentCount(t *testing.T) {
	regularUserToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	makeURL := func(endpoint string, endDate string) string {
		params := url.Values{}
		params.Set("division", "2rrfy6m2c8hazjy")
		params.Set("amount", "1000")
		params.Set("has_job", "false")
		params.Set("type", "Recurring")
		params.Set("start_date", "2026-04-30")
		params.Set("end_date", endDate)
		params.Set("frequency", "Monthly")
		return "/api/purchase_orders/" + endpoint + "?" + params.Encode()
	}
	headers := map[string]string{"Authorization": regularUserToken}

	// Two calendar-month payments are evaluated as the combined amount.
	res := performTestAPIRequest(t, app, "GET", makeURL("second_approvers", "2026-05-31"), nil, headers)
	mustStatus(t, res, 200)
	body := mustReadBody(t, res)
	if !containsAll(body, `"evaluated_amount":2000`, `"occurrences":2`) {
		t.Fatalf("expected two payments evaluated as 2000, body=%s", body)
	}

	// A single payment is a recurrence error on end_date for both endpoints,
	// not a generic failure.
	for _, endpoint := range []string{"approvers", "second_approvers"} {
		res := performTestAPIRequest(t, app, "GET", makeURL(endpoint, "2026-05-30"), nil, headers)
		mustStatus(t, res, 400)
		body := mustReadBody(t, res)
		if !containsAll(body, `"code":"invalid_recurrence"`, `"end_date":{"code":"fewer_than_two_occurrences"`) {
			t.Fatalf("expected recurrence error from %s, body=%s", endpoint, body)
		}
	}
}
