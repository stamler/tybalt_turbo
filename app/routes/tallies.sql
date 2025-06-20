-- Parameterized tallies query
-- Params:
--   :uid           - the caller id (required)
--   :role          - either 'uid' (list by owner) or 'approver' (list by approver) (default 'uid')
--   :pendingOnly   - 1 to only include sheets with ts.approved = '' (default 0)
--   :approvedOnly  - 1 to only include sheets with ts.approved != '' (default 0)
-- The WHERE clause dynamically applies filters based on the above flags.
SELECT 
  MAX(ts.id) id,
  MAX(ts.week_ending) week_ending,
  MAX(ts.salary) salary, 
  MAX(ts.work_week_hours) work_week_hours, 
  MAX(ts.rejected) rejected, 
  MAX(ts.rejection_reason) rejection_reason, 
  MAX(ts.approved) approved,
  MAX(ts.approver) approver,
  MAX(ts.committer) committer,
  MAX(ts.committed) committed,
  MAX(p.given_name) given_name,
  MAX(p.surname) surname,
  MAX(ap.given_name || ' ' || ap.surname) approver_name,
  MAX(cp.given_name || ' ' || cp.surname) committer_name,
  SUM(CASE WHEN (tt.code = 'R' OR tt.code = 'RT') AND te.job == '' THEN te.hours ELSE 0 END) work_hours,
  SUM(CASE WHEN (tt.code = 'R' OR tt.code = 'RT') AND te.job != '' THEN te.hours ELSE 0 END) work_job_hours,
  SUM(CASE WHEN (tt.code = 'R' OR tt.code = 'RT') THEN te.hours ELSE 0 END) work_total_hours,
  SUM(te.meals_hours) meals_hours,
  SUM(CASE WHEN tt.code = 'OP' THEN te.hours ELSE 0 END) op_hours,
  SUM(CASE WHEN tt.code = 'OV' THEN te.hours ELSE 0 END) ov_hours,
  SUM(CASE WHEN tt.code = 'OH' THEN te.hours ELSE 0 END) oh_hours,
  SUM(CASE WHEN tt.code = 'OB' THEN te.hours ELSE 0 END) ob_hours,
  SUM(CASE WHEN tt.code = 'OS' THEN te.hours ELSE 0 END) os_hours,
  SUM(CASE WHEN tt.code = 'RB' THEN te.hours ELSE 0 END) rb_hours,
  SUM(CASE WHEN tt.code NOT IN ('R', 'RT', 'OR', 'OW', 'OTO', 'RB') THEN te.hours ELSE 0 END) non_work_total_hours,
  SUM(te.payout_request_amount) payout_request_amount,
  JSON_GROUP_ARRAY(DISTINCT j.number) FILTER (WHERE j.number IS NOT NULL) job_numbers,
  JSON_GROUP_ARRAY(DISTINCT d.code) FILTER (WHERE d.code IS NOT NULL) divisions,
  JSON_GROUP_ARRAY(DISTINCT d.name) FILTER (WHERE d.name IS NOT NULL) division_names,
  JSON_GROUP_ARRAY(DISTINCT tt.code) FILTER (WHERE tt.code IS NOT NULL) time_types,
  JSON_GROUP_ARRAY(DISTINCT tt.name) FILTER (WHERE tt.name IS NOT NULL) time_type_names,
  JSON_GROUP_ARRAY(te.date) FILTER (WHERE tt.code = 'OR') off_rotation_dates,
  JSON_GROUP_ARRAY(te.date) FILTER (WHERE tt.code = 'OW') off_week_dates,
  JSON_GROUP_ARRAY(te.date) FILTER (WHERE tt.code = 'OTO') payout_request_dates,
  JSON_GROUP_ARRAY(te.date) FILTER (WHERE tt.code = 'RB') bank_entry_dates
FROM time_entries te
INNER JOIN time_sheets ts ON te.tsid = ts.id -- use INNER JOIN to exclude time entries without a time sheet
LEFT JOIN divisions d ON  te.division = d.id
LEFT JOIN time_types tt ON te.time_type = tt.id
LEFT JOIN jobs j ON te.job = j.id
LEFT JOIN profiles p ON te.uid = p.uid
LEFT JOIN profiles ap ON ts.approver = ap.uid
LEFT JOIN profiles cp ON ts.committer = cp.uid
WHERE (
  ( {:role} = 'uid'      AND te.uid      = {:uid} ) OR
  ( {:role} = 'approver' AND ts.approver = {:uid} )
)
  AND ( {:pendingOnly}  = 0 OR ts.approved = '' )
  AND ( {:approvedOnly} = 0 OR ts.approved != '' )
GROUP BY te.tsid
ORDER BY ts.week_ending DESC