-- Count active purchase orders for a job with same filters
SELECT COUNT(*)
FROM   purchase_orders po
WHERE  po.status = 'Active'
  AND  po.job = {:id}
  AND  ({:branch}   IS NULL OR {:branch}   = '' OR po.branch   = {:branch})
  AND  ({:division} IS NULL OR {:division} = '' OR po.division = {:division})
  AND  ({:type} IS NULL OR {:type} = '' OR po.type = {:type})
  AND  ({:uid} IS NULL OR {:uid} = '' OR po.uid = {:uid}); 