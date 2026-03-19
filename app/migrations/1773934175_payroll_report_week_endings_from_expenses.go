package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const payrollReportWeekEndingsPreviousViewQuery = `WITH payroll_periods AS (
  SELECT
    MIN(id) AS id,
    week_ending
  FROM time_sheets
  WHERE committed != ''
    AND (CAST(JULIANDAY(week_ending) - JULIANDAY('2025-03-01') AS INTEGER)) % 14 = 0
  GROUP BY week_ending
),
timesheet_counts AS (
  SELECT
    CASE
      WHEN (CAST(JULIANDAY(week_ending) - JULIANDAY('2025-03-01') AS INTEGER)) % 14 = 0
        THEN week_ending
      ELSE date(week_ending, '+7 days')
    END AS payroll_week_ending,
    COUNT(*) AS committed_timesheet_count
  FROM time_sheets
  WHERE committed != ''
  GROUP BY 1
),
expense_counts AS (
  SELECT
    pay_period_ending AS payroll_week_ending,
    COUNT(*) AS committed_expense_count
  FROM expenses
  WHERE committed != ''
    AND pay_period_ending != ''
  GROUP BY pay_period_ending
)
SELECT
  p.id,
  p.week_ending,
  COALESCE(t.committed_timesheet_count, 0) AS committed_timesheet_count,
  COALESCE(e.committed_expense_count, 0) AS committed_expense_count
FROM payroll_periods p
LEFT JOIN timesheet_counts t
  ON t.payroll_week_ending = p.week_ending
LEFT JOIN expense_counts e
  ON e.payroll_week_ending = p.week_ending
ORDER BY p.week_ending DESC;`

const payrollReportWeekEndingsViewQueryWithExpenses = `WITH raw_payroll_periods AS (
  SELECT
    week_ending
  FROM time_sheets
  WHERE committed != ''
    AND (CAST(JULIANDAY(week_ending) - JULIANDAY('2025-03-01') AS INTEGER)) % 14 = 0
  GROUP BY week_ending

  UNION

  SELECT
    pay_period_ending AS week_ending
  FROM expenses
  WHERE committed != ''
    AND pay_period_ending != ''
    AND (CAST(JULIANDAY(pay_period_ending) - JULIANDAY('2025-03-01') AS INTEGER)) % 14 = 0
  GROUP BY pay_period_ending
),
payroll_periods AS (
  SELECT
    'p' || REPLACE(raw.week_ending, '-', '') AS id,
    raw.week_ending
  FROM raw_payroll_periods raw
),
timesheet_counts AS (
  SELECT
    CASE
      WHEN (CAST(JULIANDAY(week_ending) - JULIANDAY('2025-03-01') AS INTEGER)) % 14 = 0
        THEN week_ending
      ELSE date(week_ending, '+7 days')
    END AS payroll_week_ending,
    COUNT(*) AS committed_timesheet_count
  FROM time_sheets
  WHERE committed != ''
  GROUP BY 1
),
expense_counts AS (
  SELECT
    pay_period_ending AS payroll_week_ending,
    COUNT(*) AS committed_expense_count
  FROM expenses
  WHERE committed != ''
    AND pay_period_ending != ''
    AND (CAST(JULIANDAY(pay_period_ending) - JULIANDAY('2025-03-01') AS INTEGER)) % 14 = 0
  GROUP BY pay_period_ending
)
SELECT
  p.id,
  p.week_ending,
  COALESCE(t.committed_timesheet_count, 0) AS committed_timesheet_count,
  COALESCE(e.committed_expense_count, 0) AS committed_expense_count
FROM payroll_periods p
LEFT JOIN timesheet_counts t
  ON t.payroll_week_ending = p.week_ending
LEFT JOIN expense_counts e
  ON e.payroll_week_ending = p.week_ending
ORDER BY p.week_ending DESC;`

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1013075334")
		if err != nil {
			return err
		}

		collection.ViewQuery = payrollReportWeekEndingsViewQueryWithExpenses
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1013075334")
		if err != nil {
			return err
		}

		collection.ViewQuery = payrollReportWeekEndingsPreviousViewQuery
		return app.Save(collection)
	})
}
