package main

import (
	"encoding/csv"
	"net/http"
	"slices"
	"strconv"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

const (
	payrollBranchColumnsWeekEnding  = "2030-01-12"
	payrollBranchColumnsTimeSheetID = "ptbranchsheet001"
	payrollBranchColumnsTimeEntryID = "ptbranchentry001"
	payrollBranchColumnsAmendmentID = "ptbranchamend001"
	payrollBranchColumnsMissingID   = "ptbranchamend002"
	payrollBranchTorontoID          = "xeq9q81q5307f70"
	payrollBranchThunderBayID       = "80875lm27v8wgi4"
	payrollBranchRegularTimeTypeID  = "sdyfl3q7j7ap849"
	payrollBranchPPTOTimeTypeID     = "mn8q8lnnqln0kt6"
	payrollBranchExpectedThunderBay = 2.0
	payrollBranchExpectedToronto    = 8.0
	payrollBranchExpectedUnassigned = 3.0
)

func TestPayrollTimeReport_IncludesOrderedBranchAllocationColumns(t *testing.T) {
	reportToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	expectedBranchHeaders := []string{
		"Collingwood",
		"Corporate",
		"Fort Frances",
		"Kenora",
		"Kitchener-Waterloo",
		"Ottawa",
		"Thunder Bay",
		"Toronto",
		"Unassigned",
	}

	scenario := tests.ApiScenario{
		Name:   "payroll time report appends ordered branch allocation columns",
		Method: http.MethodGet,
		URL:    "/api/reports/payroll_time/" + payrollBranchColumnsWeekEnding + "/2",
		Headers: map[string]string{
			"Authorization": reportToken,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			"payrollId,weekEnding",
		},
		TestAppFactory: testutils.SetupTestApp,
		AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
			defer res.Body.Close()

			rows, err := csv.NewReader(res.Body).ReadAll()
			if err != nil {
				tb.Fatalf("failed to parse payroll csv: %v", err)
			}
			if len(rows) < 2 {
				tb.Fatalf("expected header plus at least one data row, got %d rows", len(rows))
			}

			headers := rows[0]
			salaryIndex := slices.Index(headers, "salary")
			if salaryIndex == -1 {
				tb.Fatalf("missing salary header in %+v", headers)
			}

			if got := headers[salaryIndex+1:]; !slices.Equal(got, expectedBranchHeaders) {
				tb.Fatalf("branch headers = %+v, want %+v", got, expectedBranchHeaders)
			}

			rowByPayrollID := map[string]map[string]string{}
			for _, record := range rows[1:] {
				if len(record) != len(headers) {
					tb.Fatalf("row has %d columns, want %d: %+v", len(record), len(headers), record)
				}
				row := map[string]string{}
				for i, header := range headers {
					row[header] = record[i]
				}
				rowByPayrollID[row["payrollId"]] = row
			}

			row, ok := rowByPayrollID[payrollReportSeedPayrollID]
			if !ok {
				tb.Fatalf("missing payroll row for payrollId %s", payrollReportSeedPayrollID)
			}

			assertCSVFloatEquals(tb, row["hours worked"], 8)
			assertCSVFloatEquals(tb, row["PPTO"], 5)
			assertCSVFloatEquals(tb, row["Thunder Bay"], payrollBranchExpectedThunderBay)
			assertCSVFloatEquals(tb, row["Toronto"], payrollBranchExpectedToronto)
			assertCSVFloatEquals(tb, row["Collingwood"], 0)
			assertCSVFloatEquals(tb, row["Unassigned"], payrollBranchExpectedUnassigned)
		},
	}

	scenario.Test(t)
}

func assertCSVFloatEquals(tb testing.TB, value string, want float64) {
	tb.Helper()

	got, err := strconv.ParseFloat(value, 64)
	if err != nil {
		tb.Fatalf("failed to parse %q as float: %v", value, err)
	}
	if got != want {
		tb.Fatalf("csv value = %v, want %v", got, want)
	}
}
