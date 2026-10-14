-- Aggregate clients with their contacts and job counts using LEFT JOINs to avoid N+1 queries.
-- referenced is true when any job, job request, invoicing profile or note uses
-- the client. The clients deleteRule and delete hook refuse to delete those, so
-- the UI can tell when a delete will be refused.
-- Each table is scanned once: json_each turns each job and job request into one
-- row for its client and one for its job owner. (A CASE over a cross join with
-- constant rows also scans once, but SQLite then joins its grouped result
-- without an index.)
SELECT
  c.id,
  c.name,
  COALESCE(c.alias, '') AS alias,
  COALESCE(cc_agg.contacts_json, '[]') AS contacts_json,
  COALESCE(j_use.referencing_jobs_count, 0) AS referencing_jobs_count,
  COALESCE(j_use.total_outstanding_balance, 0) AS outstanding_balance,
  (j_use.client IS NOT NULL OR uses.client IS NOT NULL) AS referenced
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
  -- The job count includes jobs the client owns, and the balance only jobs
  -- billed to it.
  SELECT
    u.value AS client,
    COUNT(DISTINCT j.id) AS referencing_jobs_count,
    SUM(CASE WHEN u.key = 0 THEN j.outstanding_balance END) AS total_outstanding_balance
  FROM jobs j
  JOIN json_each(json_array(j.client, j.job_owner)) u
  WHERE u.value != ''
  GROUP BY u.value
) j_use ON j_use.client = c.id
LEFT JOIN (
  SELECT u.value AS client
  FROM job_requests r
  JOIN json_each(json_array(r.client, r.job_owner)) u
  UNION
  SELECT client FROM client_invoicing_information
  UNION
  SELECT client FROM client_notes
) uses ON uses.client = c.id
WHERE ({:id} IS NULL OR {:id} = '' OR c.id = {:id})
ORDER BY c.name;
