-- top-level summary for expenses associated with a job. Accepts optional filters
-- by division id, payment_type, uid, and category id similar to job_time_summary.sql.
SELECT
  SUM(e.total)                   total_amount,
  MIN(e.date)                    earliest_expense,
  MAX(e.date)                    latest_expense,
  json_group_array(
    DISTINCT json_object('id', b.id, 'code', b.code)
  )                               branches,
  json_group_array(
    DISTINCT json_object('id', d.id, 'code', d.code)
  )                               divisions,
  json_group_array(
    DISTINCT json_object('name', e.payment_type)
  )                               payment_types,
  json_group_array(
    DISTINCT json_object('id', p.uid, 'name', p.given_name || ' ' || p.surname)
  )                               names,
  json_group_array(
    DISTINCT json_object('id', COALESCE(c.id, 'none'), 'name', COALESCE(c.name, 'No Category'))
  )                               categories
FROM   expenses e
LEFT   JOIN branches  b ON e.branch  = b.id
LEFT   JOIN divisions  d ON e.division = d.id
LEFT   JOIN profiles   p ON e.uid      = p.uid
LEFT   JOIN categories c ON e.category = c.id
WHERE  e.committed != ''
  AND  e.total > 0
  AND  e.job = {:id}
  AND  ({:branch}      IS NULL OR {:branch}      = '' OR e.branch      = {:branch})
  AND  ({:division}     IS NULL OR {:division}     = '' OR e.division     = {:division})
  AND  ({:payment_type} IS NULL OR {:payment_type} = '' OR e.payment_type = {:payment_type})
  AND  ({:uid}          IS NULL OR {:uid}          = '' OR e.uid          = {:uid})
  AND (
    ({:category} IS NULL OR {:category} = '')
    OR
    ({:category} = 'none' AND e.category = '')
    OR
    ({:category} != 'none' AND e.category = {:category})
  ); 