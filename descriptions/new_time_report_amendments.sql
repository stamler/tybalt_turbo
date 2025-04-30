SELECT
  ap.payroll_id,
  ta.committed_week_ending weekEnding, -- Amendments use commit week
  p.surname,
  p.given_name givenName,
  p.given_name || ' ' || p.surname AS name,
  p2.given_name || ' ' || p2.surname AS manager, -- Amendment committer is manager
  ta.meals_hours AS meals,
  ap.work_week_hours,
  ta.payout_request_amount,
  ap.salary,
  ta.uid AS primaryUid,
  ta.time_type,
  ta.hours,
  y.hasAmendmentsForWeeksEnding -- Join the calculated list
FROM
  time_amendments ta
LEFT JOIN admin_profiles ap ON ta.uid = ap.uid
LEFT JOIN profiles p ON ta.uid = p.uid
LEFT JOIN profiles p2 ON ta.committer = p2.uid
LEFT OUTER JOIN (
  -- Calculate the list of original weekEndings for amendments
  -- grouped by the week they were committed
  SELECT
    uid,
    committed_week_ending,
    json_group_array(week_ending) AS hasAmendmentsForWeeksEnding
  FROM (
    SELECT DISTINCT uid, committed_week_ending, week_ending
    FROM time_amendments
  ) DistinctTriplets
  GROUP BY uid, committed_week_ending
) y ON ta.uid = y.uid AND ta.committed_week_ending = y.committed_week_ending
WHERE ta.committed_week_ending = '2025-04-19'