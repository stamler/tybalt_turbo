# po_second_approval_required notifications

A function will populate the notifications list once per day with the `po_second_approval_required` notifications. These notifications will be sent to recipients who

1. hold the `po_approver` claim
2. At least one `purchase_orders` record that has a non-zero `approved` field and a status of `Unapproved` has an:
   - approval_total that is <= their `po_approver_props.max_amount` AND
   - if their `po_approver_props.divisions` list is not empty, a division that the list OR
   - their `po_approver_props.divisions` is not empty
3. HOWEVER, we group recipients by `po_approval_thresholds`. Reference the List/View rules in purchase_orders for details. Specifically: `po_approver_props_via_user_claim.max_amount` <= `purchase_orders_augmented.upper_threshold`. In other words, a user who is qualified to second_approve purchase orders above threshold X should not receive a notification to approve purchase_orders whose approval_total <= threshold X UNLESS no other user whose max_amount is <= threshold X but >= approval_total exists.

## Notes

The `purchase_orders_augmented` view is probably useful here. By joining it to the purchase_orders, we can see the thresholds range each po belongs to.

---

## Refined Specification for Daily Second Approval Notifications

This section details the logic for the daily scheduled job that generates `po_second_approval_required` notifications.

**Objective:** To notify the most appropriate, lowest-threshold approvers about purchase orders requiring second approval that haven't been assigned a `priority_second_approver` or where the priority approver hasn't acted.

**Trigger:** Daily scheduled task (e.g., cron job).

**Notification Template:** Use the `notification_templates` record where `code = 'po_second_approval_required'`.

**Process:**

1. **Identify Target Purchase Orders:**
    - Query the `purchase_orders` table.
    - Filter for records where:
        - `approved` is not empty (`approved != ''`).
        - `status` is `'Unapproved'`.
        - `second_approval` is empty (`second_approval = ''`).
        - *(Optional Consideration: Should we exclude POs that have a non-empty `priority_second_approver` and were approved less than 24 hours ago to give the priority approver time? The original spec doesn't mention this, but it might align with the VP logic in issue #18. For now, assume all POs meeting the above criteria are included).*
    - For clarity, let's call this set `PendingPOs`.

2. **Identify Potential Approvers:**
    - Query the `user_po_permission_data` view (or construct a similar query joining `users`, `user_claims`, `claims`, `po_approver_props`).
    - Filter for users where:
        - The `claims` JSON array contains `'po_approver'`.
    - Let's call this set `PotentialApprovers`.

3. **Determine Notifications per PO:**
    - Iterate through each `po` in `PendingPOs`. Let the current PO have `po_id`, `po_approval_total`, and `po_division`.
    - **Find Qualified Approvers for this PO:**
        - Filter `PotentialApprovers` to find the subset `QualifiedApproversForPO` where each `user` satisfies:
            - `user.max_amount >= po_approval_total`.
            - Division Match: `po_division` is present in the `user.divisions` JSON array OR `user.divisions` is an empty JSON array (`'[]'`).
    - **Apply Lowest Threshold Filter:**
        - If `QualifiedApproversForPO` is empty, continue to the next `po`.
        - Find the minimum `max_amount` among all users in `QualifiedApproversForPO`. Let this be `min_qualified_amount`.
        - Filter `QualifiedApproversForPO` further to get the final set `NotifiedApproversForPO` containing only those users whose `max_amount` equals `min_qualified_amount`.
    - **Generate Notifications:**
        - For each `user` in `NotifiedApproversForPO`:
            - Check if a `pending` notification already exists in the `notifications` table for this `user.id` and this `po_id` (using the `po_second_approval_required` template ID). This prevents duplicate notifications if the job runs unexpectedly.
            - If no pending notification exists, create a new record in the `notifications` table with:
                - `recipient`: `user.id`
                - `template`: ID of the `po_second_approval_required` template.
                - `status`: `'pending'`
                - `user`: (Optional) A system user ID, if applicable, to indicate the job created it.
                - *(Consideration: Should the notification record link directly to the PO? The current `notifications` schema doesn't have a dedicated field for this. Information might need to be embedded in the template content itself).*

**Data Sources:**

- `purchase_orders`: To find POs needing second approval.
- `user_po_permission_data` (View): To efficiently get user claims, max amounts, and division permissions. (Alternatively: `users`, `user_claims`, `claims`, `po_approver_props`).
- `notification_templates`: To get the ID for `po_second_approval_required`.
- `notifications`: To check for existing pending notifications and to insert new ones.

**Exclusions/Edge Cases:**

- POs that are already second-approved (`second_approval != ''`) or not yet first-approved (`approved = ''`) or have a status other than `'Unapproved'` are ignored.
- POs with a `priority_second_approver` are currently *included* in this daily job based on this spec (needs confirmation if this is desired).
- If multiple users share the exact same *minimum* qualifying `max_amount` for a specific PO, they will *all* receive a notification.
- If no user is qualified to approve a pending PO (based on amount or division), no notification is generated for that PO by this job.
