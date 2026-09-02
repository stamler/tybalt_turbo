# KPI Employee Branch Hours Report

## Purpose

This read-only management report shows regular hours by employee and branch. It
separates hours assigned to a job from hours that have no job.

## Access

- The report requires the `kpi` claim.
- The KPI navigation item is under Reports.
- A `report` or `admin` claim does not grant KPI access by itself.
- Production claim assignment is a separate manual task.

## Pages

- `/reports/kpi` lists the available KPI reports.
- `/reports/kpi/employee-branch-hours` displays this report.
- The default period is the complete previous calendar month.
- Start and end dates are inclusive.
- Selecting dates does not run the report. The user must select Update.
- The table is sortable and has a CSV download action.

## API

```text
GET /api/kpi/reports/employee_branch_hours
```

Parameters:

- `start_date`: required date in `YYYY-MM-DD` format.
- `end_date`: required date in `YYYY-MM-DD` format.
- `format`: optional `json` or `csv`. The default is `json`.

The API accepts only authenticated users who have the `kpi` claim.

## Data Rules

- The source is `time_entries`.
- The report includes only `R` and `RT` time types.
- The report does not include `time_amendments`.
- The report includes an employee when they have at least one qualifying entry
  in the selected period and the entry has a valid branch.
- The employee's current active status does not affect the result.
- Employees with no qualifying hours do not get a row.
- The entry branch selects the branch-hours column.
- `defaultBranch` is the name of `admin_profiles.default_branch`.
- A branch column contains entries where `job != ''`.
- The corresponding `NoJob` column contains entries where `job = ''`.
- `total` is the sum of all displayed branch and no-job columns.
- The SQL includes all matching entries. It does not require a committed time
  sheet.

## Dynamic Branch Columns

SQLite cannot bind column names. The server reads the ordered branch list and
builds only the repeated SQL projection expressions. Branch IDs, start date,
and end date remain bound parameters. All filtering, grouping, and hour totals
are calculated in `employee_branch_hours.sql`.

The `idx_time_entries_time_type_date` index supports the report's time-type and
inclusive date-range filter.
