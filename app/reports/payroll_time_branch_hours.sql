SELECT payrollId,
  branchName,
  -- salary is the payroll-row setting used by reports.go to cap total branch
  -- allocations at the user's work_week_hours. branchName only identifies where
  -- this row's hours came from; it cannot tell us whether the employee needs
  -- salary normalization.
  salary,
  -- workWeekHours is the payroll-row cap for salary branch allocations. Normal
  -- time entries use the committed time_sheets snapshot; amendments use their
  -- linked time_sheets snapshot when present and fall back to admin_profiles.
  workWeekHours,
  -- defaultBranchName is the branch reports.go reduces first when a salary
  -- employee's R/RT branch allocation exceeds workWeekHours. The employee can
  -- still have separate totals in several branchName buckets; the default only
  -- picks the first bucket to cap down.
  defaultBranchName,
  SUM(hours) AS totalHours
FROM (
  SELECT ap.payroll_id AS payrollId,
    COALESCE(b.name, 'Unassigned') AS branchName,
    ts.salary AS salary,
    ts.work_week_hours AS workWeekHours,
    db.name AS defaultBranchName,
    IFNULL(te.hours, 0) AS hours
  FROM time_entries te
  LEFT JOIN time_sheets ts ON te.tsid = ts.id
  LEFT JOIN admin_profiles ap ON ap.uid = te.uid
  LEFT JOIN branches b ON te.branch = b.id
  LEFT JOIN branches db ON ap.default_branch = db.id
  LEFT JOIN time_types tt ON te.time_type = tt.id
  WHERE te.week_ending = {:weekEnding}
  AND te.tsid != ''
  AND ts.committed != ''
  AND tt.code IN ('R', 'RT')
  AND NOT ({:placeholder_payroll_id_condition})

  UNION ALL

  SELECT ap.payroll_id AS payrollId,
    COALESCE(b.name, 'Unassigned') AS branchName,
    COALESCE(ts.salary, ap.salary) AS salary,
    COALESCE(ts.work_week_hours, ap.work_week_hours) AS workWeekHours,
    db.name AS defaultBranchName,
    IFNULL(ta.hours, 0) AS hours
  FROM time_amendments ta
  LEFT JOIN time_sheets ts ON ta.tsid = ts.id
  LEFT JOIN admin_profiles ap ON ap.uid = ta.uid
  LEFT JOIN branches b ON ta.branch = b.id
  LEFT JOIN branches db ON ap.default_branch = db.id
  LEFT JOIN time_types tt ON ta.time_type = tt.id
  WHERE ta.committed_week_ending = {:weekEnding}
  AND ta.committed != ''
  AND tt.code IN ('R', 'RT')
  AND NOT ({:placeholder_payroll_id_condition})
)
GROUP BY payrollId, branchName, salary, workWeekHours, defaultBranchName
ORDER BY LENGTH(payrollId), payrollId, branchName
