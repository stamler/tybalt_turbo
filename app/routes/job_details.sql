-- Aggregate a single job with related entities to avoid PocketBase N+1 queries.
-- :id must be provided and selects one job.
SELECT
  j.id,
  j.number,
  j.description,
  j.status,
  j.client           AS client_id,
  cli.name           AS client_name,
  j.contact          AS contact_id,
  cc.given_name      AS contact_given_name,
  cc.surname         AS contact_surname,
  j.manager          AS manager_id,
  m.given_name       AS manager_given_name,
  m.surname          AS manager_surname,
  j.alternate_manager        AS alternate_manager_id,
  am.given_name      AS alternate_manager_given_name,
  am.surname         AS alternate_manager_surname,
  j.job_owner        AS job_owner_id,
  jo.given_name      AS job_owner_given_name,
  jo.surname         AS job_owner_surname,
  j.proposal         AS proposal_id,
  pr.number          AS proposal_number,
  j.fn_agreement     AS fn_agreement,
  j.project_award_date AS project_award_date,
  j.proposal_opening_date AS proposal_opening_date,
  j.proposal_submission_due_date AS proposal_submission_due_date,
  (
    SELECT COALESCE(json_group_array(json_object('id', pj.id, 'number', pj.number)), '[]')
    FROM jobs pj
    WHERE pj.proposal = j.id
  ) AS projects_json,
  COALESCE(
    (
      SELECT json_group_array(
        json_object(
          'id', d.id,
          'code', d.code,
          'name', d.name
        )
      )
      FROM divisions d
      WHERE d.id IN (SELECT value FROM json_each(j.divisions))
    ),
    '[]'
  ) AS divisions_json
FROM jobs j
LEFT JOIN jobs pr           ON pr.id = j.proposal
LEFT JOIN clients cli          ON cli.id = j.client
LEFT JOIN client_contacts cc   ON cc.id  = j.contact
LEFT JOIN managers m           ON m.id   = j.manager
LEFT JOIN managers am          ON am.id  = j.alternate_manager
LEFT JOIN managers jo          ON jo.id  = j.job_owner
WHERE j.id = {:id}; 