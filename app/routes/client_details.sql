WITH client_profiles AS (
  SELECT * FROM client_invoicing_information WHERE client = {:id}
),
-- UNION removes duplicate use when a project and its invoice use one contact.
contact_job_use AS (
  SELECT j.contact AS contact, j.id AS job FROM jobs j
  JOIN client_contacts c ON c.id = j.contact AND c.client = {:id}
  UNION
  SELECT p.contact, j.id FROM jobs j
  JOIN client_profiles p ON p.id = j.invoicing_information
),
contacts AS (
  SELECT c.*,
    (SELECT COUNT(*) FROM contact_job_use u WHERE u.contact = c.id) AS job_count,
    (SELECT COUNT(*) FROM client_profiles p WHERE p.contact = c.id) AS profile_count
  FROM client_contacts c WHERE c.client = {:id}
  ORDER BY c.given_name COLLATE NOCASE, c.surname COLLATE NOCASE, c.id
),
invoicing_profiles AS (
  SELECT p.*,
    COALESCE(NULLIF(TRIM(COALESCE(creator.given_name, '') || ' ' || COALESCE(creator.surname, '')), ''), creator_user.email, '') AS creator_name,
    (SELECT COUNT(*) FROM jobs j WHERE j.invoicing_information = p.id) AS job_count
  FROM client_profiles p
  LEFT JOIN profiles creator ON creator.uid = p.creator
  LEFT JOIN users creator_user ON creator_user.id = p.creator
  ORDER BY p.name COLLATE NOCASE, p.id
)
SELECT
  c.id,
  c.name,
  c.address,
  c.city,
  c.province_state,
  c.postal_code,
  c.country,
  c.phone,
  COALESCE(c.business_development_lead, '') AS business_development_lead,
  COALESCE(j_sum.total_outstanding_balance, 0) AS outstanding_balance,
  COALESCE(j_sum.latest_outstanding_balance_date, '') AS outstanding_balance_date,
  COALESCE(p.given_name, '') AS lead_given_name,
  COALESCE(p.surname, '') AS lead_surname,
  COALESCE(lead.email, '') AS lead_email,
  (SELECT json_group_array(json_object(
    'id', id, 'client', client, 'given_name', given_name, 'surname', surname,
    'email', email, 'phone', phone, 'address', address, 'city', city,
    'province_state', province_state, 'postal_code', postal_code, 'country', country,
    'job_count', job_count, 'profile_count', profile_count
  )) FROM contacts) AS contacts_json,
  (SELECT json_group_array(json_object(
    'id', id, 'name', name, 'client', client, 'contact', contact,
    'fax', fax, 'invoicing_instructions', invoicing_instructions,
    'creator', creator, 'creator_name', creator_name, 'created', created,
    'job_count', job_count
  )) FROM invoicing_profiles) AS invoicing_profiles_json,
  COALESCE(j_count.referencing_jobs_count, 0) AS referencing_jobs_count
FROM clients c
LEFT JOIN profiles p ON p.uid = c.business_development_lead
LEFT JOIN users lead ON lead.id = c.business_development_lead
LEFT JOIN (
  SELECT client, COUNT(*) AS referencing_jobs_count
  FROM (
    SELECT id AS job_id, client FROM jobs WHERE client IS NOT NULL AND client != ''
    UNION
    SELECT id AS job_id, job_owner AS client FROM jobs WHERE job_owner IS NOT NULL AND job_owner != ''
  ) j
  GROUP BY client
) j_count ON j_count.client = c.id
LEFT JOIN (
  SELECT client,
    SUM(outstanding_balance) AS total_outstanding_balance,
    MAX(outstanding_balance_date) AS latest_outstanding_balance_date
  FROM jobs
  WHERE client IS NOT NULL AND client != ''
  GROUP BY client
) j_sum ON j_sum.client = c.id
WHERE c.id = {:id};
