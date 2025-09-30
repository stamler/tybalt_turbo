SELECT DISTINCT
  u.id AS id,
  COALESCE(p.given_name, '') AS given_name,
  COALESCE(p.surname, '') AS surname,
  COALESCE(u.email, '') AS email
FROM user_claims uc
JOIN claims c ON c.id = uc.cid AND c.name = 'busdev'
JOIN users u ON u.id = uc.uid
LEFT JOIN profiles p ON p.uid = uc.uid
ORDER BY p.surname, p.given_name;


