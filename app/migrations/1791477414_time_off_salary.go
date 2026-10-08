package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const timeOffPreviousViewQuery = `SELECT 
  p.uid as id,
  CAST(CONCAT(p.surname, ', ', p.given_name) AS TEXT) AS name,
  p.manager AS manager_uid,
  CAST(CONCAT(mp.surname, ', ', mp.given_name) AS TEXT) AS manager,
  ap.opening_date,
  ap.opening_ov,
  ap.opening_op,
  CAST(COALESCE(SUM(CASE WHEN src.code = 'OV' THEN src.hours ELSE 0 END), 0) AS REAL) AS used_ov,
  CAST(COALESCE(SUM(CASE WHEN src.code = 'OP' THEN src.hours ELSE 0 END), 0) AS REAL) AS used_op,
  CAST(COALESCE(SUM(CASE WHEN src.code = 'OV' AND src.tsid != '' THEN src.hours ELSE 0 END), 0) AS REAL) AS timesheet_ov,
  CAST(COALESCE(SUM(CASE WHEN src.code = 'OP' AND src.tsid != '' THEN src.hours ELSE 0 END), 0) AS REAL) AS timesheet_op,
  CAST(MAX(CASE WHEN src.code = 'OV' THEN src.date END) AS TEXT) AS last_ov,
  CAST(MAX(CASE WHEN src.code = 'OP' THEN src.date END) AS TEXT) AS last_op
FROM 
  admin_profiles ap
JOIN
    profiles p ON p.uid = ap.uid AND ap.untracked_time_off = false AND ap.time_sheet_expected = true
LEFT JOIN
    profiles mp ON p.manager = mp.uid
LEFT JOIN (
  -- time_entries (committed only)
  SELECT 
  te.uid,
  te.hours,
  te.tsid,
  te.date,
  te.week_ending,
  tt.code
  FROM time_entries te
  JOIN time_types tt ON te.time_type = tt.id
  JOIN time_sheets ts ON te.tsid = ts.id
  WHERE te.tsid != '' AND ts.committed != '' AND tt.code IN ('OV','OP')
  UNION ALL
  -- time_amendments (committed only)
  SELECT 
  ta.uid,
  ta.hours,
  IFNULL(ta.tsid, '') AS tsid,
  ta.date,
  ta.committed_week_ending AS week_ending,
  tt2.code
  FROM time_amendments ta
  JOIN time_types tt2 ON ta.time_type = tt2.id
  WHERE ta.committed != '' AND tt2.code IN ('OV','OP')
) AS src 
  ON p.uid = src.uid
  AND src.week_ending > ap.opening_date
GROUP BY 
  p.uid, p.surname, p.given_name, p.manager, mp.surname, mp.given_name, ap.opening_date, ap.opening_ov, ap.opening_op`

// timeOffViewQueryWithSalary exposes admin_profiles.salary so the Time Off
// list can label each employee as hourly or salaried.
const timeOffViewQueryWithSalary = `SELECT 
  p.uid as id,
  CAST(CONCAT(p.surname, ', ', p.given_name) AS TEXT) AS name,
  p.manager AS manager_uid,
  CAST(CONCAT(mp.surname, ', ', mp.given_name) AS TEXT) AS manager,
  ap.opening_date,
  ap.opening_ov,
  ap.opening_op,
  ap.salary,
  CAST(COALESCE(SUM(CASE WHEN src.code = 'OV' THEN src.hours ELSE 0 END), 0) AS REAL) AS used_ov,
  CAST(COALESCE(SUM(CASE WHEN src.code = 'OP' THEN src.hours ELSE 0 END), 0) AS REAL) AS used_op,
  CAST(COALESCE(SUM(CASE WHEN src.code = 'OV' AND src.tsid != '' THEN src.hours ELSE 0 END), 0) AS REAL) AS timesheet_ov,
  CAST(COALESCE(SUM(CASE WHEN src.code = 'OP' AND src.tsid != '' THEN src.hours ELSE 0 END), 0) AS REAL) AS timesheet_op,
  CAST(MAX(CASE WHEN src.code = 'OV' THEN src.date END) AS TEXT) AS last_ov,
  CAST(MAX(CASE WHEN src.code = 'OP' THEN src.date END) AS TEXT) AS last_op
FROM 
  admin_profiles ap
JOIN
    profiles p ON p.uid = ap.uid AND ap.untracked_time_off = false AND ap.time_sheet_expected = true
LEFT JOIN
    profiles mp ON p.manager = mp.uid
LEFT JOIN (
  -- time_entries (committed only)
  SELECT 
  te.uid,
  te.hours,
  te.tsid,
  te.date,
  te.week_ending,
  tt.code
  FROM time_entries te
  JOIN time_types tt ON te.time_type = tt.id
  JOIN time_sheets ts ON te.tsid = ts.id
  WHERE te.tsid != '' AND ts.committed != '' AND tt.code IN ('OV','OP')
  UNION ALL
  -- time_amendments (committed only)
  SELECT 
  ta.uid,
  ta.hours,
  IFNULL(ta.tsid, '') AS tsid,
  ta.date,
  ta.committed_week_ending AS week_ending,
  tt2.code
  FROM time_amendments ta
  JOIN time_types tt2 ON ta.time_type = tt2.id
  WHERE ta.committed != '' AND tt2.code IN ('OV','OP')
) AS src 
  ON p.uid = src.uid
  AND src.week_ending > ap.opening_date
GROUP BY 
  p.uid, p.surname, p.given_name, p.manager, mp.surname, mp.given_name, ap.opening_date, ap.opening_ov, ap.opening_op, ap.salary`

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("6z8rcof9bkpzz1t")
		if err != nil {
			return err
		}

		collection.ViewQuery = timeOffViewQueryWithSalary
		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("6z8rcof9bkpzz1t")
		if err != nil {
			return err
		}

		collection.ViewQuery = timeOffPreviousViewQuery
		return app.Save(collection)
	})
}
