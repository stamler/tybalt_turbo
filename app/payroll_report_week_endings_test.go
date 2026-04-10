package main

import (
	"testing"
	"tybalt/internal/testseed"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	payrollReportSeedUserID      = "f2j5a8vk006baub"
	payrollReportSeedCommitterID = "wegviunlyr2jjjv"
	payrollReportSeedKindID      = "prj0kind0000001"
	payrollReportSeedPayrollID   = "9999"
)

func TestPayrollReportWeekEndings_IncludesExpenseOnlyPayPeriods(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	record := findPayrollReportWeekEndingRecord(t, app, "2026-04-11")
	if record == nil {
		t.Fatal("expected expense-only payroll period to appear in payroll_report_week_endings")
	}

	if got := record.GetInt("committed_timesheet_count"); got != 0 {
		t.Fatalf("expected 0 committed timesheets for expense-only row, got %d", got)
	}
	if got := record.GetInt("committed_expense_count"); got != 1 {
		t.Fatalf("expected 1 committed expense for expense-only row, got %d", got)
	}

	if invalidRecord := findPayrollReportWeekEndingRecord(t, app, "2026-04-04"); invalidRecord != nil {
		t.Fatal("expected non-payroll pay_period_ending to be excluded from payroll_report_week_endings")
	}
}

func TestPayrollReportWeekEndings_CountsCommittedTimeSheetsAndExpenses(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	record := findPayrollReportWeekEndingRecord(t, app, "2026-04-25")
	if record == nil {
		t.Fatal("expected payroll row with committed week 1, week 2, and expense data to appear")
	}

	if got := record.GetInt("committed_timesheet_count"); got != 5 {
		t.Fatalf("expected 5 committed timesheets to be counted, got %d", got)
	}
	if got := record.GetInt("committed_expense_count"); got != 3 {
		t.Fatalf("expected 3 committed expenses to be counted, got %d", got)
	}
}

func findPayrollReportWeekEndingRecord(t *testing.T, app core.App, weekEnding string) *core.Record {
	t.Helper()

	records, err := app.FindRecordsByFilter(
		"payroll_report_week_endings",
		"week_ending = {:weekEnding}",
		"",
		1,
		0,
		dbx.Params{"weekEnding": weekEnding},
	)
	if err != nil {
		t.Fatalf("failed to query payroll_report_week_endings for %s: %v", weekEnding, err)
	}
	if len(records) == 0 {
		return nil
	}

	return records[0]
}
