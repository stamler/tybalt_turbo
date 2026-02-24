-- Aggregate jobs with their client name and categories in a single query.
-- This avoids PocketBase's N+1 problem when using expand on back-references.
-- :id is bound from the route; when empty we return all jobs, otherwise the specific job.
-- {:id} is bound from the route; when empty we return all jobs, otherwise the specific job.
SELECT
  j.id,
  j.number,
  j.description,
  j.status AS status,
  j._imported AS imported,
  j.location AS location,
  j.client AS client_id,
  c.name AS client,
  COALESCE(b.code, '') AS branch,
  COALESCE(m.given_name || ' ' || m.surname, '') AS manager,
  COALESCE(j.outstanding_balance, 0) AS outstanding_balance,
  COALESCE(j.outstanding_balance_date, '') AS outstanding_balance_date
FROM jobs j
LEFT JOIN clients c ON c.id = j.client
LEFT JOIN branches b ON b.id = j.branch
LEFT JOIN profiles m ON m.uid = j.manager
WHERE ({:id} IS NULL OR {:id} = '' OR j.id = {:id})
ORDER BY j.number DESC; 
