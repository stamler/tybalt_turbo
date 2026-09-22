-- Appended to job_priced_time_entries.sql. Keep labour pricing identical to
-- the Staff and Divisions summaries. Aggregate each source before joining.
, time_value AS (
  SELECT COALESCE(SUM(value), 0) AS time_value,
    COALESCE(SUM(hours), 0) AS hours,
    COALESCE(SUM(estimated_hours), 0) AS estimated_hours,
    COALESCE(SUM(unpriced_hours), 0) AS unpriced_hours
  FROM priced_entries
), committed_expenses AS (
  SELECT e.*,
    CASE
      WHEN e.settled_total > 0 THEN e.settled_total
      WHEN COALESCE(e.currency, '') = '' OR c.code = 'CAD' THEN e.total
      WHEN e.total = 0 THEN 0
      ELSE NULL
    END AS cad_value
  FROM expenses e
  LEFT JOIN currencies c ON c.id = e.currency
  WHERE e.committed != '' AND COALESCE(e.rejected, '') = ''
    AND e.date <= {:end_date}
    AND (e.job = {:job_id} OR e.purchase_order IN (
      SELECT id FROM purchase_orders WHERE job = {:job_id} AND status = 'Active'
    ))
), expense_value AS (
  SELECT COALESCE(SUM(cad_value), 0) AS expense_value,
    COUNT(*) FILTER (WHERE cad_value IS NULL) AS unpriced_expenses
  FROM committed_expenses WHERE job = {:job_id}
), po_spend AS (
  -- PO and expense currencies must match on save. Subtract native amounts
  -- before converting the outstanding balance, not settled CAD payments.
  SELECT purchase_order, SUM(total) AS native_spend
  FROM committed_expenses GROUP BY purchase_order
), po_balances AS (
  SELECT po.id,
    CASE
      WHEN po.type = 'Recurring' AND po.approval_total <= 0 AND po.total > 0 THEN NULL
      ELSE MAX(0, (CASE WHEN po.type = 'Recurring' THEN po.approval_total ELSE po.total END)
        - COALESCE(s.native_spend, 0))
    END AS native_balance,
    CASE
      WHEN COALESCE(po.currency, '') = '' OR c.code = 'CAD' THEN 1
      WHEN c.rate > 0 THEN c.rate
      ELSE NULL
    END AS cad_rate,
    CASE WHEN COALESCE(po.currency, '') != '' AND COALESCE(c.code, '') != 'CAD' THEN 1 ELSE 0 END AS foreign_currency
  FROM purchase_orders po
  LEFT JOIN po_spend s ON s.purchase_order = po.id
  LEFT JOIN currencies c ON c.id = po.currency
  WHERE po.job = {:job_id} AND po.status = 'Active'
), po_value AS (
  SELECT COALESCE(SUM(ROUND(native_balance * cad_rate, 2)), 0) AS po_value,
    COUNT(*) FILTER (WHERE native_balance IS NULL OR (native_balance > 0 AND cad_rate IS NULL)) AS unpriced_pos,
    COUNT(*) FILTER (WHERE native_balance > 0 AND cad_rate IS NOT NULL AND foreign_currency = 1) AS estimated_pos
  FROM po_balances
)
SELECT COALESCE(j.project_value, 0) AS project_value,
  {:end_date} AS as_of,
  CASE WHEN COALESCE(j.rate_sheet, '') = '' THEN 1 ELSE 0 END AS no_rate_sheet,
  ROUND(t.time_value, 2) AS time_value, t.hours, t.estimated_hours, t.unpriced_hours,
  ROUND(e.expense_value, 2) AS expense_value, e.unpriced_expenses,
  ROUND(p.po_value, 2) AS po_value, p.unpriced_pos, p.estimated_pos
FROM jobs j CROSS JOIN time_value t CROSS JOIN expense_value e CROSS JOIN po_value p
WHERE j.id = {:job_id};
