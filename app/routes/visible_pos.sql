WITH visibility_base AS (
__PO_VISIBILITY_BASE__
)
SELECT
  id,
  po_number,
  status,
  uid,
  type,
  date,
  end_date,
  frequency,
  division,
  description,
  total,
  payment_type,
  attachment,
  rejector,
  rejected,
  rejection_reason,
  approver,
  approved,
  second_approver,
  second_approval,
  canceller,
  cancelled,
  job,
  category,
  kind,
  vendor,
  parent_po,
  created,
  updated,
  closer,
  closed,
  closed_by_system,
  priority_second_approver,
  approval_total,
  committed_expenses_count,
  expenses_total,
  recurring_expected_occurrences,
  recurring_remaining_occurrences,
  cumulative_remaining_balance,
  uid_name,
  approver_name,
  second_approver_name,
  priority_second_approver_name,
  rejector_name,
  parent_po_number,
  vendor_name,
  vendor_alias,
  job_number,
  client_name,
  client_id,
  job_description,
  division_code,
  division_name,
  category_name
FROM visibility_base
WHERE
  (
    is_active_visible = 1
    OR is_closed_cancelled_visible = 1
    OR is_unapproved_direct_visible = 1
    OR is_unapproved_second_stage_eligible = 1
  )
  AND (
    {:scope} = 'all'
    OR ({:scope} = 'mine' AND uid = {:userId})
    OR ({:scope} = 'active' AND status = 'Active')
    OR (
      {:scope} = 'rejected'
      AND status = 'Unapproved'
      AND rejected != ''
      AND uid = {:userId}
    )
    OR (
      {:scope} = 'stale'
      AND status = 'Active'
      AND (
        (second_approval != '' AND second_approval < {:staleBefore})
        OR (approved < {:staleBefore})
      )
    )
    OR (
      {:scope} = 'expiring'
      AND status = 'Active'
      AND type = 'Recurring'
      AND end_date != ''
      AND end_date <= {:expiringBefore}
    )
  )
ORDER BY
  CASE WHEN {:scope} = 'expiring' THEN end_date END ASC,
  date DESC,
  updated DESC
