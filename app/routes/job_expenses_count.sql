-- count total expenses for a job with the same filters as the main query
SELECT COUNT(*)
FROM   expenses e
WHERE  e.committed != ''
  AND  e.job = {:id}
  AND  ({:branch}       IS NULL OR {:branch}       = '' OR e.branch       = {:branch})
  AND  ({:division}     IS NULL OR {:division}     = '' OR e.division     = {:division})
  AND  ({:payment_type} IS NULL OR {:payment_type} = '' OR e.payment_type = {:payment_type})
  AND  ({:uid}          IS NULL OR {:uid}          = '' OR e.uid          = {:uid})
  AND (
    ({:category} IS NULL OR {:category} = '')
    OR
    ({:category} = 'none' AND e.category = '')
    OR
    ({:category} != 'none' AND e.category = {:category})
  ); 