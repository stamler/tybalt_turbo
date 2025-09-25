SELECT
  c.id,
  c.name,
  COALESCE(c.business_development_lead, '')        AS business_development_lead,
  COALESCE(j_sum.total_outstanding_balance, 0)     AS outstanding_balance,
  COALESCE(j_sum.latest_outstanding_balance_date, '') AS outstanding_balance_date,
  COALESCE(p.given_name, '')                       AS lead_given_name,
  COALESCE(p.surname, '')                          AS lead_surname,
  COALESCE(lead.email, '')                         AS lead_email,
  COALESCE(cc.contacts_json, '[]')                 AS contacts_json,
  COALESCE(j_count.referencing_jobs_count, 0)      AS referencing_jobs_count
FROM clients c
LEFT JOIN profiles p ON p.uid = c.business_development_lead
LEFT JOIN users lead ON lead.id = c.business_development_lead
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
) cc ON cc.client = c.id
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
LEFT JOIN (
  SELECT
    client,
    SUM(outstanding_balance) AS total_outstanding_balance,
    MAX(outstanding_balance_date) AS latest_outstanding_balance_date
  FROM jobs
  WHERE client IS NOT NULL AND client != ''
  GROUP BY client
) j_sum ON j_sum.client = c.id
WHERE c.id = {:id};

