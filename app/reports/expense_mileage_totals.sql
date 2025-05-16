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

-- 2) Base: compute cumulative & interval boundaries
base AS (
  SELECT
    e.id,
    e.uid,
    e.date,
    r.date AS reset_mileage_date,
    -- interval start = cumulative distance - this row's distance
    SUM(e.distance) OVER (
      PARTITION BY e.uid, r.date
      ORDER BY e.date
    ) - e.distance AS start_distance,
    e.distance,
    -- interval end = cumulative distance
    SUM(e.distance) OVER (
      PARTITION BY e.uid, r.date
      ORDER BY e.date
    ) AS end_distance,
    m.effective_date
  FROM expenses e
  INNER JOIN mileage_reset_dates r
    ON r.date = (
      SELECT MAX(r2.date)
      FROM mileage_reset_dates r2
      WHERE r2.date <= e.date
    )
  INNER JOIN expense_rates m
    ON m.effective_date = (
      SELECT MAX(m2.effective_date)
      FROM expense_rates m2
      WHERE m2.effective_date <= e.date
    )
  WHERE e.payment_type = 'Mileage'
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
    b.start_distance,
    b.end_distance,
    r.tier_lower,
    r.tier_upper,
    r.tier_rate
  FROM base b
  JOIN rates_expanded r
    ON r.effective_date = b.effective_date
  WHERE b.end_distance > r.tier_lower
    AND (r.tier_upper IS NULL OR b.start_distance < r.tier_upper)
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
    CASE
      WHEN tier_upper IS NULL THEN
        max(0, end_distance - max(start_distance, tier_lower))
      ELSE
        max(0,
          min(end_distance, tier_upper)
          - max(start_distance, tier_lower)
        )
    END AS overlap_km,
    tier_rate
  FROM overlaps
)

-- 5) Sum up reimbursement per expense
SELECT
  b.*,
  COALESCE(
    -- sum up this expense’s (overlap × rate) directly
    (SELECT SUM(overlap_km * tier_rate)
     FROM tier_calcs tc
     WHERE tc.id = b.id),
    0
  ) AS mileage_total
FROM base b
ORDER BY b.date;
