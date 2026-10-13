-- Lists the caller's committed expenses that are repaid to them on payroll,
-- one row per expense, bucketed by pay_period_ending.
--
-- The payroll expense report (app/reports/expenses.sql) includes every
-- committed expense, but only these payment types are paid to the employee:
-- Expense, PersonalReimbursement, Allowance, Mileage, and the legacy Meals type
-- (the pre-2021 name for Allowance). OnAccount, CorporateCreditCard, FuelCard,
-- and FuelOnAccount were paid by the company and are excluded.
--
-- amount is computed exactly as the payroll expense report computes Total so
-- the two always agree. In particular Mileage is NOT read from e.total, which
-- is fixed at commit time and can differ from the report when mileage
-- expenses spanning a tier boundary are committed out of date order.
--
-- An empty pay_period_ending parameter lists every pay period.
WITH rates_expanded AS (
  SELECT
    m.effective_date,
    CAST(t.key AS INTEGER) AS tier_lower,
    LEAD(CAST(t.key AS INTEGER)) OVER (
      PARTITION BY m.effective_date
      ORDER BY CAST(t.key AS INTEGER)
    ) AS tier_upper,
    CAST(t.value AS REAL) AS tier_rate
  FROM expense_rates m
  CROSS JOIN json_each(m.mileage) AS t
),

-- Cumulative distance must be computed across all of the caller's committed
-- mileage (not just the requested pay period) so tiers line up with payroll.
mileage_base AS (
  SELECT
    e.id,
    e.distance,
    SUM(e.distance) OVER (
      PARTITION BY (
        SELECT MAX(r.date)
        FROM mileage_reset_dates r
        WHERE r.date <= e.date
      )
      ORDER BY e.date
    ) AS end_distance,
    (
      SELECT MAX(m.effective_date)
      FROM expense_rates m
      WHERE m.effective_date <= e.date
    ) AS effective_date
  FROM expenses e
  WHERE e.uid = {:uid}
    AND e.payment_type = 'Mileage'
    AND e.committed != ''
),

mileage_tier_calcs AS (
  SELECT
    b.id,
    max(0,
      min(b.end_distance, COALESCE(r.tier_upper, 1e9))
      - max(b.end_distance - b.distance, r.tier_lower)
    ) AS overlap_km,
    r.tier_rate
  FROM mileage_base b
  JOIN rates_expanded r
    ON r.effective_date = b.effective_date
  WHERE b.end_distance > r.tier_lower
    AND (r.tier_upper IS NULL OR (b.end_distance - b.distance) < r.tier_upper)
),

mileage_totals AS (
  SELECT
    b.id,
    ROUND(COALESCE(
      (SELECT SUM(tc.overlap_km * tc.tier_rate) FROM mileage_tier_calcs tc WHERE tc.id = b.id),
      0
    ), 2) AS mileage_total
  FROM mileage_base b
)

SELECT
  e.id,
  e.date,
  e.pay_period_ending,
  -- Payday is always the Friday 13 days after the Saturday pay period ending.
  date(e.pay_period_ending, '+13 days') AS pay_date,
  e.payment_type,
  CASE
    WHEN e.payment_type IN ('Allowance', 'Meals') THEN RTRIM(
      (CASE WHEN e.allowance_types LIKE '%"Breakfast"%' THEN 'Breakfast, ' ELSE '' END) ||
      (CASE WHEN e.allowance_types LIKE '%"Lunch"%'     THEN 'Lunch, '     ELSE '' END) ||
      (CASE WHEN e.allowance_types LIKE '%"Dinner"%'    THEN 'Dinner, '    ELSE '' END) ||
      (CASE WHEN e.allowance_types LIKE '%"Lodging"%'   THEN 'Lodging, '   ELSE '' END),
      ', '
    )
    ELSE e.description
  END AS description,
  CAST(COALESCE(e.distance, 0) AS REAL) AS distance,
  COALESCE(j.number, '') AS job_number,
  COALESCE(j.description, '') AS job_description,
  COALESCE(cl.name, '') AS client_name,
  COALESCE(cur.code, 'CAD') AS currency_code,
  CASE
    WHEN COALESCE(cur.code, 'CAD') = 'CAD' THEN 0
    ELSE CAST(e.total AS REAL)
  END AS foreign_currency_total,
  CAST(CASE
    WHEN e.payment_type = 'Mileage' THEN COALESCE(mt.mileage_total, 0)
    WHEN e.payment_type IN ('Allowance', 'Meals') THEN
      (CASE WHEN e.allowance_types LIKE '%"Breakfast"%' THEN r.breakfast ELSE 0 END)
      + (CASE WHEN e.allowance_types LIKE '%"Lunch"%'   THEN r.lunch     ELSE 0 END)
      + (CASE WHEN e.allowance_types LIKE '%"Dinner"%'  THEN r.dinner    ELSE 0 END)
      + (CASE WHEN e.allowance_types LIKE '%"Lodging"%' THEN r.lodging   ELSE 0 END)
    WHEN COALESCE(cur.code, 'CAD') != 'CAD' THEN COALESCE(NULLIF(e.settled_total, 0), e.total)
    ELSE e.total
  END AS REAL) AS amount
FROM expenses e
LEFT JOIN mileage_totals mt ON mt.id = e.id
LEFT JOIN expense_rates r
  ON e.payment_type IN ('Allowance', 'Meals')
  AND r.effective_date = (SELECT MAX(i.effective_date) FROM expense_rates i WHERE i.effective_date <= e.date)
LEFT JOIN currencies cur ON cur.id = e.currency
LEFT JOIN jobs j ON j.id = e.job
LEFT JOIN clients cl ON cl.id = j.client
WHERE e.uid = {:uid}
  AND e.committed != ''
  AND e.pay_period_ending != ''
  AND e.payment_type IN ('Expense', 'PersonalReimbursement', 'Allowance', 'Meals', 'Mileage')
  AND ({:pay_period_ending} = '' OR e.pay_period_ending = {:pay_period_ending})
