package main

import (
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestReportRoutesRequireReportClaim(t *testing.T) {
	reportToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	noReportToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "payroll route rejects logged-in user without report claim",
			Method:         http.MethodGet,
			URL:            "/api/reports/payroll_time/2026-04-25/2",
			Headers:        map[string]string{"Authorization": noReportToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				"You are not authorized to view this report",
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "payroll route allows report holder",
			Method:         http.MethodGet,
			URL:            "/api/reports/payroll_time/2026-04-25/2",
			Headers:        map[string]string{"Authorization": reportToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"payrollId,weekEnding",
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "payroll expense route rejects logged-in user without report claim",
			Method:         http.MethodGet,
			URL:            "/api/reports/payroll_expense/2026-04-25",
			Headers:        map[string]string{"Authorization": noReportToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				"You are not authorized to view this report",
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "payroll receipts route rejects logged-in user without report claim",
			Method:         http.MethodGet,
			URL:            "/api/reports/payroll_receipts/2026-04-25",
			Headers:        map[string]string{"Authorization": noReportToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				"You are not authorized to view this report",
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "weekly route rejects logged-in user without report claim",
			Method:         http.MethodGet,
			URL:            "/api/reports/weekly_time/2026-04-25",
			Headers:        map[string]string{"Authorization": noReportToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				"You are not authorized to view this report",
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestReportWeekEndingViewsRequireReportClaim(t *testing.T) {
	reportToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	noReportToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "payroll report week endings reject logged-in user without report claim",
			Method:         http.MethodGet,
			URL:            "/api/collections/payroll_report_week_endings/records",
			Headers:        map[string]string{"Authorization": noReportToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"items":[]`,
				`"totalItems":0`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "payroll report week endings allow report holder",
			Method:         http.MethodGet,
			URL:            "/api/collections/payroll_report_week_endings/records",
			Headers:        map[string]string{"Authorization": reportToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"week_ending":"2026-04-25"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "time report week endings reject logged-in user without report claim",
			Method:         http.MethodGet,
			URL:            "/api/collections/time_report_week_endings/records",
			Headers:        map[string]string{"Authorization": noReportToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"items":[]`,
				`"totalItems":0`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
