-- Aggregate jobs with their client name and categories in a single query.
-- This avoids PocketBase's N+1 problem when using expand on back-references.
-- :id is bound from the route; when empty we return all jobs, otherwise the specific job.
-- {:id} is bound from the route; when empty we return all jobs, otherwise the specific job.
SELECT
  j.id,
  j.number,
  j.description,
  j.client AS client_id,
  c.name AS client
FROM jobs j
LEFT JOIN clients c ON c.id = j.client
WHERE ({:id} IS NULL OR {:id} = '' OR j.id = {:id})
ORDER BY j.number DESC; 