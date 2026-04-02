SELECT
  te.id AS id,
  te.work_record AS work_record,
  te.week_ending AS week_ending,
  te.uid AS uid,
  COALESCE(CAST(te.hours AS REAL), 0) AS hours,
  COALESCE(j.number, '') AS job_number,
  COALESCE(te.job, '') AS job_id,
  COALESCE(te.description, '') AS description,
  COALESCE(p.surname, '') AS surname,
  COALESCE(p.given_name, '') AS given_name,
  COALESCE(ts_exact.id, ts_fallback.id, '') AS timesheet_id
FROM time_entries te
LEFT JOIN jobs j
  ON te.job = j.id
LEFT JOIN profiles p
  ON te.uid = p.uid
LEFT JOIN time_sheets ts_exact
  ON ts_exact.id = te.tsid
LEFT JOIN time_sheets ts_fallback
  ON te.tsid = ''
 AND ts_fallback.uid = te.uid
 AND ts_fallback.week_ending = te.week_ending
WHERE te.work_record = {:work_record}
  AND te.work_record != ''
ORDER BY te.week_ending DESC
