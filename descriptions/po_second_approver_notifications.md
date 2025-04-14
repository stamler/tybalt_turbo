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

## The View

### pending_items_for_qualified_po_second_approvers

```sql
-- This view identifies users qualified to perform second approvals on purchase orders
-- that have been awaiting second approval for more than 24 hours (based on 'updated' timestamp)
-- and counts how many such POs each user is qualified for.
-- It implements the logic described in the "The View" section of po_second_approver_notifications.md

-- Note: This view considers a user qualified for a PO if:
-- 1. They have the 'po_approver' claim.
-- 2. The PO needs second approval (approved, not rejected, status='Unapproved', second_approval='', updated > 24h ago).
-- 3. The user's max_amount >= PO's approval_total.
-- 4. The user has permission for the PO's division (user.divisions is empty OR contains po.division).
-- 5. The user's max_amount <= PO's upper_threshold (ensuring users aren't counted for POs below their effective tier).

WITH QualifiedUsers AS (
    -- Select users with the 'po_approver' claim and their properties
    SELECT
        u.id AS user_id,
        pap.max_amount,
        pap.divisions
    FROM users u
    -- Join user claims to get claim IDs
    JOIN user_claims uc ON u.id = uc.uid
    -- Join claims to filter by claim name
    JOIN claims c ON uc.cid = c.id
    -- Join PO approver properties using the user_claims ID
    JOIN po_approver_props pap ON uc.id = pap.user_claim
    WHERE c.name = 'po_approver'
),
POsNeedingSecondApproval AS (
    -- Select Purchase Orders that require second approval and are past the 24h priority window
    SELECT
        po.id AS po_id,
        po.approval_total,
        po.division,
        poa.upper_threshold
    FROM purchase_orders po
    -- Join with the augmented view to get threshold information
    JOIN purchase_orders_augmented poa ON po.id = poa.id
    WHERE
        po.approved != ''           -- Must be first-approved
        AND po.rejected == ''           -- Must not be rejected
        AND po.status = 'Unapproved'  -- Must still be Unapproved (awaiting second approval)
        AND po.second_approval = '' -- Must not be second-approved yet
        -- Check if the PO was last updated more than 24 hours ago
        AND po.updated < strftime('%Y-%m-%d %H:%M:%fZ', 'now', '-24 hours')
)
-- Final selection: Group by user and count the POs they are qualified to second-approve
SELECT
    qu.user_id AS id,
    COUNT(pnsa.po_id) AS num_pos_qualified
FROM QualifiedUsers qu
-- Left join ensures all qualified users appear, even if they have 0 POs to approve currently
LEFT JOIN POsNeedingSecondApproval pnsa
    -- Join condition: Check if the user (qu) is qualified for the PO (pnsa)
    ON qu.max_amount >= pnsa.approval_total  -- 1. Check amount threshold
    AND (
        json_valid(qu.divisions) AND json_array_length(qu.divisions) = 0 -- 2a. User can approve any division (divisions is '[]')
        OR (
           json_valid(qu.divisions) AND EXISTS (SELECT 1 FROM json_each(qu.divisions) WHERE value = pnsa.division) -- 2b. User's divisions list contains the PO's division
        )
    )
    AND qu.max_amount <= pnsa.upper_threshold -- 3. Tier check: User's max_amount is within the PO's threshold band
GROUP BY qu.user_id; -- Group results per user
```
