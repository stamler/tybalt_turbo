# `po_second_approval_required` Notifications (Current)

This describes the daily notification input for second-stage PO approvals.

## Source of Recipients

Recipients are read from:

- `pending_items_for_qualified_po_second_approvers`

Each row is:

- `id`: user id
- `num_pos_qualified`: count of POs currently visible to that user in post-timeout Stage 2

`QueuePoSecondApproverNotifications` sends `po_second_approval_required` to users where `num_pos_qualified > 0`.

## Qualification Rules (as implemented by the view)

A PO contributes to a user's count only when all are true:

1. PO is first-approved (`approved != ''`)
2. PO is not second-approved (`second_approval = ''`)
3. PO status is still `Unapproved`
4. PO is past Stage-2 timeout window (`approved + T`)
5. PO is dual-required for its kind (`approval_total > second_approval_threshold` and threshold > 0)
6. User is an active `po_approver`
7. User can approve the PO division
8. User is second-stage eligible for the PO kind/amount:
   - resolved limit column is used by kind/job context
   - limit `> second_approval_threshold`
   - limit `>= approval_total`

`T` is loaded from `app_config.purchase_orders.second_stage_timeout_hours` with backend fallback to `24` hours when missing/invalid/non-positive.

## Important Changes from Legacy Model

No longer used:

- `po_approval_thresholds`
- `upper_threshold` / `lower_threshold`
- tier-ceiling grouping
- `user_po_permission_data`

This notification flow now follows stage pools and timeout-based visibility, matching pending queue behavior.
