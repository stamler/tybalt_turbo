-- get time entries for a job, ordered by date descending. We will modify this
-- accept the option to filter by d.id, tt.id, p.uid, c.id as requested by the
-- caller
-- Branch, division, and profile can be missing, so blank their codes and
-- names; the Go row fields are plain strings and cannot hold NULL.
SELECT te.description,
	te.hours,
	te.id,
	te.work_record,
	te.date,
	te.week_ending,
	te.tsid,
	COALESCE(b.code, '') AS branch_code,
	COALESCE(d.code, '') AS division_code,
	COALESCE(tt.code, '') AS time_type_code,
	COALESCE(p.surname, '') AS surname,
	COALESCE(p.given_name, '') AS given_name,
	COALESCE(c.name, 'No Category') AS category_name
FROM time_entries te
LEFT JOIN branches b ON te.branch = b.id
LEFT JOIN divisions d ON te.division = d.id
LEFT JOIN time_types tt ON te.time_type = tt.id
LEFT JOIN profiles p ON te.uid = p.uid
LEFT JOIN categories c ON te.category = c.id
LEFT JOIN time_sheets ts ON te.tsid = ts.id
WHERE ts.committed != '' 
AND te.hours > 0
AND te.job = {:id}
AND ({:branch} IS NULL OR {:branch} = '' OR te.branch = {:branch})
AND ({:division} IS NULL OR {:division} = '' OR te.division = {:division})
AND ({:time_type} IS NULL OR {:time_type} = '' OR te.time_type = {:time_type})
AND ({:uid} IS NULL OR {:uid} = '' OR te.uid = {:uid})
AND (
  ({:category} IS NULL OR {:category} = '')
  OR
  ({:category} = 'none' AND te.category = '')
  OR
  ({:category} != 'none' AND te.category = {:category})
)
ORDER BY date DESC
LIMIT {:limit} OFFSET {:offset};