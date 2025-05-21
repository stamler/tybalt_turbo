-- FORMAT
-- payment_type-surname,given_name-YYYY_MMM_DD-total-filename.extension
-- This query is a modified version of payroll_expenses.sql
WITH expenses_having_attachments AS (
SELECT
  e2.id,
  e2.payment_type,
  p.surname,
  p.given_name,
  -- get the last two digits of the date
  SUBSTRING(e2.date, 1, 4) Year,
  substr('  JanFebMarAprMayJunJulAugSepOctNovDec', strftime('%m', e2.date) * 3, 3) Month,
  SUBSTRING(e2.date, 9, 2) Date,
  e2.merged_total total,
  e2.attachment
FROM (
SELECT e.id,
  e.uid,
  e.date,
  e.division,
  e.description,
  e.payment_type,
  e.attachment,
  e.approver,
  e.job,
  e.category,
  e.pay_period_ending,
  e.allowance_types,
  e.distance,
  e.cc_last_4_digits,
  e.purchase_order,
  e.vendor,
  CAST(CASE
    WHEN e.payment_type = 'Mileage' THEN m.mileage_total
    WHEN e.payment_type = 'Allowance' OR e.payment_type = 'Meals' THEN a.allowance_total
    ELSE e.total
  END AS REAL) merged_total,
  CASE
    WHEN e.payment_type = 'Allowance' OR e.payment_type = 'Meals' THEN a.allowance_description
    ELSE e.description
  END merged_description
FROM Expenses e
LEFT JOIN (
  /* This is the exact query from the expense_mileage_totals view except that it
  includes a filter on pay_period_ending which massively speeds up the entire
  report by pre-filtering the data that must be processed. It calculates the
  mileage_total, that is the amount of each expense in dollars based on the
  distance claimed and the mileage tier for each claim. If the claim spans two
  mileage tiers, this is accounted for and the calculation is performed
  piecewise for each tier. This result can be LEFT JOINed to Expenses ON id =
  id. */
  -- 1) Explode + window up front
  WITH rates_expanded AS (
    SELECT
      m.effective_date,
      CAST(t.key   AS INTEGER) AS tier_lower,
      LEAD(CAST(t.key AS INTEGER)) OVER (
        PARTITION BY m.effective_date
        ORDER BY CAST(t.key AS INTEGER)
      ) AS tier_upper,
      CAST(t.value AS REAL)    AS tier_rate

    FROM expense_rates m
    CROSS JOIN json_each(m.mileage) AS t
  ),

  -- 2) Calculate cumulative mileage for relevant users, then filter to pay period
  CumulativeMileage AS (
    SELECT
      e.id,
      e.uid,
      e.date,
      e.pay_period_ending, -- Keep this to filter on later
      (
        SELECT MAX(r.date)
        FROM mileage_reset_dates r
        WHERE r.date <= e.date
      ) AS reset_mileage_date,
      e.distance,
      -- interval end = cumulative distance
      SUM(e.distance) OVER (
        PARTITION BY e.uid, (
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
    WHERE e.payment_type = 'Mileage'
      AND e.committed != ''
      AND e.uid IN (
        SELECT DISTINCT ee.uid
        FROM expenses ee
        WHERE ee.payment_type = 'Mileage' AND ee.pay_period_ending = {:pay_period_ending}
      )
  ),

  base AS (
    SELECT
      cm.id,
      cm.uid,
      cm.date,
      cm.reset_mileage_date,
      cm.distance,
      cm.end_distance,
      cm.effective_date
    FROM CumulativeMileage cm
    WHERE cm.pay_period_ending = {:pay_period_ending}
  ),

  /* 
  3) Join each expense to its tiers, filtering to only those that overlap We are
  pairing each expense’s [start_distance, end_distance) interval with only those
  tier intervals [tier_lower, tier_upper) that intersect it. By exploding each
  tier into its own row, an expense whose kilometres cross a tier boundary will
  join to two (or more, in theory) tier‐rows. Thus we can expect that the number
  of rows in overlaps can be more than the number of rows in expenses.
  */
  overlaps AS (
    SELECT
      b.id,
      b.end_distance - b.distance AS start_distance,
      b.end_distance,
      r.tier_lower,
      COALESCE(r.tier_upper, 1e9) AS tier_upper,
      r.tier_rate
    FROM base b
    JOIN rates_expanded r
      ON r.effective_date = b.effective_date
    WHERE b.end_distance > r.tier_lower
      AND (r.tier_upper IS NULL OR (b.end_distance - b.distance) < r.tier_upper)
  ),

  /*
  4) Compute overlap length per tier
  for each expense × tier row, compute how many kilometres of the expense fall into that tier by:
    1. Determining the overlap interval between the expense’s [start_distance, end_distance)
      and the tier’s [tier_lower, tier_upper) interval:
        • overlap_start = max(start_distance, tier_lower)
        • overlap_end   = tier_upper IS NULL
                            ? end_distance
                            : min(end_distance, tier_upper)
    2. Calculating overlap_km = max(0, overlap_end − overlap_start)
        • Ensures negative values (no overlap) are clipped to zero
    3. Carrying along tier_rate so we can later multiply overlap_km × tier_rate
  */
  tier_calcs AS (
    SELECT
      id,
      max(0,
        min(end_distance, tier_upper)
        - max(start_distance, tier_lower)
      ) AS overlap_km,
      tier_rate
    FROM overlaps
  )

  -- 5) Sum up reimbursement per expense
  SELECT
    b.id,
    b.uid,
    b.date,
    b.reset_mileage_date,
    b.distance,
    b.end_distance AS cumulative,
    b.effective_date,
    ROUND(COALESCE(
      -- sum up this expense’s (overlap × rate) directly
      (SELECT SUM(overlap_km * tier_rate)
      FROM tier_calcs tc
      WHERE tc.id = b.id),
      0
    ), 2) AS mileage_total
  FROM base b
) m ON m.id = e.id
LEFT JOIN (
  /* This is the exact query from the expense_allowance_totals view except that
  it includes a filter on pay_period_ending which massively speeds up the entire
  report by pre-filtering the data that must be processed. It calculates the
  allowance_total, that is the amount of each expense in dollars based on the
  allowances claimed. This result can be LEFT JOINed to Expenses ON id = id.*/
  SELECT e.id, 
    e.date, 
    r.effective_date allowance_rates_effective_date, 
    e.payment_type,
    e.allowance_types,
    r.breakfast breakfast_rate,
    r.lunch lunch_rate,
    r.dinner dinner_rate,
    r.lodging lodging_rate,
    r.mileage,
    -- sum up each rate only if the corresponding token appears in the JSON text
    ((CASE WHEN e.allowance_types LIKE '%"Breakfast"%'   THEN r.breakfast ELSE 0 END)
    + (CASE WHEN e.allowance_types LIKE '%"Lunch"%'     THEN r.lunch     ELSE 0 END)
    + (CASE WHEN e.allowance_types LIKE '%"Dinner"%'    THEN r.dinner    ELSE 0 END)
    + (CASE WHEN e.allowance_types LIKE '%"Lodging"%'   THEN r.lodging   ELSE 0 END)
    )AS allowance_total,
    -- build the description in the fixed order
    RTRIM(
      (CASE WHEN e.allowance_types LIKE '%"Breakfast"%' THEN 'Breakfast ' ELSE '' END) ||
      (CASE WHEN e.allowance_types LIKE '%"Lunch"%'     THEN 'Lunch '     ELSE '' END) ||
      (CASE WHEN e.allowance_types LIKE '%"Dinner"%'    THEN 'Dinner '    ELSE '' END) ||
      (CASE WHEN e.allowance_types LIKE '%"Lodging"%'   THEN 'Lodging '   ELSE '' END)
    ) AS allowance_description
  FROM expenses e 
  LEFT JOIN expense_rates r ON ((r.effective_date = (SELECT MAX(i.effective_date) FROM expense_rates i WHERE (i.effective_date <= e.date))))
  WHERE e.payment_type IN ('Allowance','Meals')
  AND e.pay_period_ending = {:pay_period_ending}
  AND e.committed != ''
) a ON a.id = e.id
WHERE e.pay_period_ending = {:pay_period_ending}
AND e.committed != ''
AND e.attachment != ''
) AS e2
LEFT JOIN admin_profiles ap ON ap.uid = e2.uid
LEFT JOIN profiles p ON p.uid = e2.uid
ORDER BY e2.date, ap.payroll_id, e2.merged_total
)

SELECT 
  a.payment_type || '-' || a.surname || ',' || a.given_name || '-' || a.Year || '_' || a.Month || '_' || a.Date || '-' || a.total || '-' || a.attachment AS filename,
  a.id || '/' || a.attachment AS source_path
FROM expenses_having_attachments a