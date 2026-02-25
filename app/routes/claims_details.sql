-- INNER JOIN profiles: every user must have a profile (core identity record).
-- A missing profile indicates a data integrity issue, and a nameless row
-- would not be useful to display, so we intentionally exclude those.
--
-- LEFT JOIN admin_profiles: not every user necessarily has an admin_profiles
-- record. We still want to show them as claim holders even if the link to
-- their admin profile won't work.
SELECT
  ap.id AS admin_profile_id,
  p.given_name,
  p.surname
FROM user_claims uc
INNER JOIN profiles p ON p.uid = uc.uid
LEFT JOIN admin_profiles ap ON ap.uid = uc.uid
WHERE uc.cid = {:claimId}
ORDER BY p.surname, p.given_name
