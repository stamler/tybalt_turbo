-- Aggregate vendors with their expense and purchase order counts using LEFT JOINs.
-- The counts include only committed expenses and Active purchase orders.
-- referenced is true when any expense or purchase order uses the vendor, which
-- is what the vendors deleteRule checks, so the UI can tell when a delete will
-- be refused.
SELECT
  v.id,
  v.name,
  v.alias,
  COALESCE(ec.expenses_count, 0) AS expenses_count,
  COALESCE(poc.purchase_orders_count, 0) AS purchase_orders_count,
  (ec.vendor IS NOT NULL OR poc.vendor IS NOT NULL) AS referenced
FROM vendors v
LEFT JOIN (
  SELECT
    vendor,
    SUM(expenses.committed != '') AS expenses_count
  FROM expenses
  WHERE expenses.vendor != ''
  GROUP BY vendor
) ec ON ec.vendor = v.id
LEFT JOIN (
  SELECT
    vendor,
    SUM(purchase_orders.status = 'Active') AS purchase_orders_count
  FROM purchase_orders
  WHERE purchase_orders.vendor != ''
  GROUP BY vendor
) poc ON poc.vendor = v.id
WHERE ({:id} IS NULL OR {:id} = '' OR v.id = {:id})
ORDER BY v.name; 