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
