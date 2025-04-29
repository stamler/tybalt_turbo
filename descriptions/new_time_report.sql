-- WIP, built somewhat from scratch against SQLite.
-- The payroll report for a specific week

SELECT payrollId,
  weekEnding,
  surname,
  givenName,
  name,
  manager,
  meals,
  daysOffRotation "days off rotation",
  hoursWorked "hours worked",
  CASE
      WHEN salary = 1 AND hoursWorked > 44 THEN hoursWorked - 44
      ELSE 0
  END AS salaryHoursOver44,
  CASE
      WHEN salary = 1 THEN CASE
          WHEN hoursWorked + IFNULL(Stat,0) + IFNULL(Bereavement,0) > workWeekHours THEN workWeekHours - IFNULL(Stat,0) - IFNULL(Bereavement,0)
          ELSE hoursWorked
      END
      ELSE CASE
          WHEN hoursWorked > 44 THEN 44
          ELSE hoursWorked
      END
  END AS adjustedHoursWorked,
  CASE
      WHEN salary = 0 AND hoursWorked > 44 THEN hoursWorked - 44
      ELSE 0
  END AS "total overtime hours",
  CASE
      WHEN salary = 0 THEN CASE
          WHEN overtimeHoursToBank > 0 THEN (CASE WHEN hoursWorked > 44 THEN hoursWorked - 44 ELSE 0 END) - overtimeHoursToBank
          ELSE (CASE WHEN hoursWorked > 44 THEN hoursWorked - 44 ELSE 0 END)
      END
      ELSE 0
  END "overtime hours to pay",
  Bereavement,
  Stat "Stat Holiday",
  PPTO,
  Sick,
  Vacation,
  overtimeHoursToBank "overtime hours to bank",
  overtimePayoutRequested "Overtime Payout Requested",
  "TBD" hasAmendmentsForWeeksEnding,
  CASE WHEN salary = 1 THEN "TRUE" ELSE "FALSE" END salary
FROM (
  
SELECT ap.payroll_id payrollId,
  te.week_ending  weekEnding,
  p.surname surname,
  p.given_name givenName,
  p.given_name || ' ' ||  p. surname name,
  pm.given_name || ' ' || pm.surname manager,
  SUM(te.meals_hours) AS meals,
  SUM(CASE WHEN tt.code = "OR" THEN 1 ELSE 0 END) daysOffRotation,
  SUM(CASE WHEN tt.code IN ("R", "RT") THEN IFNULL(hours, 0) ELSE 0 END) hoursWorked,
  SUM(CASE WHEN tt.code = "OB" THEN IFNULL(hours,0) ELSE 0 END) Bereavement,
  SUM(CASE WHEN tt.code = "OH" THEN IFNULL(hours,0) ELSE 0 END) Stat,
  SUM(CASE WHEN tt.code = "OP" THEN IFNULL(hours,0) ELSE 0 END) PPTO,
  SUM(CASE WHEN tt.code = "OS" THEN IFNULL(hours,0) ELSE 0 END) Sick,
  SUM(CASE WHEN tt.code = "OV" THEN IFNULL(hours,0) ELSE 0 END) Vacation,
  SUM(CASE WHEN tt.code = "RB" THEN IFNULL(hours,0) ELSE 0 END) overtimeHoursToBank,
  SUM(IFNULL(te.payout_request_amount,0)) overtimePayoutRequested,
  ts.salary salary,
  MAX(ts.work_week_hours) workWeekHours
FROM time_entries te
lEFT JOIN time_sheets ts ON te.tsid = ts.id
LEFT JOIN admin_profiles ap ON ap.uid = te.uid 
LEFT JOIN profiles p ON p.uid = te.uid
LEFT JOIN profiles pm ON pm.uid = ts.approver
LEFT JOIN time_types tt ON te.time_type = tt.id 
WHERE te.week_ending = '2025-04-19'
GROUP BY te.uid
)
ORDER BY LENGTH(payrollId), payrollId