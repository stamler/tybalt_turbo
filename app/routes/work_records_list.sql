SELECT
  te.work_record AS work_record,
  SUBSTR(te.work_record, 1, 1) AS prefix,
  COUNT(*) AS entry_count
FROM time_entries te
WHERE te.work_record != ''
GROUP BY te.work_record
ORDER BY te.work_record ASC
