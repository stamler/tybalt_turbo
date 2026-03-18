package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1013075334")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`{
			"fields": [
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text3208210256",
					"max": 0,
					"min": 0,
					"name": "id",
					"pattern": "^[a-z0-9]+$",
					"presentable": false,
					"primaryKey": true,
					"required": true,
					"system": true,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_Asnq",
					"max": 0,
					"min": 0,
					"name": "week_ending",
					"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
					"presentable": true,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "number1324909552",
					"max": null,
					"min": null,
					"name": "committed_timesheet_count",
					"onlyInt": true,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "number3557218540",
					"max": null,
					"min": null,
					"name": "committed_expense_count",
					"onlyInt": true,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				}
			],
			"viewQuery": "WITH payroll_periods AS (\n  SELECT\n    MIN(id) AS id,\n    week_ending\n  FROM time_sheets\n  WHERE committed != ''\n    AND (CAST(JULIANDAY(week_ending) - JULIANDAY('2025-03-01') AS INTEGER)) % 14 = 0\n  GROUP BY week_ending\n),\ntimesheet_counts AS (\n  SELECT\n    CASE\n      WHEN (CAST(JULIANDAY(week_ending) - JULIANDAY('2025-03-01') AS INTEGER)) % 14 = 0\n        THEN week_ending\n      ELSE date(week_ending, '+7 days')\n    END AS payroll_week_ending,\n    COUNT(*) AS committed_timesheet_count\n  FROM time_sheets\n  WHERE committed != ''\n  GROUP BY 1\n),\nexpense_counts AS (\n  SELECT\n    pay_period_ending AS payroll_week_ending,\n    COUNT(*) AS committed_expense_count\n  FROM expenses\n  WHERE committed != ''\n    AND pay_period_ending != ''\n  GROUP BY pay_period_ending\n)\nSELECT\n  p.id,\n  p.week_ending,\n  COALESCE(t.committed_timesheet_count, 0) AS committed_timesheet_count,\n  COALESCE(e.committed_expense_count, 0) AS committed_expense_count\nFROM payroll_periods p\nLEFT JOIN timesheet_counts t\n  ON t.payroll_week_ending = p.week_ending\nLEFT JOIN expense_counts e\n  ON e.payroll_week_ending = p.week_ending\nORDER BY p.week_ending DESC;"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1013075334")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`{
			"fields": [
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text3208210256",
					"max": 0,
					"min": 0,
					"name": "id",
					"pattern": "^[a-z0-9]+$",
					"presentable": false,
					"primaryKey": true,
					"required": true,
					"system": true,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_Asnq",
					"max": 0,
					"min": 0,
					"name": "week_ending",
					"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
					"presentable": true,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				}
			],
			"viewQuery": "SELECT\n  MIN(id) as id,\n  week_ending\nFROM time_sheets\nWHERE committed != ''\n  AND (CAST(JULIANDAY(week_ending) - JULIANDAY('2025-03-01') AS INTEGER)) % 14 = 0\nGROUP BY week_ending\nORDER BY week_ending DESC;"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
