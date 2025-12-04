-- Highly optimized query for both "stale" and "unused" jobs whose numbers start with the provided prefix.
-- - When :age > 0 → returns stale jobs: last reference (across time_entries, time_amendments,
--   expenses, purchase_orders) is older than :age days.
-- - When :age <= 0 → returns unused jobs: no references across the above tables.
-- Parameters:
--   - prefix: arbitrary string, matched as j.number LIKE :prefix || '%'
--   - age: integer number of days; behavior depends on :age as described above
--   - limit: integer limit for number of rows returned
WITH matching_jobs AS (
  SELECT j.id, j.number, j.description, j.location, j.client, j.branch,
         j.outstanding_balance, j.outstanding_balance_date
  FROM jobs j
  WHERE j.status = 'Active'
    AND j.number LIKE {:prefix} || '%'
),
job_refs AS (
  SELECT job_id, ref_date, ref_type, type_rank
  FROM (
    SELECT te.job AS job_id, te.date AS ref_date, 'time_entry' AS ref_type, 1 AS type_rank
    FROM time_entries te
    INNER JOIN matching_jobs mj ON mj.id = te.job
    UNION ALL
    SELECT ta.job, ta.date, 'time_amendment', 2
    FROM time_amendments ta
    INNER JOIN matching_jobs mj ON mj.id = ta.job
    UNION ALL
    SELECT e.job, e.date, 'expense', 3
    FROM expenses e
    INNER JOIN matching_jobs mj ON mj.id = e.job
    UNION ALL
    SELECT po.job, po.date, 'purchase_order', 4
    FROM purchase_orders po
    INNER JOIN matching_jobs mj ON mj.id = po.job
  ) refs
),
ranked_refs AS (
  SELECT
    job_id,
    ref_date,
    ref_type,
    ROW_NUMBER() OVER (PARTITION BY job_id ORDER BY ref_date DESC, type_rank ASC) AS rn
  FROM job_refs
),
last_refs AS (
  SELECT job_id, ref_date AS last_reference, ref_type AS last_reference_type
  FROM ranked_refs
  WHERE rn = 1
)
SELECT
  j.id,
  j.number,
  j.description,
  j.location AS location,
  j.client AS client_id,
  c.name AS client,
  COALESCE(b.code, '') AS branch,
  COALESCE(j.outstanding_balance, 0) AS outstanding_balance,
  COALESCE(j.outstanding_balance_date, '') AS outstanding_balance_date,
  COALESCE(lr.last_reference, '') AS last_reference,
  COALESCE(lr.last_reference_type, '') AS last_reference_type
FROM matching_jobs j
LEFT JOIN clients c ON c.id = j.client
LEFT JOIN branches b ON b.id = j.branch
LEFT JOIN last_refs lr ON lr.job_id = j.id
WHERE ({:age} <= 0 AND lr.last_reference IS NULL)
   OR ({:age} > 0 AND lr.last_reference IS NOT NULL AND lr.last_reference <= date('now', printf('-%d days', {:age})))
ORDER BY (CASE WHEN {:age} > 0 THEN lr.last_reference END) ASC, j.number ASC
LIMIT {:limit};


