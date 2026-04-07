SELECT
  po.payment_type,
  COALESCE(j.number, '') AS job_number,
  COALESCE(d.code, '') AS division_code,
  COALESCE(b.code, '') AS branch_code,
  COALESCE(po.type, '') AS po_type,
  po.date AS record_date,
  CASE
    WHEN po.second_approval != '' AND po.second_approval > po.approved
    THEN po.second_approval
    ELSE po.approved
  END AS approval_date,
  po.total,
  COALESCE(cur.code, 'CAD') AS currency_code,
  po.po_number,
  po.description,
  COALESCE(v.name, '') AS vendor_name,
  COALESCE(owner.given_name || ' ' || owner.surname, '') AS employee,
  CASE
    WHEN po.second_approver != ''
    THEN COALESCE(sa.given_name || ' ' || sa.surname, '')
    ELSE COALESCE(ap.given_name || ' ' || ap.surname, '')
  END AS approved_by,
  po.status
FROM purchase_orders po
LEFT JOIN jobs j ON po.job = j.id
LEFT JOIN divisions d ON po.division = d.id
LEFT JOIN branches b ON po.branch = b.id
LEFT JOIN vendors v ON po.vendor = v.id
LEFT JOIN currencies cur ON po.currency = cur.id
LEFT JOIN profiles owner ON po.uid = owner.uid
LEFT JOIN profiles ap ON po.approver = ap.uid
LEFT JOIN profiles sa ON po.second_approver = sa.uid
WHERE po.status != 'Unapproved'
  AND po.po_number != ''
  -- Keep only the "normal" PO series by requiring the numeric block after the
  -- first hyphen to be under 5000, which excludes legacy/control numbers like
  -- YYMM-5XXX through YYMM-9XXX.
  AND (INSTR(po.po_number, '-') = 0 OR CAST(SUBSTR(po.po_number, INSTR(po.po_number, '-') + 1) AS INTEGER) < 5000)
  AND (po.approved != '' OR po.second_approval != '')
