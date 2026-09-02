WITH qualifying_time AS (
  SELECT
    te.uid,
    te.branch,
    COALESCE(te.job, '') AS job,
    CAST(te.hours AS REAL) AS hours
  FROM time_entries te
  INNER JOIN time_types tt
    ON tt.id = te.time_type
  INNER JOIN branches entry_branch
    ON entry_branch.id = te.branch
  WHERE tt.code IN ('R', 'RT')
    AND te.date >= {:start_date}
    AND te.date <= {:end_date}
)
SELECT
  COALESCE(ap.payroll_id, '') AS payrollId,
  COALESCE(p.surname, '') AS surname,
  COALESCE(p.given_name, '') AS givenName,
  COALESCE(default_branch.name, '') AS defaultBranch,
  /* employee_branch_hours_columns */
  COALESCE(SUM(qt.hours), 0) AS total
FROM qualifying_time qt
INNER JOIN admin_profiles ap
  ON ap.uid = qt.uid
INNER JOIN profiles p
  ON p.uid = qt.uid
LEFT JOIN branches default_branch
  ON default_branch.id = ap.default_branch
GROUP BY
  qt.uid,
  ap.payroll_id,
  p.surname,
  p.given_name,
  default_branch.name
ORDER BY
  p.surname,
  p.given_name,
  LENGTH(ap.payroll_id),
  ap.payroll_id
