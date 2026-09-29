WITH selected_jobs AS (
  SELECT j.* FROM jobs j
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
)
