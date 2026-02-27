/*
  -----------------------------------------------------------------------------
  PO VISIBILITY / ACTIONABILITY BASE QUERY
  -----------------------------------------------------------------------------
  Purpose
  - Provide one shared SQL source of truth for purchase-order visibility flags.
  - Feed both API surfaces:
    1) /api/purchase_orders/visible* (broad visibility)
    2) /api/purchase_orders/pending* (actionability queue)

  How this file is consumed
  - Go embeds this file and injects it into thin selector queries (pending/visible).
  - This file should return one row per purchase order plus computed boolean flags.
  - Thin selector queries decide which rows to expose based on those flags.

  Bound parameters expected by callers
  - userId: authenticated caller user id.
  - poApproverClaim: claim id used for PO approver policy rows.
  - timeoutHours: second-stage timeout window for pending/actionability checks.

  High-level policy model encoded here
  - Active POs are visible to any authenticated user.
  - Closed/Cancelled are visible to creator, approvers, or report claim holders.
  - Unapproved visibility has two classes:
    - Direct visibility (creator, assigned first approver, and priority second
      approver for post-first-approval records).
    - Policy-based second-stage visibility (eligible second approvers).
  - Actionability (pending queue) is intentionally narrower than visibility:
    - Stage 1 assigned-first-approver path (including assigned approver self-bypass).
    - Stage 2 priority owner path.
    - Stage 2 fallback after timeout path.

  Maintainability notes
  - Kind-to-limit resolution is intentionally centralized in caller_po_limits.
  - Threshold fallback uses job-based effective kind (capital when no job, project when job present).
  - Do not reintroduce duplicated limit CASE expressions elsewhere in this file.
  -----------------------------------------------------------------------------
*/

-- CTE: caller_context
-- Captures caller-level claims that are independent of any particular PO row.
WITH caller_context AS (
  SELECT
    CASE
      WHEN EXISTS (
        SELECT 1
        FROM user_claims uc
        JOIN claims c1 ON c1.id = uc.cid
        WHERE uc.uid = {:userId}
          AND c1.name = 'report'
      ) THEN 1
      ELSE 0
    END AS has_report_claim
),

-- CTE: caller_approver_props
-- Fetches caller PO approver properties only when caller has the PO approver claim,
-- and only while the caller's admin profile is active.
caller_approver_props AS (
  SELECT
    pap.divisions,
    pap.max_amount,
    pap.project_max,
    pap.sponsorship_max,
    pap.staff_and_social_max,
    pap.media_and_event_max,
    pap.computer_max
  FROM user_claims uc
  JOIN po_approver_props pap ON pap.user_claim = uc.id
  JOIN admin_profiles ap ON ap.uid = uc.uid AND ap.active = 1
  WHERE uc.uid = {:userId}
    AND uc.cid = {:poApproverClaim}
),

-- CTE: caller_po_limits
-- Centralized per-(PO, caller) policy materialization.
--
-- For each PO, computes:
--   - second_approval_threshold with standard-kind fallback
--   - resolved_limit using kind + hasJob rules
--
-- This prevents repeating the kind->limit CASE across multiple predicates.
caller_po_limits AS (
  SELECT
    po.id AS po_id,
    cap.divisions,
    COALESCE(
      ek.second_approval_threshold,
      ek_fallback.second_approval_threshold,
      0
    ) AS second_approval_threshold,
    CASE COALESCE(ek.name, CASE WHEN po.job != '' THEN 'project' ELSE 'capital' END)
      WHEN 'capital' THEN cap.max_amount
      WHEN 'project' THEN cap.project_max
      WHEN 'sponsorship' THEN cap.sponsorship_max
      WHEN 'staff_and_social' THEN cap.staff_and_social_max
      WHEN 'media_and_event' THEN cap.media_and_event_max
      WHEN 'computer' THEN cap.computer_max
      ELSE NULL
    END AS resolved_limit
  FROM purchase_orders po
  JOIN caller_approver_props cap ON 1 = 1
  LEFT JOIN expenditure_kinds ek ON po.kind = ek.id
  -- Fallback kind for threshold when PO kind is missing: use job-based effective kind.
  LEFT JOIN expenditure_kinds ek_fallback
    ON ek.id IS NULL
    AND ek_fallback.name = CASE WHEN po.job != '' THEN 'project' ELSE 'capital' END
)

-- Main row projection: one row per purchase order + visibility/actionability flags.
SELECT
  po.id,
  po.po_number,
  po.status,
  po.uid,
  po.type,
  po.date,
  po.end_date,
  po.frequency,
  po.division,
  po.description,
  po.total,
  po.payment_type,
  po.attachment,
  po.rejector,
  po.rejected,
  po.rejection_reason,
  po.approver,
  po.approved,
  po.second_approver,
  po.second_approval,
  po.canceller,
  po.cancelled,
  po.job,
  po.category,
  po.kind,
  po.vendor,
  po.parent_po,
  po.created,
  po.updated,
  po.closer,
  po.closed,
  po.closed_by_system,
  po.priority_second_approver,
  po.approval_total,
  (SELECT COUNT(*) FROM expenses WHERE expenses.purchase_order = po.id AND expenses.committed != '') AS committed_expenses_count,
  COALESCE((SELECT SUM(expenses.total) FROM expenses WHERE expenses.purchase_order = po.id), 0) AS expenses_total,
  CASE
    WHEN po.type = 'Recurring' AND po.end_date != '' AND po.frequency != '' THEN
      CASE po.frequency
        WHEN 'Weekly' THEN CAST((julianday(po.end_date) - julianday(po.date)) / 7 AS INTEGER)
        WHEN 'Biweekly' THEN CAST((julianday(po.end_date) - julianday(po.date)) / 14 AS INTEGER)
        WHEN 'Monthly' THEN CAST((julianday(po.end_date) - julianday(po.date)) / 30 AS INTEGER)
        ELSE 0
      END
    ELSE 0
  END AS recurring_expected_occurrences,
  CASE
    WHEN po.type = 'Recurring' AND po.end_date != '' AND po.frequency != '' THEN
      MAX(
        (
          CASE po.frequency
            WHEN 'Weekly' THEN CAST((julianday(po.end_date) - julianday(po.date)) / 7 AS INTEGER)
            WHEN 'Biweekly' THEN CAST((julianday(po.end_date) - julianday(po.date)) / 14 AS INTEGER)
            WHEN 'Monthly' THEN CAST((julianday(po.end_date) - julianday(po.date)) / 30 AS INTEGER)
            ELSE 0
          END
        ) - (SELECT COUNT(*) FROM expenses WHERE expenses.purchase_order = po.id AND expenses.committed != ''),
        0
      )
    ELSE 0
  END AS recurring_remaining_occurrences,
  CASE
    WHEN po.type = 'Cumulative' THEN
      po.total - COALESCE((SELECT SUM(expenses.total) FROM expenses WHERE expenses.purchase_order = po.id), 0)
    ELSE 0
  END AS cumulative_remaining_balance,
  COALESCE((p0.given_name || ' ' || p0.surname), '') AS uid_name,
  COALESCE((p1.given_name || ' ' || p1.surname), '') AS approver_name,
  COALESCE((p2.given_name || ' ' || p2.surname), '') AS second_approver_name,
  COALESCE((p3.given_name || ' ' || p3.surname), '') AS priority_second_approver_name,
  COALESCE((p4.given_name || ' ' || p4.surname), '') AS rejector_name,
  COALESCE(po2.po_number, '') AS parent_po_number,
  COALESCE(v.name, '') AS vendor_name,
  COALESCE(v.alias, '') AS vendor_alias,
  COALESCE(j.number, '') AS job_number,
  COALESCE(cl.name, '') AS client_name,
  COALESCE(cl.id, '') AS client_id,
  COALESCE(j.description, '') AS job_description,
  COALESCE(d.code, '') AS division_code,
  COALESCE(d.name, '') AS division_name,
  COALESCE(c.name, '') AS category_name,

  -- Flag: Active records are visible to any authenticated user.
  CASE
    WHEN po.status = 'Active' THEN 1
    ELSE 0
  END AS is_active_visible,

  -- Flag: Closed/Cancelled visibility for direct participants + report claim.
  CASE
    WHEN
      po.status IN ('Cancelled', 'Closed')
      AND (
        po.uid = {:userId}
        OR po.approver = {:userId}
        OR po.second_approver = {:userId}
        OR caller_context.has_report_claim = 1
      )
    THEN 1
    ELSE 0
  END AS is_closed_cancelled_visible,

  -- Flag: Direct unapproved visibility (no policy pool computation required).
  -- Creators can always see their own unapproved records, including rejected
  -- ones, so they can review rejection context and resubmit. Rejectors can
  -- also see rejected records they acted on.
  CASE
    WHEN
      po.status = 'Unapproved'
      AND (
        po.uid = {:userId}
        OR (po.rejected != '' AND po.rejector = {:userId})
        OR (
          po.rejected = ''
          AND (
            (po.approved = '' AND po.approver = {:userId})
            OR (
              po.approved != ''
              AND po.second_approval = ''
              AND (
                po.approver = {:userId}
                OR po.priority_second_approver = {:userId}
              )
            )
          )
        )
      )
    THEN 1
    ELSE 0
  END AS is_unapproved_direct_visible,

  -- Flag: Non-direct unapproved visibility via second-stage eligibility.
  --
  -- This does NOT apply timeout gating. It answers:
  -- "Can caller ever be valid second-stage approver for this PO right now?"
  CASE
    WHEN
      po.status = 'Unapproved'
      AND po.rejected = ''
      AND po.approved != ''
      AND po.second_approval = ''
      AND EXISTS (
        SELECT 1
        FROM caller_po_limits cpl
        WHERE
          cpl.po_id = po.id
          -- Division gate: empty division list means unrestricted.
          AND (
            JSON_ARRAY_LENGTH(cpl.divisions) = 0
            OR EXISTS (
              SELECT 1
              FROM JSON_EACH(cpl.divisions)
              WHERE value = po.division
            )
          )
          AND cpl.resolved_limit IS NOT NULL
          -- Dual-stage eligibility requires thresholded PO and fully sufficient limit.
          AND cpl.second_approval_threshold > 0
          AND po.approval_total > cpl.second_approval_threshold
          AND cpl.resolved_limit > cpl.second_approval_threshold
          AND cpl.resolved_limit >= po.approval_total
      )
    THEN 1
    ELSE 0
  END AS is_unapproved_second_stage_eligible,

  -- Flag: Actionable "pending now" semantics.
  --
  -- This encodes queue ownership paths used by /pending:
  --  1) Stage 1 assigned approver path (including first-stage pool partition rule
  --     and assigned-approver self-bypass for dual-stage records).
  --  2) Stage 2 priority-second-approver exclusive path.
  --  3) Stage 2 fallback path after timeout.
  CASE
    WHEN
      po.status = 'Unapproved'
      AND po.rejected = ''
      AND (
        (
          -- Stage 1 pending: assigned approver only.
          po.approved = ''
          AND po.approver = {:userId}
          AND EXISTS (
            SELECT 1
            FROM caller_po_limits cpl
            WHERE
              cpl.po_id = po.id
              AND (
                JSON_ARRAY_LENGTH(cpl.divisions) = 0
                OR EXISTS (
                  SELECT 1
                  FROM JSON_EACH(cpl.divisions)
                  WHERE value = po.division
                )
              )
              AND cpl.resolved_limit IS NOT NULL
              AND (
                (
                  -- Dual-stage case: first-stage pool is <= threshold.
                  cpl.second_approval_threshold > 0
                  AND po.approval_total > cpl.second_approval_threshold
                  AND cpl.resolved_limit <= cpl.second_approval_threshold
                )
                OR (
                  -- Dual-stage assigned-approver self-bypass path.
                  cpl.second_approval_threshold > 0
                  AND po.approval_total > cpl.second_approval_threshold
                  AND cpl.resolved_limit > cpl.second_approval_threshold
                  AND cpl.resolved_limit >= po.approval_total
                )
                OR (
                  -- Single-stage case: threshold not triggered.
                  NOT (
                    cpl.second_approval_threshold > 0
                    AND po.approval_total > cpl.second_approval_threshold
                  )
                )
              )
          )
        )
        OR (
          -- Stage 2 priority-owner window.
          po.approved != ''
          AND po.second_approval = ''
          AND po.priority_second_approver = {:userId}
        )
        OR (
          -- Stage 2 fallback queue after timeout expiration.
          po.approved != ''
          AND po.second_approval = ''
          AND po.approved < strftime('%Y-%m-%d %H:%M:%fZ', 'now', '-' || CAST({:timeoutHours} AS TEXT) || ' hours')
          AND EXISTS (
            SELECT 1
            FROM caller_po_limits cpl
            WHERE
              cpl.po_id = po.id
              AND (
                JSON_ARRAY_LENGTH(cpl.divisions) = 0
                OR EXISTS (
                  SELECT 1
                  FROM JSON_EACH(cpl.divisions)
                  WHERE value = po.division
                )
              )
              AND cpl.resolved_limit IS NOT NULL
              AND cpl.second_approval_threshold > 0
              AND po.approval_total > cpl.second_approval_threshold
              AND cpl.resolved_limit > cpl.second_approval_threshold
              AND cpl.resolved_limit >= po.approval_total
          )
        )
      )
    THEN 1
    ELSE 0
  END AS is_unapproved_actionable_now
FROM purchase_orders AS po
LEFT JOIN profiles AS p0 ON po.uid = p0.uid
LEFT JOIN profiles AS p1 ON po.approver = p1.uid
LEFT JOIN profiles AS p2 ON po.second_approver = p2.uid
LEFT JOIN profiles AS p3 ON po.priority_second_approver = p3.uid
LEFT JOIN profiles AS p4 ON po.rejector = p4.uid
LEFT JOIN purchase_orders AS po2 ON po.parent_po = po2.id
LEFT JOIN vendors AS v ON po.vendor = v.id
LEFT JOIN jobs AS j ON po.job = j.id
LEFT JOIN divisions AS d ON po.division = d.id
LEFT JOIN categories AS c ON po.category = c.id
LEFT JOIN clients AS cl ON j.client = cl.id
CROSS JOIN caller_context
