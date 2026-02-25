WITH qualifying_pos AS (
  SELECT id, uid, approver, second_approver
  FROM purchase_orders
  WHERE status IN ('Active', 'Closed')
    AND (_imported IS NULL OR _imported = false)
),
approved_expenses AS (
  SELECT e.id, e.uid, e.approver, e.committer
  FROM expenses e
  INNER JOIN qualifying_pos qp ON e.purchase_order = qp.id
  WHERE e.approved != ''
    AND e.rejected = ''
),
distinct_users AS (
  SELECT user_id FROM (
    SELECT uid AS user_id FROM qualifying_pos WHERE uid != ''
    UNION
    SELECT approver AS user_id FROM qualifying_pos WHERE approver != ''
    UNION
    SELECT second_approver AS user_id FROM qualifying_pos WHERE second_approver != ''
    UNION
    SELECT uid AS user_id FROM approved_expenses WHERE uid != ''
    UNION
    SELECT approver AS user_id FROM approved_expenses WHERE approver != ''
    UNION
    SELECT committer AS user_id FROM approved_expenses WHERE committer != ''
  )
  WHERE user_id != ''
)
SELECT
  (SELECT COUNT(*) FROM qualifying_pos) AS qualifying_po_count,
  (SELECT COUNT(*) FROM approved_expenses) AS approved_expense_count,
  (SELECT COUNT(*) FROM distinct_users) AS distinct_user_count
