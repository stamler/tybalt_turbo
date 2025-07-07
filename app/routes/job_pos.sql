-- List active purchase orders for a job with optional filters and pagination
SELECT po.id,
       po.po_number,
       po.date,
       po.total,
       po.type,
       d.code AS division_code,
       p.surname AS surname,
       p.given_name AS given_name
FROM   purchase_orders po
LEFT   JOIN divisions d ON po.division = d.id
LEFT   JOIN profiles  p ON po.uid      = p.uid
WHERE  po.status = 'Active'
  AND  po.job = {:id}
  AND  ({:division} IS NULL OR {:division} = '' OR po.division = {:division})
  AND  ({:type} IS NULL OR {:type} = '' OR po.type = {:type})
  AND  ({:uid} IS NULL OR {:uid} = '' OR po.uid = {:uid})
ORDER BY po.date DESC
LIMIT {:limit} OFFSET {:offset}; 