package main

import (
	"encoding/csv"
	"net/http"
	"slices"
	"strconv"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
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
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			app := testutils.SetupTestApp(tb)

			insertCommittedTimeSheetForPayrollBranchColumnsTest(tb, app, payrollBranchColumnsTimeSheetID, payrollBranchColumnsWeekEnding)
			insertCommittedTimeEntryForPayrollBranchColumnsTest(tb, app, payrollBranchColumnsTimeEntryID, payrollBranchColumnsTimeSheetID, payrollReportSeedUserID, payrollBranchColumnsWeekEnding, payrollBranchTorontoID, payrollBranchRegularTimeTypeID, 8)
			insertCommittedTimeAmendmentForPayrollBranchColumnsTest(tb, app, payrollBranchColumnsAmendmentID, payrollBranchColumnsTimeSheetID, payrollReportSeedUserID, payrollBranchColumnsWeekEnding, payrollBranchThunderBayID, payrollBranchPPTOTimeTypeID, 2)
			insertCommittedTimeAmendmentForPayrollBranchColumnsTest(tb, app, payrollBranchColumnsMissingID, payrollBranchColumnsTimeSheetID, payrollReportSeedUserID, payrollBranchColumnsWeekEnding, "", payrollBranchPPTOTimeTypeID, 3)

			return app
		},
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

func insertCommittedTimeSheetForPayrollBranchColumnsTest(tb testing.TB, app core.App, id string, weekEnding string) {
	tb.Helper()

	_, err := app.NonconcurrentDB().NewQuery(`
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
	`).Bind(dbx.Params{
		"approver":    payrollReportSeedUserID,
		"committer":   payrollReportSeedCommitterID,
		"id":          id,
		"payroll_id":  payrollReportSeedPayrollID,
		"uid":         payrollReportSeedUserID,
		"week_ending": weekEnding,
	}).Execute()
	if err != nil {
		tb.Fatalf("failed to insert committed timesheet %s: %v", id, err)
	}
}

func insertCommittedTimeEntryForPayrollBranchColumnsTest(tb testing.TB, app core.App, id string, tsid string, uid string, weekEnding string, branchID string, timeTypeID string, hours float64) {
	tb.Helper()

	_, err := app.NonconcurrentDB().NewQuery(`
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
			{:branch},
			'',
			'2026-04-25 11:56:00.000Z',
			{:date},
			{:description},
			'',
			{:hours},
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
	`).Bind(dbx.Params{
		"branch":      branchID,
		"date":        weekEnding,
		"description": "branch allocation regular time",
		"hours":       hours,
		"id":          id,
		"time_type":   timeTypeID,
		"tsid":        tsid,
		"uid":         uid,
		"week_ending": weekEnding,
	}).Execute()
	if err != nil {
		tb.Fatalf("failed to insert committed time entry %s: %v", id, err)
	}
}

func insertCommittedTimeAmendmentForPayrollBranchColumnsTest(tb testing.TB, app core.App, id string, tsid string, uid string, weekEnding string, branchID string, timeTypeID string, hours float64) {
	tb.Helper()

	_, err := app.NonconcurrentDB().NewQuery(`
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
			{:branch},
			'',
			'2026-04-25 12:06:00.000Z',
			{:committed_week_ending},
			{:committer},
			'2026-04-25 12:01:00.000Z',
			{:creator},
			{:date},
			{:description},
			'',
			{:hours},
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
	`).Bind(dbx.Params{
		"branch":                branchID,
		"committed_week_ending": weekEnding,
		"committer":             payrollReportSeedCommitterID,
		"creator":               uid,
		"date":                  weekEnding,
		"description":           "branch allocation ppto amendment",
		"hours":                 hours,
		"id":                    id,
		"time_type":             timeTypeID,
		"tsid":                  tsid,
		"uid":                   uid,
		"week_ending":           weekEnding,
	}).Execute()
	if err != nil {
		tb.Fatalf("failed to insert committed time amendment %s: %v", id, err)
	}
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
