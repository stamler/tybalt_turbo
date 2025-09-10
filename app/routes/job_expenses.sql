-- get expenses for a job, ordered by date descending. Accepts optional filters
-- by division id, payment_type, uid, and category id similar to job_time_entries.sql
SELECT e.description,
       e.total,
       e.id,
       e.date,
       e.committed_week_ending,
       b.code AS branch_code,
       d.code AS division_code,
       e.payment_type AS payment_type,
       p.surname AS surname,
       p.given_name AS given_name,
       COALESCE(c.name, 'No Category') AS category_name
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
  )
ORDER BY e.date DESC
LIMIT {:limit} OFFSET {:offset}; 