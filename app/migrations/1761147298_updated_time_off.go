package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("6z8rcof9bkpzz1t")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"viewQuery": "SELECT \n  p.uid as id,\n  CAST(CONCAT(p.surname, ', ', p.given_name) AS TEXT) AS name,\n  p.manager AS manager_uid,\n  CAST(CONCAT(mp.surname, ', ', mp.given_name) AS TEXT) AS manager,\n  ap.opening_date,\n  ap.opening_ov,\n  ap.opening_op,\n  CAST(COALESCE(SUM(CASE WHEN src.code = 'OV' THEN src.hours ELSE 0 END), 0) AS REAL) AS used_ov,\n  CAST(COALESCE(SUM(CASE WHEN src.code = 'OP' THEN src.hours ELSE 0 END), 0) AS REAL) AS used_op,\n  CAST(COALESCE(SUM(CASE WHEN src.code = 'OV' AND src.tsid != '' THEN src.hours ELSE 0 END), 0) AS REAL) AS timesheet_ov,\n  CAST(COALESCE(SUM(CASE WHEN src.code = 'OP' AND src.tsid != '' THEN src.hours ELSE 0 END), 0) AS REAL) AS timesheet_op,\n  CAST(MAX(CASE WHEN src.code = 'OV' THEN src.date END) AS TEXT) AS last_ov,\n  CAST(MAX(CASE WHEN src.code = 'OP' THEN src.date END) AS TEXT) AS last_op\nFROM \n  profiles p\nLEFT JOIN\n    admin_profiles ap ON p.uid = ap.uid\nLEFT JOIN\n    profiles mp ON p.manager = mp.uid\nLEFT JOIN (\n  -- time_entries (committed only)\n  SELECT \n  te.uid,\n  te.hours,\n  te.tsid,\n  te.date,\n  te.week_ending,\n  tt.code\n  FROM time_entries te\n  JOIN time_types tt ON te.time_type = tt.id\n  JOIN time_sheets ts ON te.tsid = ts.id\n  WHERE te.tsid != '' AND ts.committed != '' AND tt.code IN ('OV','OP')\n  UNION ALL\n  -- time_amendments (committed only)\n  SELECT \n  ta.uid,\n  ta.hours,\n  IFNULL(ta.tsid, '') AS tsid,\n  ta.date,\n  ta.committed_week_ending AS week_ending,\n  tt2.code\n  FROM time_amendments ta\n  JOIN time_types tt2 ON ta.time_type = tt2.id\n  WHERE ta.committed != '' AND tt2.code IN ('OV','OP')\n) AS src \n  ON p.uid = src.uid\n  AND src.week_ending > ap.opening_date\n  AND ap.untracked_time_off = false\nGROUP BY \n  p.uid, p.surname, p.given_name, p.manager, mp.surname, mp.given_name, ap.opening_date, ap.opening_ov, ap.opening_op"
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_6Em1")

		// remove field
		collection.Fields.RemoveById("_clone_fl0C")

		// remove field
		collection.Fields.RemoveById("_clone_Mqxk")

		// remove field
		collection.Fields.RemoveById("_clone_qcXR")

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(2, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_OtuA",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "manager_uid",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(4, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_Ka62",
			"max": 0,
			"min": 0,
			"name": "opening_date",
			"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(5, []byte(`{
			"hidden": false,
			"id": "_clone_mTxU",
			"max": 200,
			"min": 0,
			"name": "opening_ov",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(6, []byte(`{
			"hidden": false,
			"id": "_clone_hFDQ",
			"max": 332,
			"min": 0,
			"name": "opening_op",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("6z8rcof9bkpzz1t")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"viewQuery": "SELECT \n  p.uid as id,\n  CAST(CONCAT(p.surname, ', ', p.given_name) AS TEXT) AS name,\n  p.manager AS manager_uid,\n  CAST(CONCAT(mp.surname, ', ', mp.given_name) AS TEXT) AS manager,\n  ap.opening_date,\n  ap.opening_ov,\n  ap.opening_op,\n  CAST(COALESCE(SUM(CASE WHEN src.code = 'OV' THEN src.hours ELSE 0 END), 0) AS REAL) AS used_ov,\n  CAST(COALESCE(SUM(CASE WHEN src.code = 'OP' THEN src.hours ELSE 0 END), 0) AS REAL) AS used_op,\n  CAST(COALESCE(SUM(CASE WHEN src.code = 'OV' AND src.tsid != '' THEN src.hours ELSE 0 END), 0) AS REAL) AS timesheet_ov,\n  CAST(COALESCE(SUM(CASE WHEN src.code = 'OP' AND src.tsid != '' THEN src.hours ELSE 0 END), 0) AS REAL) AS timesheet_op,\n  CAST(MAX(CASE WHEN src.code = 'OV' THEN src.date END) AS TEXT) AS last_ov,\n  CAST(MAX(CASE WHEN src.code = 'OP' THEN src.date END) AS TEXT) AS last_op\nFROM \n  profiles p\nLEFT JOIN\n    admin_profiles ap ON p.uid = ap.uid\nLEFT JOIN\n    profiles mp ON p.manager = mp.uid\nLEFT JOIN (\n  -- time_entries (committed only)\n  SELECT \n    te.uid,\n    te.hours,\n    te.tsid,\n    te.date,\n    te.week_ending,\n    tt.code\n  FROM time_entries te\n  LEFT JOIN time_types tt ON te.time_type = tt.id\n  LEFT JOIN time_sheets ts ON te.tsid = ts.id\n  WHERE te.tsid != '' AND ts.committed != ''\n  UNION ALL\n  -- time_amendments (committed only)\n  SELECT\n    ta.uid,\n    ta.hours,\n    IFNULL(ta.tsid, '') AS tsid,\n    ta.date,\n    ta.committed_week_ending AS week_ending,\n    tt2.code\n  FROM time_amendments ta\n  LEFT JOIN time_types tt2 ON ta.time_type = tt2.id\n  WHERE ta.committed != ''\n) AS src ON p.uid = src.uid\nWHERE \n  src.week_ending > ap.opening_date\n  AND ap.untracked_time_off = false\nGROUP BY \n  p.uid, p.surname, p.given_name, p.manager, mp.surname, mp.given_name, ap.opening_date, ap.opening_ov, ap.opening_op"
		}`), &collection); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(2, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_6Em1",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "manager_uid",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(4, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_fl0C",
			"max": 0,
			"min": 0,
			"name": "opening_date",
			"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(5, []byte(`{
			"hidden": false,
			"id": "_clone_Mqxk",
			"max": 200,
			"min": 0,
			"name": "opening_ov",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(6, []byte(`{
			"hidden": false,
			"id": "_clone_qcXR",
			"max": 332,
			"min": 0,
			"name": "opening_op",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_OtuA")

		// remove field
		collection.Fields.RemoveById("_clone_Ka62")

		// remove field
		collection.Fields.RemoveById("_clone_mTxU")

		// remove field
		collection.Fields.RemoveById("_clone_hFDQ")

		return app.Save(collection)
	})
}
