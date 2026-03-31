package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Migrations snapshot SQL at creation time so they remain stable even if the
// runtime placeholder helper changes later.
const placeholderPayrollIDSQLCondition = "(LENGTH(COALESCE(ap.payroll_id, '')) = 9 AND COALESCE(ap.payroll_id, '') GLOB '9[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]')"

var payrollReportWeekEndingsViewQueryWithPlaceholderPayrollIDCounts = `WITH raw_payroll_periods AS (
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
),
placeholder_time_rows AS (
  SELECT
    ts.week_ending AS source_week_ending
  FROM time_sheets ts
  LEFT JOIN admin_profiles ap ON ap.uid = ts.uid
  WHERE ts.committed != ''
    AND ` + placeholderPayrollIDSQLCondition + `

  UNION ALL

  SELECT
    ta.committed_week_ending AS source_week_ending
  FROM time_amendments ta
  LEFT JOIN admin_profiles ap ON ap.uid = ta.uid
  WHERE ta.committed != ''
    AND ta.committed_week_ending != ''
    AND ` + placeholderPayrollIDSQLCondition + `
),
placeholder_time_counts AS (
  SELECT
    CASE
      WHEN (CAST(JULIANDAY(source_week_ending) - JULIANDAY('2025-03-01') AS INTEGER)) % 14 = 0
        THEN source_week_ending
      ELSE date(source_week_ending, '+7 days')
    END AS payroll_week_ending,
    SUM(
      CASE
        WHEN (CAST(JULIANDAY(source_week_ending) - JULIANDAY('2025-03-01') AS INTEGER)) % 14 = 0
          THEN 0
        ELSE 1
      END
    ) AS placeholder_payroll_id_week1_time_count,
    SUM(
      CASE
        WHEN (CAST(JULIANDAY(source_week_ending) - JULIANDAY('2025-03-01') AS INTEGER)) % 14 = 0
          THEN 1
        ELSE 0
      END
    ) AS placeholder_payroll_id_week2_time_count
  FROM placeholder_time_rows
  GROUP BY 1
),
placeholder_expense_counts AS (
  SELECT
    e.pay_period_ending AS payroll_week_ending,
    COUNT(*) AS placeholder_payroll_id_expense_count
  FROM expenses e
  LEFT JOIN admin_profiles ap ON ap.uid = e.uid
  WHERE e.committed != ''
    AND e.pay_period_ending != ''
    AND (CAST(JULIANDAY(e.pay_period_ending) - JULIANDAY('2025-03-01') AS INTEGER)) % 14 = 0
    AND ` + placeholderPayrollIDSQLCondition + `
  GROUP BY e.pay_period_ending
)
SELECT
  p.id,
  p.week_ending,
  COALESCE(t.committed_timesheet_count, 0) AS committed_timesheet_count,
  COALESCE(e.committed_expense_count, 0) AS committed_expense_count,
  COALESCE(pt.placeholder_payroll_id_week1_time_count, 0) AS placeholder_payroll_id_week1_time_count,
  COALESCE(pt.placeholder_payroll_id_week2_time_count, 0) AS placeholder_payroll_id_week2_time_count,
  COALESCE(pe.placeholder_payroll_id_expense_count, 0) AS placeholder_payroll_id_expense_count
FROM payroll_periods p
LEFT JOIN timesheet_counts t
  ON t.payroll_week_ending = p.week_ending
LEFT JOIN expense_counts e
  ON e.payroll_week_ending = p.week_ending
LEFT JOIN placeholder_time_counts pt
  ON pt.payroll_week_ending = p.week_ending
LEFT JOIN placeholder_expense_counts pe
  ON pe.payroll_week_ending = p.week_ending
ORDER BY p.week_ending DESC;`

var timeTrackingViewQueryWithPlaceholderPayrollIDCounts = `WITH placeholder_expense_counts AS (
  SELECT
    e.committed_week_ending AS week_ending,
    COUNT(*) AS placeholder_payroll_id_expense_count
  FROM expenses e
  LEFT JOIN admin_profiles ap ON ap.uid = e.uid
  WHERE e.committed != ''
    AND e.committed_week_ending != ''
    AND ` + placeholderPayrollIDSQLCondition + `
  GROUP BY e.committed_week_ending
)
SELECT
  MIN(ts.id) id,
  ts.week_ending,
  SUM(CASE WHEN ts.committed != '' THEN 1 ELSE 0 END) committed_count,
  SUM(CASE WHEN ts.approved != '' AND ts.committed = '' AND ts.rejected = '' THEN 1 ELSE 0 END) approved_count,
  SUM(CASE WHEN ts.submitted = 1 AND ts.approved = '' AND ts.committed = '' AND ts.rejected = '' THEN 1 ELSE 0 END) submitted_count,
  COALESCE(MAX(pe.placeholder_payroll_id_expense_count), 0) AS placeholder_payroll_id_expense_count
FROM time_sheets ts
LEFT JOIN placeholder_expense_counts pe
  ON pe.week_ending = ts.week_ending
GROUP BY ts.week_ending
ORDER BY ts.week_ending DESC;`

func init() {
	m.Register(func(app core.App) error {
		payrollCollection, err := app.FindCollectionByNameOrId("pbc_1013075334")
		if err != nil {
			return err
		}

		if err := payrollCollection.Fields.AddMarshaledJSON([]byte(`{
			"hidden": false,
			"id": "number3351042101",
			"max": null,
			"min": null,
			"name": "placeholder_payroll_id_week1_time_count",
			"onlyInt": true,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}
		if err := payrollCollection.Fields.AddMarshaledJSON([]byte(`{
			"hidden": false,
			"id": "number4289971250",
			"max": null,
			"min": null,
			"name": "placeholder_payroll_id_week2_time_count",
			"onlyInt": true,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}
		if err := payrollCollection.Fields.AddMarshaledJSON([]byte(`{
			"hidden": false,
			"id": "number4091356012",
			"max": null,
			"min": null,
			"name": "placeholder_payroll_id_expense_count",
			"onlyInt": true,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		payrollCollection.ViewQuery = payrollReportWeekEndingsViewQueryWithPlaceholderPayrollIDCounts
		if err := app.Save(payrollCollection); err != nil {
			return err
		}

		timeTrackingCollection, err := app.FindCollectionByNameOrId("pbc_331583008")
		if err != nil {
			return err
		}

		if err := timeTrackingCollection.Fields.AddMarshaledJSON([]byte(`{
			"hidden": false,
			"id": "number3588826804",
			"max": null,
			"min": null,
			"name": "placeholder_payroll_id_expense_count",
			"onlyInt": true,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		timeTrackingCollection.ViewQuery = timeTrackingViewQueryWithPlaceholderPayrollIDCounts
		return app.Save(timeTrackingCollection)
	}, func(app core.App) error {
		payrollCollection, err := app.FindCollectionByNameOrId("pbc_1013075334")
		if err != nil {
			return err
		}

		payrollCollection.Fields.RemoveById("number3351042101")
		payrollCollection.Fields.RemoveById("number4289971250")
		payrollCollection.Fields.RemoveById("number4091356012")
		payrollCollection.ViewQuery = payrollReportWeekEndingsViewQueryWithExpenses
		if err := app.Save(payrollCollection); err != nil {
			return err
		}

		timeTrackingCollection, err := app.FindCollectionByNameOrId("pbc_331583008")
		if err != nil {
			return err
		}

		timeTrackingCollection.Fields.RemoveById("number3588826804")
		timeTrackingCollection.ViewQuery = "SELECT\n  MIN(id) id,\n  week_ending,\n  SUM(CASE WHEN committed != '' THEN 1 ELSE 0 END) committed_count,\n  SUM(CASE WHEN approved != '' AND committed = '' AND rejected = '' THEN 1 ELSE 0 END) approved_count,\n  SUM(CASE WHEN submitted = 1 AND approved = '' AND committed = '' AND rejected = '' THEN 1 ELSE 0 END) submitted_count\nFROM time_sheets\nGROUP BY week_ending\nORDER BY week_ending DESC; "
		return app.Save(timeTrackingCollection)
	})
}
