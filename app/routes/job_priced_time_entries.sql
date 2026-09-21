-- Shared pricing rules for the staff and division summaries.
-- Use the assigned revision even when it is inactive. Rate changes therefore
-- recalculate past values. Keep the existing selection of committed and
-- uncommitted entries; amendments and meals are not part of the value.
WITH entry_rates AS (
  SELECT te.uid, te.division, te.hours, te.meals_hours, j.number,
    COALESCE(rs.name, '') AS rate_sheet_name, rs.revision AS rate_sheet_revision,
    rse.rate AS sheet_rate,
    -- A zero or missing employee default cannot supply a dollar value.
    CASE WHEN ap.default_charge_out_rate > 0 THEN ap.default_charge_out_rate END AS default_rate
  FROM time_entries te
  LEFT JOIN jobs j ON te.job = j.id
  LEFT JOIN rate_sheets rs ON rs.id = j.rate_sheet
  LEFT JOIN rate_sheet_entries rse
    ON rse.rate_sheet = j.rate_sheet AND rse.role = te.role
  LEFT JOIN admin_profiles ap ON ap.uid = te.uid
  WHERE te.job = {:job_id}
    AND te.date >= {:start_date}
    AND te.date <= {:end_date}
), priced_entries AS (
  SELECT *,
    hours * COALESCE(sheet_rate, default_rate) AS value,
    CASE WHEN sheet_rate IS NULL AND default_rate IS NOT NULL THEN hours ELSE 0 END AS estimated_hours,
    CASE WHEN COALESCE(sheet_rate, default_rate) IS NULL THEN hours ELSE 0 END AS unpriced_hours
  FROM entry_rates
)
