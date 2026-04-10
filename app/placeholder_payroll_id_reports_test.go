package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"
)

const (
	placeholderPayrollPlaceholderID      = "912345678"
	placeholderPayrollWeek1              = "2026-04-18"
	placeholderPayrollWeek2              = "2026-04-25"
	placeholderPayrollTimesheetWeek1ID   = "phw1sheet000001"
	placeholderPayrollTimesheetWeek2ID   = "phw2sheet000001"
	placeholderPayrollControlSheetID     = "ctw2sheet000001"
	placeholderPayrollAmendWeek2ID       = "phw2amend000001"
	placeholderPayrollExpenseID          = "phw2expense0001"
	placeholderPayrollControlExpenseID   = "ctw2expense0001"
	placeholderPayrollVendorID           = "phvendor0000001"
	placeholderPayrollControlVendorID    = "ctvendor0000001"
	placeholderPayrollVendorName         = "Placeholder Only Vendor"
	placeholderPayrollControlVendorName  = "Control Export Vendor"
	placeholderPayrollTimeDescription    = "Placeholder payroll time row"
	placeholderPayrollControlTimeDesc    = "Control payroll time row"
	placeholderPayrollExpenseDesc        = "Placeholder payroll expense row"
	placeholderPayrollControlExpenseDesc = "Control payroll expense row"
)

func TestPlaceholderPayrollIDWritebackExportsOmitRows(t *testing.T) {
	validToken := "test-secret-123"

	scenarios := []tests.ApiScenario{
		{
			Name:   "timesheet writeback omits placeholder payroll rows",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/time_sheets/" + placeholderPayrollWeek2,
			Headers: map[string]string{
				"Authorization": "Bearer " + validToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"timeSheets"`,
				`"timeAmendments"`,
			},
			TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
				defer res.Body.Close()

				var payload struct {
					TimeSheets []struct {
						ID string `json:"id"`
					} `json:"timeSheets"`
					TimeAmendments []struct {
						ID string `json:"id"`
					} `json:"timeAmendments"`
				}
				if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
					tb.Fatalf("failed to decode timesheet export response: %v", err)
				}

				sheetIDs := make([]string, 0, len(payload.TimeSheets))
				for _, sheet := range payload.TimeSheets {
					sheetIDs = append(sheetIDs, sheet.ID)
				}

				amendmentIDs := make([]string, 0, len(payload.TimeAmendments))
				for _, amendment := range payload.TimeAmendments {
					amendmentIDs = append(amendmentIDs, amendment.ID)
				}

				assertJSONIDsContain(tb, sheetIDs, placeholderPayrollControlSheetID)
				assertJSONIDsNotContain(tb, sheetIDs, placeholderPayrollTimesheetWeek2ID)
				assertJSONIDsNotContain(tb, amendmentIDs, placeholderPayrollAmendWeek2ID)
			},
		},
		{
			Name:   "expense writeback omits placeholder payroll rows",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/expenses/2000-01-01",
			Headers: map[string]string{
				"Authorization": "Bearer " + validToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"expenses"`,
			},
			TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
				defer res.Body.Close()

				var payload struct {
					Expenses []struct {
						ID          string `json:"immutableID"`
						Description string `json:"description"`
					} `json:"expenses"`
					Vendors []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"vendors"`
				}
				if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
					tb.Fatalf("failed to decode expenses export response: %v", err)
				}

				foundControl := false
				for _, expense := range payload.Expenses {
					if expense.ID == placeholderPayrollControlExpenseID && expense.Description == placeholderPayrollControlExpenseDesc {
						foundControl = true
					}
					if expense.ID == placeholderPayrollExpenseID || expense.Description == placeholderPayrollExpenseDesc {
						tb.Fatalf("placeholder expense unexpectedly exported: %+v", expense)
					}
				}
				if !foundControl {
					tb.Fatalf("missing control expense %s", placeholderPayrollControlExpenseID)
				}

				foundControlVendor := false
				for _, vendor := range payload.Vendors {
					if vendor.ID == placeholderPayrollVendorID || vendor.Name == placeholderPayrollVendorName {
						tb.Fatalf("placeholder-only vendor unexpectedly exported: %+v", vendor)
					}
					if vendor.ID == placeholderPayrollControlVendorID && vendor.Name == placeholderPayrollControlVendorName {
						foundControlVendor = true
					}
				}
				if !foundControlVendor {
					tb.Fatalf("missing control vendor %s", placeholderPayrollControlVendorID)
				}
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestPlaceholderPayrollIDReportSourcesExposeCounts(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	payrollRecord := findPayrollReportWeekEndingRecord(t, app, placeholderPayrollWeek2)
	if payrollRecord == nil {
		t.Fatal("expected payroll_report_week_endings row to exist")
	}
	if got := payrollRecord.GetInt("placeholder_payroll_id_week1_time_count"); got != 2 {
		t.Fatalf("placeholder_payroll_id_week1_time_count = %d, want 2", got)
	}
	if got := payrollRecord.GetInt("placeholder_payroll_id_week2_time_count"); got != 2 {
		t.Fatalf("placeholder_payroll_id_week2_time_count = %d, want 2", got)
	}
	if got := payrollRecord.GetInt("placeholder_payroll_id_expense_count"); got != 1 {
		t.Fatalf("placeholder_payroll_id_expense_count = %d, want 1", got)
	}

	timeTrackingRecords, err := app.FindRecordsByFilter(
		"time_tracking",
		"week_ending = {:weekEnding}",
		"",
		1,
		0,
		dbx.Params{"weekEnding": placeholderPayrollWeek2},
	)
	if err != nil {
		t.Fatalf("failed to query time_tracking: %v", err)
	}
	if len(timeTrackingRecords) != 1 {
		t.Fatalf("time_tracking rows = %d, want 1", len(timeTrackingRecords))
	}
	if got := timeTrackingRecords[0].GetInt("placeholder_payroll_id_expense_count"); got != 1 {
		t.Fatalf("time_tracking placeholder_payroll_id_expense_count = %d, want 1", got)
	}

	reportToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	scenario := tests.ApiScenario{
		Name:   "expense tracking counts expose placeholder payroll count",
		Method: http.MethodGet,
		URL:    "/api/expenses/tracking_counts",
		Headers: map[string]string{
			"Authorization": reportToken,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"committed_week_ending"`,
		},
		TestAppFactory: testutils.SetupTestApp,
		AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
			defer res.Body.Close()

			var payload []struct {
				CommittedWeekEnding              string `json:"committed_week_ending"`
				PlaceholderPayrollIDExpenseCount int    `json:"placeholder_payroll_id_expense_count"`
			}
			if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
				tb.Fatalf("failed to decode tracking counts response: %v", err)
			}

			for _, row := range payload {
				if row.CommittedWeekEnding == placeholderPayrollWeek2 {
					if row.PlaceholderPayrollIDExpenseCount != 1 {
						tb.Fatalf("placeholder_payroll_id_expense_count = %d, want 1", row.PlaceholderPayrollIDExpenseCount)
					}
					return
				}
			}

			tb.Fatalf("missing tracking counts row for %s", placeholderPayrollWeek2)
		},
	}
	scenario.Test(t)
}

func TestPlaceholderPayrollIDCSVReportsOmitRows(t *testing.T) {
	reportToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "payroll time report omits placeholder payroll rows",
			Method: http.MethodGet,
			URL:    "/api/reports/payroll_time/" + placeholderPayrollWeek2 + "/2",
			Headers: map[string]string{
				"Authorization": reportToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"payrollId,weekEnding",
				"900002,2026 Apr 25,Maclean,Fatty,Fatty Maclean",
			},
			NotExpectedContent: []string{
				placeholderPayrollPlaceholderID,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "payroll expense report omits placeholder payroll rows",
			Method: http.MethodGet,
			URL:    "/api/reports/payroll_expense/" + placeholderPayrollWeek2,
			Headers: map[string]string{
				"Authorization": reportToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"payrollId,Acct/Visa/Exp",
				placeholderPayrollControlExpenseDesc,
			},
			NotExpectedContent: []string{
				placeholderPayrollExpenseDesc,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "weekly expense report omits placeholder payroll rows",
			Method: http.MethodGet,
			URL:    "/api/reports/weekly_expense/" + placeholderPayrollWeek2,
			Headers: map[string]string{
				"Authorization": reportToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"payrollId,Acct/Visa/Exp",
				placeholderPayrollControlExpenseDesc,
			},
			NotExpectedContent: []string{
				placeholderPayrollExpenseDesc,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func assertJSONIDsContain(tb testing.TB, ids []string, expected string) {
	tb.Helper()

	for _, id := range ids {
		if id == expected {
			return
		}
	}
	tb.Fatalf("expected id %q not found in %+v", expected, ids)
}

func assertJSONIDsNotContain(tb testing.TB, ids []string, unexpected string) {
	tb.Helper()

	for _, id := range ids {
		if id == unexpected {
			tb.Fatalf("unexpected id %q found in %+v", unexpected, ids)
		}
	}
}
