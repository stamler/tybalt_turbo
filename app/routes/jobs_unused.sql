-- Return zero-use jobs whose numbers start with the provided prefix.
-- A zero-use job has no references in time_entries, time_amendments,
-- expenses, or purchase_orders.
-- Parameters:
--   - prefix: arbitrary string, matched as j.number LIKE :prefix || '%'
--   - limit: integer limit for number of rows returned
SELECT
  j.id,
  j.number,
  j.description,
  j.location AS location,
  j.client AS client_id,
  c.name AS client,
  COALESCE(j.outstanding_balance, 0) AS outstanding_balance,
  COALESCE(j.outstanding_balance_date, '') AS outstanding_balance_date
FROM jobs j
LEFT JOIN clients c ON c.id = j.client
WHERE j.number LIKE {:prefix} || '%'
  AND j.status = 'Active'
  AND NOT EXISTS (SELECT 1 FROM time_entries te WHERE te.job = j.id)
  AND NOT EXISTS (SELECT 1 FROM time_amendments ta WHERE ta.job = j.id)
  AND NOT EXISTS (SELECT 1 FROM expenses e WHERE e.job = j.id)
  AND NOT EXISTS (SELECT 1 FROM purchase_orders po WHERE po.job = j.id)
ORDER BY j.number ASC
LIMIT {:limit};


