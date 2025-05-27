WITH base AS (
SELECT COALESCE(c.name, {:company_short_name}) client,
  COALESCE(j.number, '') job,
  COALESCE(d.code, '') division,
  tt.code timetype,
  CAST(SUBSTRING(te.date, 9, 2) AS INTEGER) date,
  substr('  JanFebMarAprMayJunJulAugSepOctNovDec', strftime('%m', te.date) * 3, 3) month,
  CAST(SUBSTRING(te.date, 1, 4) AS INTEGER) year,
  CASE WHEN te.job != '' THEN te.hours ELSE 0 END qty,
  'hours' unit,
  CASE WHEN te.job = '' THEN te.hours ELSE 0 END nc,
  te.meals_hours meals,
  te.work_record ref,
  COALESCE(j.description, '') project,
  te.description description,
  '' comments,
  (p.given_name || ' ' || p.surname) employee,
  p.surname surname,
  p.given_name givenName,
  te.amended amended
FROM (
  SELECT te_int.uid,
  te_int.job,
  te_int.division,
  te_int.time_type,
  te_int.date,
  te_int.hours,
  te_int.meals_hours,
  te_int.work_record,
  te_int.description,
  '' amended
  FROM time_entries te_int
  LEFT JOIN time_sheets ts ON te_int.tsid = ts.id
  WHERE tsid != ''
    AND ts.committed != ''
    AND te_int.{:date_column} = {:date_column_value}

  UNION ALL
  SELECT ta.uid,
  ta.job,
  ta.division,
  ta.time_type,
  ta.date,
  ta.hours,
  ta.meals_hours,
  ta.work_record,
  ta.description,
  'True' amended
  FROM time_amendments ta
  WHERE ta.committed != ''
    AND ta.{:date_column} = {:date_column_value}
) te
LEFT JOIN jobs j ON te.job = j.id
LEFT JOIN clients c ON j.client = c.id
LEFT JOIN divisions d ON te.division = d.id
LEFT JOIN time_types tt ON te.time_type = tt.id
LEFT JOIN profiles p ON te.uid = p.uid
)
SELECT * FROM base
ORDER BY date, timetype, job, division, qty, nc, surname, givenName