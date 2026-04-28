SELECT 
  e.id,
  e.uid,
  e.creator,
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
  COALESCE(e.currency, '') AS currency,
  CAST(COALESCE(e.settled_total, 0) AS REAL) AS settled_total,
  COALESCE(e.settler, '') AS settler,
  COALESCE(e.settled, '') AS settled,
  e.purchase_order,
  e.vendor,
  COALESCE(po.po_number, '') AS purchase_order_number,
  COALESCE(cur.code, 'CAD') AS currency_code,
  COALESCE(cur.symbol, 'CAD') AS currency_symbol,
  COALESCE(cur.icon, '') AS currency_icon,
  COALESCE(CAST(cur.rate AS REAL), 1) AS currency_rate,
  COALESCE(cur.rate_date, '') AS currency_rate_date,
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
  COALESCE(p4.given_name || ' ' || p4.surname, '') AS settler_name,
  COALESCE(p5.given_name || ' ' || p5.surname, '') AS creator_name,
  COALESCE(b.name, '') AS branch_name,
  COALESCE(po.vendor, '') AS po_vendor,
  COALESCE(pov.name, '') AS po_vendor_name,
  COALESCE(pov.alias, '') AS po_vendor_alias,
  COALESCE(po.job, '') AS po_job,
  COALESCE(poj.number, '') AS po_job_number,
  COALESCE(poj.description, '') AS po_job_description,
  COALESCE(po.division, '') AS po_division,
  COALESCE(pod.code, '') AS po_division_code,
  COALESCE(pod.name, '') AS po_division_name,
  COALESCE(po.category, '') AS po_category,
  COALESCE(poca.name, '') AS po_category_name,
  COALESCE(po.description, '') AS po_description,
  COALESCE(po.payment_type, '') AS po_payment_type,
  COALESCE(po.branch, '') AS po_branch,
  COALESCE(pob.name, '') AS po_branch_name,
  COALESCE(po.kind, '') AS po_kind,
  COALESCE(poek.name, '') AS po_kind_name,
  COALESCE(CAST(po.total AS REAL), 0) AS po_total,
  COALESCE(po.currency, '') AS po_currency,
  COALESCE(pocur.code, 'CAD') AS po_currency_code,
  COALESCE(pocur.symbol, 'CAD') AS po_currency_symbol,
  COALESCE(pocur.icon, '') AS po_currency_icon,
  COALESCE(CAST(pocur.rate AS REAL), 1) AS po_currency_rate,
  COALESCE(pocur.rate_date, '') AS po_currency_rate_date,
  COALESCE(po.uid, '') AS po_uid,
  COALESCE(p3.given_name || ' ' || p3.surname, '') AS po_uid_name
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
LEFT JOIN profiles p3 ON po.uid = p3.uid
LEFT JOIN profiles p4 ON e.settler = p4.uid
LEFT JOIN profiles p5 ON e.creator = p5.uid
LEFT JOIN currencies cur ON e.currency = cur.id
LEFT JOIN branches b ON e.branch = b.id
LEFT JOIN vendors pov ON po.vendor = pov.id
LEFT JOIN jobs poj ON po.job = poj.id
LEFT JOIN divisions pod ON po.division = pod.id
LEFT JOIN categories poca ON po.category = poca.id
LEFT JOIN expenditure_kinds poek ON po.kind = poek.id
LEFT JOIN currencies pocur ON po.currency = pocur.id
LEFT JOIN branches pob ON po.branch = pob.id
WHERE
  e.id = {:id}
