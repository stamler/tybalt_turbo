-- THIS IS QUITE SLOW and depends on the expense_mileage_totals and
-- expense_allowance_totals views.
SELECT ap.payroll_id,
  e2.payment_type "Acct/Visa/Exp",
  j.number "Job #",
  c.name Client,
  j.description "Job Description",
  e2.division "Div",
  -- get the last two digits of the date
  SUBSTRING(e2.date, 9, 2) Date,
  substr('  JanFebMarAprMayJunJulAugSepOctNovDec', strftime('%m', e2.date) * 3, 3) Month,
  SUBSTRING(e2.date, 1, 4) Year,
  e2.merged_total - ROUND(e2.merged_total * 13 / 113, 2) calculatedSubtotal,
  ROUND(e2.merged_total * 13 / 113, 2) calculatedOntarioHST,
  e2.merged_total Total,
  po.po_number "PO#",
  e2.merged_description Description,
  v.name Company,
  p.given_name || ' ' || p.surname Employee,
  p2.given_name || ' ' || p2.surname "Approved By"
FROM (
SELECT e.*,
  CASE
    WHEN e.payment_type = "Mileage" THEN m.mileage_total
    WHEN e.payment_type = "Allowance" OR e.payment_type = "Meals" THEN a.allowance_total
    ELSE e.total
  END merged_total,
  CASE
    WHEN e.payment_type = "Allowance" OR e.payment_type = "Meals" THEN a.allowance_description
    ELSE e.description
  END merged_description
FROM Expenses e
LEFT JOIN expense_mileage_totals m ON m.id = e.id
LEFT JOIN expense_allowance_totals a ON a.id = e.id
WHERE e.pay_period_ending = {:pay_period_ending}
) AS e2
LEFT JOIN admin_profiles ap ON ap.uid = e2.uid
LEFT JOIN profiles p ON p.uid = e2.uid
LEFT JOIN profiles p2 ON p2.uid = e2.approver
LEFT JOIN jobs j ON j.id = e2.job
LEFT JOIN clients c ON c.id = j.client
LEFT JOIN purchase_orders po ON po.id = e2.purchase_order
LEFT JOIN vendors v ON v.id = e2.vendor
ORDER BY e2.date