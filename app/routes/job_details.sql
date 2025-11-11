-- Aggregate a single job with related entities to avoid PocketBase N+1 queries.
-- :id must be provided and selects one job.
SELECT
  j.id,
  j.number,
  j.description,
  j.status,
  j.parent          AS parent_id,
  pa.number         AS parent_number,
  j.location         AS location,
  j.authorizing_document AS authorizing_document,
  j.client_po        AS client_po,
  j.client_reference_number AS client_reference_number,
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
  jo.name            AS job_owner_name,
  j.proposal         AS proposal_id,
  pr.number          AS proposal_number,
  j.fn_agreement     AS fn_agreement,
  j.project_award_date AS project_award_date,
  j.proposal_opening_date AS proposal_opening_date,
  j.proposal_submission_due_date AS proposal_submission_due_date,
  j.outstanding_balance AS outstanding_balance,
  j.outstanding_balance_date AS outstanding_balance_date,
  j.branch           AS branch_id,
  br.code            AS branch_code,
  br.name            AS branch_name,
  (
    SELECT COALESCE(json_group_array(json_object('id', pj.id, 'number', pj.number)), '[]')
    FROM jobs pj
    WHERE pj.proposal = j.id
  ) AS projects_json,
  (
    SELECT COALESCE(json_group_array(json_object('id', cj.id, 'number', cj.number)), '[]')
    FROM jobs cj
    WHERE cj.parent = j.id
  ) AS children_json,
  (
    SELECT COALESCE(json_group_array(json_object('id', c.id, 'name', c.name)), '[]')
    FROM categories c
    WHERE c.job = j.id
  ) AS categories_json,
  COALESCE(
    (
      SELECT json_group_array(
        json_object(
          'id', jta.id,
          'division', json_object('id', d.id, 'code', d.code, 'name', d.name),
          'hours', COALESCE(jta.hours, 0)
        )
      )
      FROM job_time_allocations jta
      JOIN divisions d ON d.id = jta.division
      WHERE jta.job = j.id
    ),
    '[]'
  ) AS allocations_json
FROM jobs j
LEFT JOIN jobs pr           ON pr.id = j.proposal
LEFT JOIN jobs pa           ON pa.id = j.parent
LEFT JOIN clients cli          ON cli.id = j.client
LEFT JOIN client_contacts cc   ON cc.id  = j.contact
LEFT JOIN managers m           ON m.id   = j.manager
LEFT JOIN managers am          ON am.id  = j.alternate_manager
LEFT JOIN clients jo           ON jo.id  = j.job_owner
LEFT JOIN branches br          ON br.id  = j.branch
WHERE j.id = {:id}; 