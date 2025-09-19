-- Aggregate clients with their contacts and job counts using LEFT JOINs to avoid N+1 queries
SELECT
  c.id,
  c.name,
  COALESCE(cc_agg.contacts_json, '[]') AS contacts_json,
  COALESCE(j_count.referencing_jobs_count, 0) AS referencing_jobs_count
FROM clients c
LEFT JOIN (
  SELECT 
    client,
    json_group_array(
      json_object(
        'id', id,
        'given_name', given_name,
        'surname', surname,
        'email', email
      )
    ) AS contacts_json
  FROM client_contacts
  GROUP BY client
) cc_agg ON cc_agg.client = c.id
LEFT JOIN (
  SELECT 
    client,
    COUNT(*) AS referencing_jobs_count
  FROM (
    SELECT id AS job_id, client AS client FROM jobs WHERE client IS NOT NULL AND client != ''
    UNION
    SELECT id AS job_id, job_owner AS client FROM jobs WHERE job_owner IS NOT NULL AND job_owner != ''
  ) j
  GROUP BY client
) j_count ON j_count.client = c.id
WHERE ({:id} IS NULL OR {:id} = '' OR c.id = {:id})
ORDER BY c.name; 