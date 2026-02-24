-- Return the latest proposals and latest projects as a single list.
-- We select up to :limit from each group and label them via group_name,
-- ensuring proposals appear first, then projects, each ordered by created DESC.
WITH latest_proposals AS (
  SELECT
    j.id,
    j.number,
    j.description,
    j.location AS location,
    j.client AS client_id,
    j.branch AS branch_id,
    j.manager AS manager_id,
    COALESCE(j.outstanding_balance, 0) AS outstanding_balance,
    COALESCE(j.outstanding_balance_date, '') AS outstanding_balance_date,
    j.created,
    1 AS sort_group
  FROM jobs j
  WHERE j.number LIKE 'P%'
  ORDER BY j.created DESC
  LIMIT {:limit}
),
latest_projects AS (
  SELECT
    j.id,
    j.number,
    j.description,
    j.location AS location,
    j.client AS client_id,
    j.branch AS branch_id,
    j.manager AS manager_id,
    COALESCE(j.outstanding_balance, 0) AS outstanding_balance,
    COALESCE(j.outstanding_balance_date, '') AS outstanding_balance_date,
    j.created,
    2 AS sort_group
  FROM jobs j
  WHERE j.number NOT LIKE 'P%'
  ORDER BY j.created DESC
  LIMIT {:limit}
),
unioned AS (
  SELECT * FROM latest_proposals
  UNION ALL
  SELECT * FROM latest_projects
)
SELECT
  u.id,
  u.number,
  u.description,
  u.location,
  u.client_id,
  c.name AS client,
  COALESCE(b.code, '') AS branch,
  COALESCE(m.given_name || ' ' || m.surname, '') AS manager,
  u.outstanding_balance,
  u.outstanding_balance_date,
  CASE WHEN u.sort_group = 1 THEN 'Proposals' ELSE 'Projects' END AS group_name,
  u.sort_group,
  u.created
FROM unioned u
LEFT JOIN clients c ON c.id = u.client_id
LEFT JOIN branches b ON b.id = u.branch_id
LEFT JOIN profiles m ON m.uid = u.manager_id
ORDER BY u.sort_group ASC, u.created DESC;


