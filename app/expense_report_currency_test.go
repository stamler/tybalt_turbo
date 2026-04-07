package main

import (
	"encoding/csv"
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"
)

func TestExpenseReportKeepsTotalInCADAndAddsSourceCurrencyColumns(t *testing.T) {
	reportToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	app := testutils.SetupTestApp(t)
	res := performTestAPIRequest(
		t,
		app,
		http.MethodGet,
		"/api/reports/payroll_expense/2026-06-06",
		nil,
		map[string]string{"Authorization": reportToken},
	)
	mustStatus(t, res, http.StatusOK)

	rows, err := csv.NewReader(strings.NewReader(mustReadBody(t, res))).ReadAll()
	if err != nil {
		t.Fatalf("failed to parse csv response: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("expected at least one data row, got %d rows", len(rows))
	}

	headerIndex := map[string]int{}
	for i, header := range rows[0] {
		headerIndex[header] = i
	}

	for _, requiredHeader := range []string{"Total", "Description", "currency", "foreign_currency_total"} {
		if _, ok := headerIndex[requiredHeader]; !ok {
			t.Fatalf("missing csv header %q in %v", requiredHeader, rows[0])
		}
	}

	var foreignRow []string
	var cadRow []string
	for _, row := range rows[1:] {
		switch row[headerIndex["Description"]] {
		case "Foreign expense report row":
			foreignRow = row
		case "CAD expense report row":
			cadRow = row
		}
	}

	if foreignRow == nil {
		t.Fatal("missing foreign expense report row")
	}
	if got := foreignRow[headerIndex["Total"]]; got != "91.11" {
		t.Fatalf("expected foreign Total to stay in CAD as 91.11, got %q", got)
	}
	if got := foreignRow[headerIndex["currency"]]; got != "USD" {
		t.Fatalf("expected foreign currency code USD, got %q", got)
	}
	if got := foreignRow[headerIndex["foreign_currency_total"]]; got != "100" {
		t.Fatalf("expected foreign_currency_total 100, got %q", got)
	}

	if cadRow == nil {
		t.Fatal("missing CAD expense report row")
	}
	if got := cadRow[headerIndex["Total"]]; got != "88.88" {
		t.Fatalf("expected CAD Total 88.88, got %q", got)
	}
	if got := cadRow[headerIndex["currency"]]; got != "CAD" {
		t.Fatalf("expected CAD currency code, got %q", got)
	}
	if got := cadRow[headerIndex["foreign_currency_total"]]; got != "" {
		t.Fatalf("expected blank foreign_currency_total for CAD row, got %q", got)
	}
}
