-- WIP, built somewhat from scratch against SQLite.
-- The payroll report for a specific week
-- TODO: fold in time_amendments

SELECT payrollId,
  weekEnding,
  surname,
  givenName,
  name,
  manager,
  SUM(meals) AS meals,
  SUM(daysOffRotation) "days off rotation",
  SUM(hoursWorked) "hours worked",
  CASE
    WHEN salary = 1 AND SUM(hoursWorked) > 44 THEN SUM(hoursWorked) - 44
    ELSE 0
  END AS salaryHoursOver44,
  CASE
    WHEN salary = 1 THEN CASE
      WHEN SUM(hoursWorked) + IFNULL(SUM(Stat),0) + IFNULL(SUM(Bereavement),0) > workWeekHours THEN workWeekHours - IFNULL(SUM(Stat),0) - IFNULL(SUM(Bereavement),0)
      ELSE SUM(hoursWorked)
    END
    ELSE CASE
      WHEN SUM(hoursWorked) > 44 THEN 44
      ELSE SUM(hoursWorked)
    END
  END AS adjustedHoursWorked,
  CASE
    WHEN salary = 0 AND SUM(hoursWorked) > 44 THEN SUM(hoursWorked) - 44
    ELSE 0
  END AS "total overtime hours",
  CASE
    WHEN salary = 0 THEN CASE
      WHEN SUM(overtimeHoursToBank) > 0 THEN (CASE WHEN SUM(hoursWorked) > 44 THEN SUM(hoursWorked) - 44 ELSE 0 END) - SUM(overtimeHoursToBank)
      ELSE (CASE WHEN SUM(hoursWorked) > 44 THEN SUM(hoursWorked) - 44 ELSE 0 END)
    END
    ELSE 0
  END "overtime hours to pay",
  SUM(Bereavement) Bereavement,
  SUM(Stat) "Stat Holiday",
  SUM(PPTO) PPTO,
  SUM(Sick) Sick,
  SUM(Vacation) Vacation,
  SUM(overtimeHoursToBank) "overtime hours to bank",
  SUM(overtimePayoutRequested) "Overtime Payout Requested",
  MAX(hasAmendmentsForWeeksEnding) hasAmendmentsForWeeksEnding,
  CASE WHEN salary = 1 THEN "TRUE" ELSE "FALSE" END salary
FROM (
  SELECT ap.payroll_id payrollId,
    strftime('%Y', te.week_ending) || ' ' ||
    substr('  JanFebMarAprMayJunJulAugSepOctNovDec', strftime('%m', te.week_ending) * 3, 3) || ' ' ||
    strftime('%d', te.week_ending) AS weekEnding,
    p.surname surname,
    p.given_name givenName,
    p.given_name || ' ' ||  p. surname name,
    pm.given_name || ' ' || pm.surname manager,
    te.meals_hours meals,
    CASE WHEN tt.code = "OR" THEN 1 ELSE 0 END daysOffRotation,
    CASE WHEN tt.code IN ("R", "RT") THEN IFNULL(hours, 0) ELSE 0 END hoursWorked,
    CASE WHEN tt.code = "OB" THEN IFNULL(hours,0) ELSE 0 END Bereavement,
    CASE WHEN tt.code = "OH" THEN IFNULL(hours,0) ELSE 0 END Stat,
    CASE WHEN tt.code = "OP" THEN IFNULL(hours,0) ELSE 0 END PPTO,
    CASE WHEN tt.code = "OS" THEN IFNULL(hours,0) ELSE 0 END Sick,
    CASE WHEN tt.code = "OV" THEN IFNULL(hours,0) ELSE 0 END Vacation,
    CASE WHEN tt.code = "RB" THEN IFNULL(hours,0) ELSE 0 END overtimeHoursToBank,
    IFNULL(te.payout_request_amount,0) overtimePayoutRequested,
    NULL hasAmendmentsForWeeksEnding,
    ts.salary salary,
    ts.work_week_hours workWeekHours
  FROM time_entries te
  lEFT JOIN time_sheets ts ON te.tsid = ts.id
  LEFT JOIN admin_profiles ap ON ap.uid = te.uid 
  LEFT JOIN profiles p ON p.uid = te.uid
  LEFT JOIN profiles pm ON pm.uid = ts.approver
  LEFT JOIN time_types tt ON te.time_type = tt.id 
  WHERE te.week_ending = {:weekEnding}

  UNION ALL

  SELECT
    ap.payroll_id payrollId,
    strftime('%Y', ta.committed_week_ending) || ' ' ||
    substr('  JanFebMarAprMayJunJulAugSepOctNovDec', strftime('%m', ta.committed_week_ending) * 3, 3) || ' ' ||
    strftime('%d', ta.committed_week_ending) AS weekEnding, -- Amendments use commit week
    p.surname,
    p.given_name givenName,
    p.given_name || ' ' || p.surname AS name,
    p2.given_name || ' ' || p2.surname AS manager, -- Amendment committer is manager
    ta.meals_hours AS meals,
    CASE WHEN tt.code = "OR" THEN 1 ELSE 0 END daysOffRotation,
    CASE WHEN tt.code IN ("R", "RT") THEN IFNULL(ta.hours, 0) ELSE 0 END hoursWorked,
    CASE WHEN tt.code = "OB" THEN IFNULL(ta.hours,0) ELSE 0 END Bereavement,
    CASE WHEN tt.code = "OH" THEN IFNULL(ta.hours,0) ELSE 0 END Stat,
    CASE WHEN tt.code = "OP" THEN IFNULL(ta.hours,0) ELSE 0 END PPTO,
    CASE WHEN tt.code = "OS" THEN IFNULL(ta.hours,0) ELSE 0 END Sick,
    CASE WHEN tt.code = "OV" THEN IFNULL(ta.hours,0) ELSE 0 END Vacation,
    CASE WHEN tt.code = "RB" THEN IFNULL(ta.hours,0) ELSE 0 END overtimeHoursToBank,
    IFNULL(ta.payout_request_amount,0) overtimePayoutRequested,
    y.hasAmendmentsForWeeksEnding, -- Join the calculated list
    ap.salary,
    ap.work_week_hours
  FROM
    time_amendments ta
  LEFT JOIN admin_profiles ap ON ta.uid = ap.uid
  LEFT JOIN profiles p ON ta.uid = p.uid
  LEFT JOIN profiles p2 ON ta.committer = p2.uid
  LEFT JOIN time_types tt ON ta.time_type = tt.id 
  LEFT OUTER JOIN (
    -- Calculate the list of original weekEndings for amendments
    -- grouped by the week they were committed
    SELECT
      uid,
      committed_week_ending,
      json_group_array(week_ending) AS hasAmendmentsForWeeksEnding
    FROM (
      SELECT DISTINCT uid, committed_week_ending, week_ending
      FROM time_amendments
    ) DistinctTriplets
    GROUP BY uid, committed_week_ending
  ) y ON ta.uid = y.uid AND ta.committed_week_ending = y.committed_week_ending
  WHERE ta.committed_week_ending = {:weekEnding}
)
GROUP BY payrollId
ORDER BY LENGTH(payrollId), payrollId