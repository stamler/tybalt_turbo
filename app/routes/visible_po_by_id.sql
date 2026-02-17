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
  id = {:id}
  AND (
    is_active_visible = 1
    OR is_closed_cancelled_visible = 1
    OR is_unapproved_direct_visible = 1
    OR is_unapproved_second_stage_eligible = 1
  )
LIMIT 1
