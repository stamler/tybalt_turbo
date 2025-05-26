/* The weekly time summary per employee 
derived from weeklyTimeSummaryPerEmployee.sql (tybalt)
 */
WITH base AS (
  SELECT (p.surname || ', ' || p.given_name) Employee,
    IFNULL(GROUP_CONCAT(DISTINCT D.code), '') divisions,
    SUM(CASE WHEN TE.job != '' THEN TE.hours ELSE 0 END) c,
    SUM(CASE WHEN TE.job = '' THEN TE.hours ELSE 0 END) nc,
    SUM(CASE WHEN TT.code IN ("R", "RT") AND TE.job = '' THEN TE.hours ELSE 0 END) nc_worked,
    SUM(CASE WHEN TT.code IN ("OP", "OV", "OH", "OB", "OS") THEN TE.hours ELSE 0 END) nc_unworked
  FROM time_entries TE
    LEFT JOIN time_sheets TS ON TE.tsid = TS.id
    LEFT Join profiles p ON TS.uid = p.uid
    LEFT JOIN time_types TT ON TE.time_type = TT.id
    LEFT JOIN divisions D ON TE.division = D.id
  WHERE TT.code != 'RB'
    AND TE.{:date_column} = {:date_column_value}
  GROUP BY TE.uid
  ORDER BY p.surname
)
SELECT Employee,
  divisions,
  c,
  nc,
  c + nc total,
  ROUND(100.0 * c / (c + nc - nc_unworked),1) percentChargeable,
  ROUND(100.0 * nc_worked / (c + nc),1) percentNCWorked,
  nc_worked,
  nc_unworked
FROM base;