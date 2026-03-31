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
	placeholderPayrollTestUserID         = payrollReportSeedUserID
	placeholderPayrollControlUserID      = "etysnrlup2f6bak"
	placeholderPayrollPlaceholderID      = "912345678"
	placeholderPayrollWeek1              = "2026-04-18"
	placeholderPayrollWeek2              = "2026-04-25"
	placeholderPayrollApproverID         = payrollReportSeedCommitterID
	placeholderPayrollTimeTypeID         = "sdyfl3q7j7ap849"
	placeholderPayrollTimesheetWeek1ID   = "phw1sheet000001"
	placeholderPayrollTimesheetWeek2ID   = "phw2sheet000001"
	placeholderPayrollControlSheetID     = "ctw2sheet000001"
	placeholderPayrollEntryWeek1ID       = "phw1entry000001"
	placeholderPayrollEntryWeek2ID       = "phw2entry000001"
	placeholderPayrollControlEntryID     = "ctw2entry000001"
	placeholderPayrollAmendWeek1ID       = "phw1amend000001"
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

func setupPlaceholderPayrollIDReportApp(tb testing.TB) *tests.TestApp {
	tb.Helper()

	app := testutils.SetupTestApp(tb)

	mustExec(tb, app, `
		UPDATE admin_profiles
		SET payroll_id = {:payroll_id}
		WHERE uid = {:uid}
	`, dbx.Params{
		"payroll_id": placeholderPayrollPlaceholderID,
		"uid":        placeholderPayrollTestUserID,
	})

	insertCommittedTimeSheetForUser(tb, app, placeholderPayrollTimesheetWeek1ID, placeholderPayrollTestUserID, placeholderPayrollWeek1)
	insertCommittedTimeSheetForUser(tb, app, placeholderPayrollTimesheetWeek2ID, placeholderPayrollTestUserID, placeholderPayrollWeek2)
	insertCommittedTimeSheetForUser(tb, app, placeholderPayrollControlSheetID, placeholderPayrollControlUserID, placeholderPayrollWeek2)

	insertTimeEntryForTimesheet(tb, app, placeholderPayrollEntryWeek1ID, placeholderPayrollTimesheetWeek1ID, placeholderPayrollTestUserID, placeholderPayrollWeek1, placeholderPayrollTimeDescription+" week1")
	insertTimeEntryForTimesheet(tb, app, placeholderPayrollEntryWeek2ID, placeholderPayrollTimesheetWeek2ID, placeholderPayrollTestUserID, placeholderPayrollWeek2, placeholderPayrollTimeDescription)
	insertTimeEntryForTimesheet(tb, app, placeholderPayrollControlEntryID, placeholderPayrollControlSheetID, placeholderPayrollControlUserID, placeholderPayrollWeek2, placeholderPayrollControlTimeDesc)

	insertCommittedTimeAmendmentForUser(tb, app, placeholderPayrollAmendWeek1ID, placeholderPayrollTimesheetWeek1ID, placeholderPayrollTestUserID, placeholderPayrollWeek1, placeholderPayrollTimeDescription+" amendment week1")
	insertCommittedTimeAmendmentForUser(tb, app, placeholderPayrollAmendWeek2ID, placeholderPayrollTimesheetWeek2ID, placeholderPayrollTestUserID, placeholderPayrollWeek2, placeholderPayrollTimeDescription+" amendment week2")

	insertVendor(tb, app, placeholderPayrollVendorID, placeholderPayrollVendorName)
	insertVendor(tb, app, placeholderPayrollControlVendorID, placeholderPayrollControlVendorName)

	insertCommittedExpenseForUser(tb, app, placeholderPayrollExpenseID, placeholderPayrollTestUserID, placeholderPayrollExpenseDesc, placeholderPayrollWeek2, placeholderPayrollWeek2, placeholderPayrollWeek2, placeholderPayrollVendorID)
	insertCommittedExpenseForUser(tb, app, placeholderPayrollControlExpenseID, placeholderPayrollControlUserID, placeholderPayrollControlExpenseDesc, placeholderPayrollWeek2, placeholderPayrollWeek2, placeholderPayrollWeek2, placeholderPayrollControlVendorID)

	return app
}

func mustExec(tb testing.TB, app *tests.TestApp, query string, params dbx.Params) {
	tb.Helper()

	if _, err := app.NonconcurrentDB().NewQuery(query).Bind(params).Execute(); err != nil {
		tb.Fatal(err)
	}
}

func insertCommittedTimeSheetForUser(tb testing.TB, app *tests.TestApp, id string, uid string, weekEnding string) {
	tb.Helper()

	mustExec(tb, app, `
		INSERT INTO time_sheets (
			_imported,
			approved,
			approver,
			committed,
			committer,
			created,
			id,
			payroll_id,
			rejected,
			rejection_reason,
			rejector,
			salary,
			submitted,
			uid,
			updated,
			week_ending,
			work_week_hours
		) VALUES (
			0,
			'2026-04-25 12:00:00.000Z',
			{:approver},
			'2026-04-25 12:05:00.000Z',
			{:committer},
			'2026-04-25 11:55:00.000Z',
			{:id},
			{:payroll_id},
			'',
			'',
			'',
			1,
			1,
			{:uid},
			'2026-04-25 12:05:00.000Z',
			{:week_ending},
			40
		)
	`, dbx.Params{
		"approver":    placeholderPayrollApproverID,
		"committer":   placeholderPayrollApproverID,
		"id":          id,
		"payroll_id":  payrollIDForUser(uid),
		"uid":         uid,
		"week_ending": weekEnding,
	})
}

func insertTimeEntryForTimesheet(tb testing.TB, app *tests.TestApp, id string, tsid string, uid string, weekEnding string, description string) {
	tb.Helper()

	mustExec(tb, app, `
		INSERT INTO time_entries (
			_imported,
			branch,
			category,
			created,
			date,
			description,
			division,
			hours,
			id,
			job,
			meals_hours,
			payout_request_amount,
			role,
			time_type,
			tsid,
			uid,
			updated,
			week_ending,
			work_record
		) VALUES (
			0,
			'',
			'',
			'2026-04-25 11:56:00.000Z',
			{:date},
			{:description},
			'',
			8,
			{:id},
			'',
			0,
			0,
			'',
			{:time_type},
			{:tsid},
			{:uid},
			'2026-04-25 11:56:00.000Z',
			{:week_ending},
			''
		)
	`, dbx.Params{
		"date":        weekEnding,
		"description": description,
		"id":          id,
		"time_type":   placeholderPayrollTimeTypeID,
		"tsid":        tsid,
		"uid":         uid,
		"week_ending": weekEnding,
	})
}

func insertCommittedTimeAmendmentForUser(tb testing.TB, app *tests.TestApp, id string, tsid string, uid string, weekEnding string, description string) {
	tb.Helper()

	mustExec(tb, app, `
		INSERT INTO time_amendments (
			_imported,
			branch,
			category,
			committed,
			committed_week_ending,
			committer,
			created,
			creator,
			date,
			description,
			division,
			hours,
			id,
			job,
			meals_hours,
			payout_request_amount,
			skip_tsid_check,
			time_type,
			tsid,
			uid,
			updated,
			week_ending,
			work_record
		) VALUES (
			0,
			'',
			'',
			'2026-04-25 12:06:00.000Z',
			{:committed_week_ending},
			{:committer},
			'2026-04-25 12:01:00.000Z',
			{:creator},
			{:date},
			{:description},
			'',
			2,
			{:id},
			'',
			0,
			0,
			0,
			{:time_type},
			{:tsid},
			{:uid},
			'2026-04-25 12:06:00.000Z',
			{:week_ending},
			''
		)
	`, dbx.Params{
		"committed_week_ending": weekEnding,
		"committer":             placeholderPayrollApproverID,
		"creator":               uid,
		"date":                  weekEnding,
		"description":           description,
		"id":                    id,
		"time_type":             placeholderPayrollTimeTypeID,
		"tsid":                  tsid,
		"uid":                   uid,
		"week_ending":           weekEnding,
	})
}

func insertVendor(tb testing.TB, app *tests.TestApp, id string, name string) {
	tb.Helper()

	mustExec(tb, app, `
		INSERT INTO vendors (
			_imported,
			alias,
			created,
			id,
			name,
			status,
			updated
		) VALUES (
			0,
			'',
			'2026-04-25 11:54:00.000Z',
			{:id},
			{:name},
			'Active',
			'2026-04-25 11:54:00.000Z'
		)
	`, dbx.Params{
		"id":   id,
		"name": name,
	})
}

func insertCommittedExpenseForUser(tb testing.TB, app *tests.TestApp, id string, uid string, description string, date string, payPeriodEnding string, committedWeekEnding string, vendorID string) {
	tb.Helper()

	mustExec(tb, app, `
		INSERT INTO expenses (
			_imported,
			allowance_types,
			approved,
			approver,
			attachment,
			attachment_hash,
			branch,
			category,
			cc_last_4_digits,
			committed,
			committed_week_ending,
			committer,
			created,
			date,
			description,
			distance,
			division,
			id,
			job,
			kind,
			pay_period_ending,
			payment_type,
			purchase_order,
			rejected,
			rejection_reason,
			rejector,
			submitted,
			total,
			uid,
			updated,
			vendor
		) VALUES (
			0,
			'[]',
			'2026-04-25 12:00:00.000Z',
			{:approver},
			'',
			'',
			'',
			'',
			'',
			'2026-04-25 12:05:00.000Z',
			{:committed_week_ending},
			{:committer},
			'2026-04-25 11:55:00.000Z',
			{:date},
			{:description},
			0,
			'',
			{:id},
			'',
			{:kind},
			{:pay_period_ending},
			'OnAccount',
			'',
			'',
			'',
			'',
			1,
			123.45,
			{:uid},
			'2026-04-25 12:05:00.000Z',
			{:vendor}
		)
	`, dbx.Params{
		"approver":              placeholderPayrollApproverID,
		"committed_week_ending": committedWeekEnding,
		"committer":             placeholderPayrollApproverID,
		"date":                  date,
		"description":           description,
		"id":                    id,
		"kind":                  payrollReportSeedKindID,
		"pay_period_ending":     payPeriodEnding,
		"uid":                   uid,
		"vendor":                vendorID,
	})
}

func payrollIDForUser(uid string) string {
	if uid == placeholderPayrollTestUserID {
		return placeholderPayrollPlaceholderID
	}
	if uid == placeholderPayrollControlUserID {
		// This is intentionally in the 9xxxxxx range without being a generated placeholder.
		return "900002"
	}
	return payrollReportSeedPayrollID
}

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
			TestAppFactory: setupPlaceholderPayrollIDReportApp,
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
			TestAppFactory: setupPlaceholderPayrollIDReportApp,
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
	app := setupPlaceholderPayrollIDReportApp(t)
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
		TestAppFactory: setupPlaceholderPayrollIDReportApp,
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
			TestAppFactory: setupPlaceholderPayrollIDReportApp,
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
			TestAppFactory: setupPlaceholderPayrollIDReportApp,
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
			TestAppFactory: setupPlaceholderPayrollIDReportApp,
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
