-- Aggregate vendors with their expense and purchase order counts using LEFT JOINs
SELECT
  v.id,
  v.name,
  v.alias,
  COALESCE(ec.expenses_count, 0) AS expenses_count,
  COALESCE(poc.purchase_orders_count, 0) AS purchase_orders_count
FROM vendors v
LEFT JOIN (
  SELECT 
    vendor,
    COUNT(*) AS expenses_count
  FROM expenses
  GROUP BY vendor
) ec ON ec.vendor = v.id
LEFT JOIN (
  SELECT 
    vendor,
    COUNT(*) AS purchase_orders_count
  FROM purchase_orders
  GROUP BY vendor
) poc ON poc.vendor = v.id
WHERE ({:id} IS NULL OR {:id} = '' OR v.id = {:id})
ORDER BY v.name; 