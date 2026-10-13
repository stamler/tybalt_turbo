package main

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
)

// Fixtures: reimbursements@example.com (reimbtestuser01) owns the reimb*
// expenses in test_seed_data/data/expenses.csv. Mileage tiers are 0.70/km
// below 5000 km and 0.64/km above (2025-01-05 rates). 4800 km in the
// 2026-06-20 period puts the next claim across the tier boundary, and the
// stored totals reflect an out-of-order commit that must be ignored in favour
// of the payroll report's date-ordered calculation. Allowance (stored total
// 999) is priced from expense_rates: breakfast 20 + dinner 30. The OnAccount,
// FuelCard, CorporateCreditCard, and uncommitted rows must be excluded.
const (
	reimbursementsOwnerUID   = "reimbtestuser01"
	reimbursementsOwnerEmail = "reimbursements@example.com"
	reimbursementsPeriod     = "2026-07-04"
)

type reimbursementPeriodResponse struct {
	PayPeriodEnding string  `json:"pay_period_ending"`
	PayDate         string  `json:"pay_date"`
	ExpenseCount    int     `json:"expense_count"`
	Total           float64 `json:"total"`
}

type reimbursementRowResponse struct {
	ID                   string  `json:"id"`
	PaymentType          string  `json:"payment_type"`
	Description          string  `json:"description"`
	PayDate              string  `json:"pay_date"`
	CurrencyCode         string  `json:"currency_code"`
	ForeignCurrencyTotal float64 `json:"foreign_currency_total"`
	Amount               float64 `json:"amount"`
}

func TestExpenseReimbursementsPeriodsAndRows(t *testing.T) {
	ownerToken, err := testutils.GenerateRecordToken("users", reimbursementsOwnerEmail)
	if err != nil {
		t.Fatal(err)
	}

	app := testutils.SetupTestApp(t)

	res := performTestAPIRequest(t, app, http.MethodGet, "/api/expenses/reimbursements", nil, map[string]string{"Authorization": ownerToken})
	mustStatus(t, res, http.StatusOK)

	var periods []reimbursementPeriodResponse
	if err := json.Unmarshal(res.Body.Bytes(), &periods); err != nil {
		t.Fatalf("failed to decode periods: %v", err)
	}
	periodsByEnding := map[string]reimbursementPeriodResponse{}
	for i, p := range periods {
		periodsByEnding[p.PayPeriodEnding] = p
		if i > 0 && periods[i-1].PayPeriodEnding < p.PayPeriodEnding {
			t.Fatalf("expected periods newest first, got %v", periods)
		}
	}
	// Other users' committed expenses (e.g. author@soup.com's 2026-06-06
	// Expense rows) must not appear.
	if len(periods) != 2 {
		t.Fatalf("expected only the owner's 2 pay periods, got %+v", periods)
	}

	july := periodsByEnding[reimbursementsPeriod]
	if july.PayDate != "2026-07-17" {
		t.Fatalf("expected pay date 2026-07-17 (Friday 13 days after period end), got %q", july.PayDate)
	}
	// 40 + 12.50 + 50 + 91.11 + (200*0.70 + 100*0.64) + 100*0.64
	if july.ExpenseCount != 6 || math.Abs(july.Total-461.61) > 0.001 {
		t.Fatalf("expected 6 expenses totalling 461.61, got %d totalling %v", july.ExpenseCount, july.Total)
	}
	june := periodsByEnding["2026-06-20"]
	if june.PayDate != "2026-07-03" || june.ExpenseCount != 1 || math.Abs(june.Total-3360) > 0.001 {
		t.Fatalf("unexpected 2026-06-20 period: %+v", june)
	}

	res = performTestAPIRequest(t, app, http.MethodGet, "/api/expenses/reimbursements/"+reimbursementsPeriod, nil, map[string]string{"Authorization": ownerToken})
	mustStatus(t, res, http.StatusOK)

	var rows []reimbursementRowResponse
	if err := json.Unmarshal(res.Body.Bytes(), &rows); err != nil {
		t.Fatalf("failed to decode rows: %v", err)
	}
	amounts := map[string]float64{}
	for _, row := range rows {
		amounts[row.ID] = row.Amount
		if row.PayDate != "2026-07-17" {
			t.Fatalf("expected pay date on row %q, got %q", row.ID, row.PayDate)
		}
		switch row.ID {
		case "reimballowance1":
			if row.Description != "Breakfast, Dinner" {
				t.Fatalf("expected allowance description %q, got %q", "Breakfast, Dinner", row.Description)
			}
		case "reimbforeign001":
			if row.CurrencyCode != "USD" || row.ForeignCurrencyTotal != 100 {
				t.Fatalf("expected USD 100 foreign total, got %+v", row)
			}
		}
	}
	want := map[string]float64{
		"reimbexpense001": 40,
		"reimbpersonal01": 12.5,
		"reimballowance1": 50,
		"reimbforeign001": 91.11,
		"reimbmileage002": 204,
		"reimbmileage003": 64,
	}
	if len(amounts) != len(want) {
		t.Fatalf("expected rows %v, got %v", want, amounts)
	}
	for id, amount := range want {
		got, ok := amounts[id]
		if !ok || math.Abs(got-amount) > 0.001 {
			t.Fatalf("expected %s amount %v, got %v (present=%v)", id, amount, got, ok)
		}
	}

	// The amounts must agree with what the payroll expense report pays out.
	reportToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}
	res = performTestAPIRequest(t, app, http.MethodGet, "/api/reports/payroll_expense/"+reimbursementsPeriod, nil, map[string]string{"Authorization": reportToken})
	mustStatus(t, res, http.StatusOK)
	csvRows, err := csv.NewReader(strings.NewReader(mustReadBody(t, res))).ReadAll()
	if err != nil {
		t.Fatalf("failed to parse csv response: %v", err)
	}
	headerIndex := map[string]int{}
	for i, header := range csvRows[0] {
		headerIndex[header] = i
	}
	var ownerName string
	if err := app.DB().NewQuery("SELECT given_name || ' ' || surname FROM profiles WHERE uid = {:uid}").
		Bind(dbx.Params{"uid": reimbursementsOwnerUID}).Row(&ownerName); err != nil {
		t.Fatal(err)
	}
	reportTotal := 0.0
	for _, row := range csvRows[1:] {
		if row[headerIndex["Employee"]] != ownerName {
			continue
		}
		switch row[headerIndex["Acct/Visa/Exp"]] {
		case "Expense", "PersonalReimbursement", "Allowance", "Meals", "Mileage":
			total, err := strconv.ParseFloat(row[headerIndex["Total"]], 64)
			if err != nil {
				t.Fatalf("bad Total %q: %v", row[headerIndex["Total"]], err)
			}
			reportTotal += total
		}
	}
	if math.Abs(reportTotal-july.Total) > 0.001 {
		t.Fatalf("expected reimbursements total %v to match payroll report total %v", july.Total, reportTotal)
	}
}

func TestExpenseReimbursementsRequiresAuthAndValidDate(t *testing.T) {
	app := testutils.SetupTestApp(t)

	res := performTestAPIRequest(t, app, http.MethodGet, "/api/expenses/reimbursements", nil, nil)
	mustStatus(t, res, http.StatusUnauthorized)

	ownerToken, err := testutils.GenerateRecordToken("users", reimbursementsOwnerEmail)
	if err != nil {
		t.Fatal(err)
	}
	res = performTestAPIRequest(t, app, http.MethodGet, "/api/expenses/reimbursements/not-a-date", nil, map[string]string{"Authorization": ownerToken})
	mustStatus(t, res, http.StatusBadRequest)
}
