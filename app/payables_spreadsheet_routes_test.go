package main

import (
	"net/http"
	"testing"
	"time"
	"tybalt/internal/testutils"
	"tybalt/reports"

	"github.com/pocketbase/pocketbase/tests"
)

func setupTestAppWithFixedPayablesSpreadsheetNow(tb testing.TB) *tests.TestApp {
	tb.Helper()

	app := testutils.SetupTestApp(tb)
	reports.SetPayablesSpreadsheetNowForTest(app, func() time.Time {
		return time.Date(2026, time.March, 13, 12, 0, 0, 0, time.UTC)
	})
	return app
}

func TestPayablesSpreadsheetRoutesAuthAndValidation(t *testing.T) {
	reportToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	noReportClaimToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "dates endpoint requires auth",
			Method:         http.MethodGet,
			URL:            "/api/reports/payables_spreadsheet_dates",
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				`"status":401`,
			},
			TestAppFactory: setupTestAppWithFixedPayablesSpreadsheetNow,
		},
		{
			Name:           "dates endpoint requires report claim",
			Method:         http.MethodGet,
			URL:            "/api/reports/payables_spreadsheet_dates",
			Headers:        map[string]string{"Authorization": noReportClaimToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`You are not authorized to view this report.`,
			},
			TestAppFactory: setupTestAppWithFixedPayablesSpreadsheetNow,
		},
		{
			Name:           "daily endpoint validates date format",
			Method:         http.MethodGet,
			URL:            "/api/reports/payables_spreadsheet/2026-3-9",
			Headers:        map[string]string{"Authorization": reportToken},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`Date must be in YYYY-MM-DD format.`,
			},
			TestAppFactory: setupTestAppWithFixedPayablesSpreadsheetNow,
		},
		{
			Name:           "daily endpoint requires a completed utc day",
			Method:         http.MethodGet,
			URL:            "/api/reports/payables_spreadsheet/2026-03-13",
			Headers:        map[string]string{"Authorization": reportToken},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`Date must be at least one UTC day old.`,
			},
			TestAppFactory: setupTestAppWithFixedPayablesSpreadsheetNow,
		},
		{
			Name:           "monthly endpoint validates yymm format",
			Method:         http.MethodGet,
			URL:            "/api/reports/payables_spreadsheet_monthly/202603",
			Headers:        map[string]string{"Authorization": reportToken},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`Yymm must be a 4-digit string (e.g. 2603).`,
			},
			TestAppFactory: setupTestAppWithFixedPayablesSpreadsheetNow,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestPayablesSpreadsheetRoutesDataAndFiltering(t *testing.T) {
	reportToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "dates endpoint returns recent approval dates and uses second approval date",
			Method:         http.MethodGet,
			URL:            "/api/reports/payables_spreadsheet_dates",
			Headers:        map[string]string{"Authorization": reportToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"2026-03-10"`,
				`"2026-03-12"`,
			},
			NotExpectedContent: []string{
				`"2026-03-11"`,
				`"2026-03-09"`,
			},
			TestAppFactory: setupTestAppWithFixedPayablesSpreadsheetNow,
		},
		{
			Name:           "daily csv includes header and matching row",
			Method:         http.MethodGet,
			URL:            "/api/reports/payables_spreadsheet/2026-03-12",
			Headers:        map[string]string{"Authorization": reportToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"Acct/Visa/Exp,Job #,Div,Branch,type,Date,Mon,Year",
				"Total,Currency,Unit",
				"Seeded payables second approval fixture",
				"2712-0102",
				"One-Time",
				",654.32,CAD,",
				"TURBO",
			},
			NotExpectedContent: []string{
				"Seeded payables excluded control-range fixture",
			},
			TestAppFactory: setupTestAppWithFixedPayablesSpreadsheetNow,
		},
		{
			Name:           "daily tsv omits header",
			Method:         http.MethodGet,
			URL:            "/api/reports/payables_spreadsheet/2026-03-12?format=tsv",
			Headers:        map[string]string{"Authorization": reportToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"Seeded payables second approval fixture",
				"2712-0102",
				"\tOne-Time\t",
				"\t654.32\tCAD\t",
				"\tTURBO\t",
			},
			NotExpectedContent: []string{
				"Acct/Visa/Exp\tJob #\tDiv",
			},
			TestAppFactory: setupTestAppWithFixedPayablesSpreadsheetNow,
		},
		{
			Name:           "monthly report filters by po number prefix",
			Method:         http.MethodGet,
			URL:            "/api/reports/payables_spreadsheet_monthly/2712",
			Headers:        map[string]string{"Authorization": reportToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"Seeded payables included fixture",
				"Seeded payables second approval fixture",
				"2712-0101",
				"2712-0102",
			},
			NotExpectedContent: []string{
				"Seeded payables excluded control-range fixture",
				"2712-8001",
			},
			TestAppFactory: setupTestAppWithFixedPayablesSpreadsheetNow,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
