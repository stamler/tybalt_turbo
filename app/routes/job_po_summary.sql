-- Summary of active purchase orders for a job with optional filters.
SELECT
  SUM(po.total)                           total_amount,
  MIN(po.date)                            earliest_po,
  MAX(po.date)                            latest_po,
  json_group_array(
    DISTINCT json_object('id', d.id, 'code', d.code)
  )                                        divisions,
  json_group_array(
    DISTINCT json_object('name', po.type)
  )                                        types,
  json_group_array(
    DISTINCT json_object('id', p.uid, 'name', p.given_name || ' ' || p.surname)
  )                                        names
FROM   purchase_orders po
LEFT   JOIN divisions d ON po.division = d.id
LEFT   JOIN profiles  p ON po.uid      = p.uid
WHERE  po.status = 'Active'
  AND  po.job = {:id}
  AND  ({:division} IS NULL OR {:division} = '' OR po.division = {:division})
  AND  ({:type} IS NULL OR {:type} = '' OR po.type = {:type})
  AND  ({:uid} IS NULL OR {:uid} = '' OR po.uid = {:uid}); 