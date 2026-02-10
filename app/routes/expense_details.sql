SELECT 
  e.id,
  e.uid,
  e.date,
  e.division,
  e.description,
  CAST(e.total AS REAL) AS total,
  e.payment_type,
  e.attachment,
  e.attachment_hash,
  e.rejector,
  e.rejected,
  e.rejection_reason,
  e.approver,
  e.approved,
  e.job,
  e.category,
  e.kind,
  e.pay_period_ending,
  e.allowance_types,
  e.submitted,
  e.committer,
  e.committed,
  e.committed_week_ending,
  CAST(e.distance AS REAL) AS distance,
  COALESCE(e.cc_last_4_digits, '') AS cc_last_4_digits,
  e.purchase_order,
  e.vendor,
  COALESCE(po.po_number, '') AS purchase_order_number,
  COALESCE(cl.name, '') AS client_name,
  COALESCE(ca.name, '') AS category_name,
  COALESCE(ek.name, '') AS kind_name,
  COALESCE(j.number, '') AS job_number,
  COALESCE(j.description, '') AS job_description,
  COALESCE(d.name, '') AS division_name,
  COALESCE(d.code, '') AS division_code,
  COALESCE(v.name, '') AS vendor_name,
  COALESCE(v.alias, '') AS vendor_alias,
  COALESCE(p0.given_name || ' ' || p0.surname, '') AS uid_name,
  COALESCE(p1.given_name || ' ' || p1.surname, '') AS approver_name,
  COALESCE(p2.given_name || ' ' || p2.surname, '') AS rejector_name,
  COALESCE(b.name, '') AS branch_name
FROM expenses e
LEFT JOIN jobs j ON e.job = j.id
LEFT JOIN clients cl ON j.client = cl.id
LEFT JOIN vendors v ON e.vendor = v.id
LEFT JOIN divisions d ON e.division = d.id
LEFT JOIN categories ca ON e.category = ca.id
LEFT JOIN expenditure_kinds ek ON e.kind = ek.id
LEFT JOIN profiles p0 ON e.uid = p0.uid
LEFT JOIN profiles p1 ON e.approver = p1.uid
LEFT JOIN profiles p2 ON e.rejector = p2.uid
LEFT JOIN purchase_orders po ON e.purchase_order = po.id
LEFT JOIN branches b ON e.branch = b.id
WHERE 
  e.id = {:id}

