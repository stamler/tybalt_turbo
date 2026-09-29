WITH report_jobs AS (
  SELECT j.id, j.number, j.description, j.client, j.manager, j.branch,
    j.rate_sheet, j.project_value FROM jobs j
  WHERE j.status = 'Active' AND j.number NOT LIKE 'P%'
    AND ({:mode} != 'my' OR j.manager = {:uid})
    AND ({:mode} != 'branch' OR {:branch} = '' OR j.branch = {:branch})
    AND ({:mode} != 'division' OR (
      EXISTS (SELECT 1 FROM job_time_allocations a WHERE a.job = j.id AND a.division = {:division})
      AND ({:require_time} = 0 OR EXISTS (
        SELECT 1 FROM time_entries te
        WHERE te.job = j.id AND te.division = {:division}
          AND te.hours > 0 AND te.date <= {:end_date}
      ))
    ))
), selected_jobs AS MATERIALIZED (
  -- Unvalued jobs contribute to the exclusion count, not to WIP pricing.
  SELECT * FROM report_jobs WHERE project_value > 0
)
