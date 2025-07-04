-- count total time entries for a job with the same filters as the main query
SELECT COUNT(*)
FROM   time_entries  te
LEFT   JOIN time_sheets ts ON te.tsid = ts.id
WHERE  ts.committed != ''
  AND  te.hours > 0
  AND  te.job = {:id}
  AND  ({:division}  IS NULL OR {:division}  = '' OR te.division  = {:division})
  AND  ({:time_type} IS NULL OR {:time_type} = '' OR te.time_type = {:time_type})
  AND  ({:uid}       IS NULL OR {:uid}       = '' OR te.uid       = {:uid})
  AND  ({:category}  IS NULL OR {:category}  = '' OR te.category  = {:category});