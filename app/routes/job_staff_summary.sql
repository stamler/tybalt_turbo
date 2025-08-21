-- This result should be displayed within an ObjectTable 
-- that is itself displayed in a QueryBox. See original
-- Tybalt for an example.
WITH base AS (
  SELECT j.number,
    p.given_name,
  p.surname,
    SUM(te.hours) AS hours,
    SUM(ap.default_charge_out_rate * te.hours) AS value,
    SUM(SUM(ap.default_charge_out_rate * te.hours)) over() AS total,
    SUM(te.meals_hours) AS meals_hours,
  te.uid uid
  FROM time_entries te
  LEFT JOIN jobs j ON te.job = j.id 
  LEFT JOIN admin_profiles ap ON te.uid = ap.uid 
  LEFT JOIN profiles p ON te.uid = p.uid
  WHERE job = {:job_id}
  AND date >= {:start_date}
  AND date <= {:end_date}
  GROUP BY job, te.uid
)
SELECT *, ROUND((value * 100 / total),1) AS percent
FROM base