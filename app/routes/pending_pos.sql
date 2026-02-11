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
  COALESCE((SELECT MAX(threshold) FROM po_approval_thresholds WHERE threshold < po.approval_total), 0) AS lower_threshold,
  COALESCE((SELECT MIN(threshold) FROM po_approval_thresholds WHERE threshold >= po.approval_total), 1000000) AS upper_threshold,
  (SELECT COUNT(*) FROM expenses WHERE expenses.purchase_order = po.id AND expenses.committed != '') AS committed_expenses_count,
  (p0.given_name || ' ' || p0.surname) AS uid_name,
  (p1.given_name || ' ' || p1.surname) AS approver_name,
  (p2.given_name || ' ' || p2.surname) AS second_approver_name,
  (p3.given_name || ' ' || p3.surname) AS priority_second_approver_name,
  (p4.given_name || ' ' || p4.surname) AS rejector_name,
  po2.po_number AS parent_po_number,
  v.name AS vendor_name,
  v.alias AS vendor_alias,
  j.number AS job_number,
  cl.name AS client_name,
  cl.id AS client_id,
  j.description AS job_description,
  d.code AS division_code,
  d.name AS division_name,
  c.name AS category_name
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
LEFT JOIN expenditure_kinds AS ek ON po.kind = ek.id
WHERE
  po.status = 'Unapproved' AND po.rejected = '' AND (
    -- Block 1: First-approval candidates
    -- Show POs that have not yet been first-approved (approved = '') where the caller
    -- is eligible to perform first approval. Eligibility is determined by:
    -- - Caller holds the po_approver claim (uc.cid = {:poApproverClaim})
    -- - Caller is division-qualified (po_approver_props.divisions is empty OR contains po.division)
    (
      po.approved = ''
      AND EXISTS (
        SELECT 1 FROM user_claims uc
        JOIN po_approver_props pap ON pap.user_claim = uc.id
        WHERE uc.uid = {:userId}
          AND uc.cid = {:poApproverClaim}
          AND (
            JSON_ARRAY_LENGTH(pap.divisions) = 0 OR EXISTS (
              SELECT 1 FROM JSON_EACH(pap.divisions) WHERE value = po.division
            )
          )
      )
    )
    OR
    -- Block 2: Priority second-approval within the 24-hour exclusive window
    -- Show POs that already have first approval (approved != '') and still require second
    -- approval (second_approval = ''), where the caller is the designated
    -- priority_second_approver. We do not apply the 24-hour cutoff here because
    -- during the exclusive window only the designated user should see the item.
    (
      po.approved != '' AND po.second_approval = '' AND po.priority_second_approver = {:userId}
    )
    OR
    -- Block 3: General second-approval after the 24-hour window
    -- Show POs that are awaiting second approval and have been pending for more than
    -- 24 hours since last update (updated < now - 24h). The caller must be a qualified
    -- second approver:
    -- - Holds po_approver claim
    -- - Division-qualified (divisions empty OR contains po.division)
    -- - Amount-qualified (max_amount >= approval_total AND <= the tier ceiling for this PO)
    -- This may include the original approver if they also meet second-approver criteria.
    (
      po.approved != '' AND po.second_approval = ''
      AND po.updated < strftime('%Y-%m-%d %H:%M:%fZ', 'now', '-24 hours')
      AND EXISTS (
        SELECT 1 FROM user_claims uc
        JOIN po_approver_props pap ON pap.user_claim = uc.id
        WHERE uc.uid = {:userId}
          AND uc.cid = {:poApproverClaim}
          AND (
            JSON_ARRAY_LENGTH(pap.divisions) = 0 OR EXISTS (
              SELECT 1 FROM JSON_EACH(pap.divisions) WHERE value = po.division
            )
          )
          AND (
            CASE
              WHEN COALESCE(ek.name, 'standard') = 'standard' AND po.job != '' THEN COALESCE(pap.project_max, 0)
              WHEN COALESCE(ek.name, 'standard') = 'standard' THEN COALESCE(pap.max_amount, 0)
              WHEN COALESCE(ek.name, 'standard') = 'sponsorship' THEN COALESCE(pap.sponsorship_max, 0)
              WHEN COALESCE(ek.name, 'standard') = 'staff_and_social' THEN COALESCE(pap.staff_and_social_max, 0)
              WHEN COALESCE(ek.name, 'standard') = 'media_and_event' THEN COALESCE(pap.media_and_event_max, 0)
              WHEN COALESCE(ek.name, 'standard') = 'computer' THEN COALESCE(pap.computer_max, 0)
              ELSE 0
            END
          ) BETWEEN po.approval_total AND COALESCE((SELECT MIN(threshold) FROM po_approval_thresholds WHERE threshold >= po.approval_total), 1000000)
      )
    )
  )
ORDER BY po.updated DESC
