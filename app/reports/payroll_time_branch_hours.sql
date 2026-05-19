SELECT payrollId,
  regularBranchName,
  reductionBranchName,
  -- salary is the payroll-row setting used by reports.go to cap total branch
  -- allocations at the user's payable regular hours. regularBranchName only
  -- identifies where regular hours came from; it cannot tell us whether the
  -- employee needs salary normalization.
  salary,
  -- workWeekHours is the payroll-row cap for salary branch allocations. Normal
  -- time entries use the committed time_sheets snapshot; amendments use their
  -- linked time_sheets snapshot when present and fall back to admin_profiles.
  workWeekHours,
  -- defaultBranchName is the branch reports.go reduces first when a salary
  -- employee's R/RT branch allocation exceeds the effective salary cap. The
  -- employee can still have separate totals in several branchName buckets; the
  -- default only picks the first bucket to cap down.
  defaultBranchName,
  SUM(regularHours) AS regularHours,
  SUM(bankedHours) AS bankedHours,
  SUM(statHours) AS statHours,
  SUM(bereavementHours) AS bereavementHours
FROM (
  SELECT ap.payroll_id AS payrollId,
    CASE WHEN tt.code IN ('R', 'RT') THEN COALESCE(b.name, 'Unassigned') ELSE NULL END AS regularBranchName,
    CASE WHEN tt.code = 'RB' AND b.name IS NOT NULL THEN b.name ELSE NULL END AS reductionBranchName,
    ts.salary AS salary,
    ts.work_week_hours AS workWeekHours,
    db.name AS defaultBranchName,
    CASE WHEN tt.code IN ('R', 'RT') THEN IFNULL(te.hours, 0) ELSE 0 END AS regularHours,
    CASE WHEN tt.code = 'RB' THEN IFNULL(te.hours, 0) ELSE 0 END AS bankedHours,
    CASE WHEN tt.code = 'OH' THEN IFNULL(te.hours, 0) ELSE 0 END AS statHours,
    CASE WHEN tt.code = 'OB' THEN IFNULL(te.hours, 0) ELSE 0 END AS bereavementHours
  FROM time_entries te
  LEFT JOIN time_sheets ts ON te.tsid = ts.id
  LEFT JOIN admin_profiles ap ON ap.uid = te.uid
  LEFT JOIN branches b ON te.branch = b.id
  LEFT JOIN branches db ON ap.default_branch = db.id
  LEFT JOIN time_types tt ON te.time_type = tt.id
  WHERE te.week_ending = {:weekEnding}
  AND te.tsid != ''
  AND ts.committed != ''
  AND tt.code IN ('R', 'RT', 'RB', 'OH', 'OB')
  AND NOT ({:placeholder_payroll_id_condition})

  UNION ALL

  SELECT ap.payroll_id AS payrollId,
    CASE WHEN tt.code IN ('R', 'RT') THEN COALESCE(b.name, 'Unassigned') ELSE NULL END AS regularBranchName,
    CASE WHEN tt.code = 'RB' AND b.name IS NOT NULL THEN b.name ELSE NULL END AS reductionBranchName,
    COALESCE(ts.salary, ap.salary) AS salary,
    COALESCE(ts.work_week_hours, ap.work_week_hours) AS workWeekHours,
    db.name AS defaultBranchName,
    CASE WHEN tt.code IN ('R', 'RT') THEN IFNULL(ta.hours, 0) ELSE 0 END AS regularHours,
    CASE WHEN tt.code = 'RB' THEN IFNULL(ta.hours, 0) ELSE 0 END AS bankedHours,
    CASE WHEN tt.code = 'OH' THEN IFNULL(ta.hours, 0) ELSE 0 END AS statHours,
    CASE WHEN tt.code = 'OB' THEN IFNULL(ta.hours, 0) ELSE 0 END AS bereavementHours
  FROM time_amendments ta
  LEFT JOIN time_sheets ts ON ta.tsid = ts.id
  LEFT JOIN admin_profiles ap ON ap.uid = ta.uid
  LEFT JOIN branches b ON ta.branch = b.id
  LEFT JOIN branches db ON ap.default_branch = db.id
  LEFT JOIN time_types tt ON ta.time_type = tt.id
  WHERE ta.committed_week_ending = {:weekEnding}
  AND ta.committed != ''
  AND tt.code IN ('R', 'RT', 'RB', 'OH', 'OB')
  AND NOT ({:placeholder_payroll_id_condition})
)
GROUP BY payrollId, regularBranchName, reductionBranchName, salary, workWeekHours, defaultBranchName
ORDER BY LENGTH(payrollId), payrollId, regularBranchName, reductionBranchName
