-- Divisions with committed time on any job whose client is this client, and
-- the hours in each. Jobs where the client is only the job owner do not count.
-- CROSS JOIN makes SQLite start from the client's jobs. Left to choose, it
-- scans every time entry in division order to suit the GROUP BY, which is
-- several times slower.
SELECT
  d.id,
  d.code,
  d.name,
  SUM(te.hours) AS hours
FROM jobs j
CROSS JOIN time_entries te ON te.job = j.id
JOIN time_sheets ts ON ts.id = te.tsid
JOIN divisions d ON d.id = te.division
WHERE j.client = {:id}
  AND te.hours > 0
  AND ts.committed != ''
GROUP BY d.id
ORDER BY d.code;
