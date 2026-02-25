SELECT
  c.id,
  c.name,
  c.description,
  COUNT(uc.id) AS holder_count
FROM claims c
LEFT JOIN user_claims uc ON uc.cid = c.id
GROUP BY c.id, c.name, c.description
ORDER BY c.name
