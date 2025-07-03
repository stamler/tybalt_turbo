-- get time entries for a job, ordered by date descending. We will modify this
-- accept the option to filter by d.id, tt.id, p.uid, c.id as requested by the
-- caller
SELECT te.description,
	te.hours,
	te.id,
	te.work_record,
	te.date,
	te.week_ending,
	te.tsid,
	d.code AS division_code,
	tt.code AS time_type_code,
	p.surname AS surname,
	p.given_name AS given_name,
	c.name AS category_name
FROM time_entries te
LEFT JOIN divisions d ON te.division = d.id
LEFT JOIN time_types tt ON te.time_type = tt.id
LEFT JOIN profiles p ON te.uid = p.uid
LEFT JOIN categories c ON te.category = c.id
LEFT JOIN time_sheets ts ON te.tsid = ts.id
WHERE ts.committed != '' 
AND te. hours > 0
AND te.job = '84c3285e95a7ac0'
ORDER BY date DESC;

-- top-level summary. We will show this by default in the UI. We will modify
-- this to accept the option to filter by d.id, tt.id, p.uid, c.id as requested
-- by the caller. The caller can then click the division, time type, name, or
-- category to show a more specific summary, and then click another button to
-- show the individual entries from the above query.
SELECT
  SUM(te.hours) total_hours,
  MIN(te.date) earliest_entry,
  MAX(te.date) latest_entry,
  json_group_array(
    DISTINCT json_object('id', d.id, 'code', d.code)
  ) divisions,
  json_group_array(
    DISTINCT json_object('id', tt.id, 'code', tt.code)
  ) time_types,
  json_group_array(
    DISTINCT json_object('id', p.uid, 'name', p.given_name || ' ' || p.surname)
  ) names,
  json_group_array(
    DISTINCT json_object('id', c.id, 'name', c.name)
  ) categories
FROM time_entries te
LEFT JOIN divisions   d  ON te.division  = d.id
LEFT JOIN time_types  tt ON te.time_type = tt.id
LEFT JOIN profiles    p  ON te.uid       = p.uid
LEFT JOIN categories  c  ON te.category  = c.id
LEFT JOIN time_sheets ts ON te.tsid      = ts.id
WHERE ts.committed != ''
  AND te.hours > 0
  AND te.job = '84c3285e95a7ac0';