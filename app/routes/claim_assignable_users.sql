SELECT
  u.id,
  COALESCE(ap.id, '') AS admin_profile_id,
  COALESCE(p.given_name, '') AS given_name,
  COALESCE(p.surname, '') AS surname,
  COALESCE(u.name, '') AS name,
  COALESCE(u.username, '') AS username,
  COALESCE(u.email, '') AS email
FROM users u
INNER JOIN admin_profiles ap ON ap.uid = u.id
  AND ap.active = 1
LEFT JOIN profiles p ON p.uid = u.id
WHERE NOT EXISTS (
  SELECT 1
  FROM user_claims uc
  WHERE uc.uid = u.id
    AND uc.cid = {:claimId}
)
ORDER BY
  COALESCE(NULLIF(p.surname, ''), NULLIF(u.name, ''), NULLIF(u.email, ''), u.username),
  COALESCE(NULLIF(p.given_name, ''), u.username),
  u.id
