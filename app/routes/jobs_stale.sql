-- Return stale jobs whose numbers start with the provided prefix.
-- A stale job has at least one reference and the most recent reference
-- (across time_entries, time_amendments, expenses, purchase_orders)
-- is older than the provided :age in days.
-- Parameters:
--   - prefix: arbitrary string, matched as j.number LIKE :prefix || '%'
--   - age: integer number of days; last reference must be <= now - age days
--   - limit: integer limit for number of rows returned
WITH refs AS (
  SELECT te.job AS job_id, te.date AS ref_date, 'time_entry' AS ref_type FROM time_entries te
  UNION ALL
  SELECT ta.job, ta.date, 'time_amendment' FROM time_amendments ta
  UNION ALL
  SELECT e.job, e.date, 'expense' FROM expenses e
  UNION ALL
  SELECT po.job, po.date, 'purchase_order' FROM purchase_orders po
),
last_refs AS (
  SELECT job_id, MAX(ref_date) AS last_reference
  FROM refs
  WHERE job_id != ''
  GROUP BY job_id
),
typed_last AS (
  SELECT
    lr.job_id,
    lr.last_reference,
    MIN(
      CASE r.ref_type
        WHEN 'time_entry' THEN 1
        WHEN 'time_amendment' THEN 2
        WHEN 'expense' THEN 3
        WHEN 'purchase_order' THEN 4
        ELSE 99
      END
    ) AS type_rank
  FROM last_refs lr
  JOIN refs r ON r.job_id = lr.job_id AND r.ref_date = lr.last_reference
  GROUP BY lr.job_id, lr.last_reference
)
SELECT
  j.id,
  j.number,
  j.description,
  j.location AS location,
  j.client AS client_id,
  c.name AS client,
  COALESCE(j.outstanding_balance, 0) AS outstanding_balance,
  COALESCE(j.outstanding_balance_date, '') AS outstanding_balance_date,
  COALESCE(lr.last_reference, '') AS last_reference,
  CASE typed.type_rank
    WHEN 1 THEN 'time_entry'
    WHEN 2 THEN 'time_amendment'
    WHEN 3 THEN 'expense'
    WHEN 4 THEN 'purchase_order'
    ELSE ''
  END AS last_reference_type
FROM jobs j
LEFT JOIN clients c ON c.id = j.client
LEFT JOIN last_refs lr ON lr.job_id = j.id
LEFT JOIN typed_last typed ON typed.job_id = j.id AND typed.last_reference = lr.last_reference
WHERE j.status = 'Active'
  AND j.number LIKE {:prefix} || '%'
  AND (
    ({:age} <= 0 AND lr.last_reference IS NULL)
    OR
    ({:age} > 0 AND lr.last_reference <= date('now', printf('-%d days', {:age})))
  )
ORDER BY (CASE WHEN {:age} > 0 THEN lr.last_reference END) ASC, j.number ASC
LIMIT {:limit};


