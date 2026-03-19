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

	insertCommittedExpenseForPayrollReportTest(t, app, payrollExpenseInsertParams{
		id:                  "pexp00000000001",
		description:         "expense-only payroll row",
		date:                "2026-04-10",
		payPeriodEnding:     "2026-04-11",
		committedWeekEnding: "2026-04-11",
		total:               123.45,
	})
	insertCommittedExpenseForPayrollReportTest(t, app, payrollExpenseInsertParams{
		id:                  "pexp00000000002",
		description:         "invalid payroll ending should be ignored",
		date:                "2026-04-04",
		payPeriodEnding:     "2026-04-04",
		committedWeekEnding: "2026-04-11",
		total:               55.00,
	})

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

	insertCommittedTimeSheetForPayrollReportTest(t, app, payrollTimeSheetInsertParams{
		id:         "pts000000000001",
		weekEnding: "2026-04-18",
	})
	insertCommittedTimeSheetForPayrollReportTest(t, app, payrollTimeSheetInsertParams{
		id:         "pts000000000002",
		weekEnding: "2026-04-25",
	})
	insertCommittedExpenseForPayrollReportTest(t, app, payrollExpenseInsertParams{
		id:                  "pexp00000000003",
		description:         "combined payroll row",
		date:                "2026-04-24",
		payPeriodEnding:     "2026-04-25",
		committedWeekEnding: "2026-04-25",
		total:               200.00,
	})

	record := findPayrollReportWeekEndingRecord(t, app, "2026-04-25")
	if record == nil {
		t.Fatal("expected payroll row with committed week 1, week 2, and expense data to appear")
	}

	if got := record.GetInt("committed_timesheet_count"); got != 2 {
		t.Fatalf("expected 2 committed timesheets to be counted, got %d", got)
	}
	if got := record.GetInt("committed_expense_count"); got != 1 {
		t.Fatalf("expected 1 committed expense to be counted, got %d", got)
	}
}

type payrollExpenseInsertParams struct {
	id                  string
	description         string
	date                string
	payPeriodEnding     string
	committedWeekEnding string
	total               float64
}

func insertCommittedExpenseForPayrollReportTest(t *testing.T, app core.App, params payrollExpenseInsertParams) {
	t.Helper()

	_, err := app.NonconcurrentDB().NewQuery(`
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
			{:total},
			{:uid},
			'2026-04-25 12:05:00.000Z',
			''
		)
	`).Bind(dbx.Params{
		"approver":              payrollReportSeedUserID,
		"committed_week_ending": params.committedWeekEnding,
		"committer":             payrollReportSeedCommitterID,
		"date":                  params.date,
		"description":           params.description,
		"id":                    params.id,
		"kind":                  payrollReportSeedKindID,
		"pay_period_ending":     params.payPeriodEnding,
		"total":                 params.total,
		"uid":                   payrollReportSeedUserID,
	}).Execute()
	if err != nil {
		t.Fatalf("failed to insert committed expense %s: %v", params.id, err)
	}
}

type payrollTimeSheetInsertParams struct {
	id         string
	weekEnding string
}

func insertCommittedTimeSheetForPayrollReportTest(t *testing.T, app core.App, params payrollTimeSheetInsertParams) {
	t.Helper()

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
		"id":          params.id,
		"payroll_id":  payrollReportSeedPayrollID,
		"uid":         payrollReportSeedUserID,
		"week_ending": params.weekEnding,
	}).Execute()
	if err != nil {
		t.Fatalf("failed to insert committed timesheet %s: %v", params.id, err)
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
