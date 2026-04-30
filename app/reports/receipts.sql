-- This query lists the attachments for the payroll for the given pay period. It
-- is much simpler than payroll_expenses.sql because Mileage, Allowance, and
-- Meals payment_types NEVER have attachments, so we can ignore the complex
-- joins used to get the totals for those payment_types and just find committed
-- expenses with attachments for the specific pay period.
WITH expenses_having_attachments AS (
SELECT
  e.id,
  CASE WHEN ed.id IS NOT NULL THEN 'pbc_2089657321' ELSE 'o1vpz1mm7qsfoyy' END AS collection_id,
  CASE WHEN ed.id IS NOT NULL THEN ed.id ELSE e.id END AS attachment_record_id,
  e.payment_type,
  p.surname,
  p.given_name,
  -- get the last two digits of the date
  SUBSTRING(e.date, 1, 4) Year,
  substr('  JanFebMarAprMayJunJulAugSepOctNovDec', strftime('%m', e.date) * 3, 3) Month,
  SUBSTRING(e.date, 9, 2) Date,
  e.total,
  COALESCE(ed.attachment, e.attachment) AS attachment,
  COALESCE(ed.attachment_hash, e.attachment_hash) AS attachment_hash
FROM Expenses e
LEFT JOIN admin_profiles ap ON ap.uid = e.uid
LEFT JOIN profiles p ON p.uid = e.uid
LEFT JOIN expense_documents ed ON e.attachment_document = ed.id
WHERE e.{:date_column} = {:date_column_value}
AND e.committed != ''
AND COALESCE(ed.attachment, e.attachment) != ''
)

SELECT 
  a.id AS id,
  a.attachment AS filename,
  a.payment_type || '-' || a.surname || ',' || a.given_name || '-' || a.Year || '_' || a.Month || '_' || a.Date || '-' || a.total || '-' || a.attachment AS zip_filename,
  a.collection_id AS collection_id,
  a.attachment_record_id || '/' || a.attachment AS source_path,
  a.attachment_hash AS sha256
FROM expenses_having_attachments a
