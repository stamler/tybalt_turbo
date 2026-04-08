SELECT payrollId,
  branchName,
  CAST(SUM(hours) AS TEXT) AS totalHours
FROM (
  SELECT ap.payroll_id AS payrollId,
    COALESCE(b.name, 'Unassigned') AS branchName,
    IFNULL(te.hours, 0) AS hours
  FROM time_entries te
  LEFT JOIN time_sheets ts ON te.tsid = ts.id
  LEFT JOIN admin_profiles ap ON ap.uid = te.uid
  LEFT JOIN branches b ON te.branch = b.id
  WHERE te.week_ending = {:weekEnding}
  AND te.tsid != ''
  AND ts.committed != ''
  AND NOT ({:placeholder_payroll_id_condition})

  UNION ALL

  SELECT ap.payroll_id AS payrollId,
    COALESCE(b.name, 'Unassigned') AS branchName,
    IFNULL(ta.hours, 0) AS hours
  FROM time_amendments ta
  LEFT JOIN admin_profiles ap ON ap.uid = ta.uid
  LEFT JOIN branches b ON ta.branch = b.id
  WHERE ta.committed_week_ending = {:weekEnding}
  AND ta.committed != ''
  AND NOT ({:placeholder_payroll_id_condition})
)
GROUP BY payrollId, branchName
ORDER BY LENGTH(payrollId), payrollId, branchName
