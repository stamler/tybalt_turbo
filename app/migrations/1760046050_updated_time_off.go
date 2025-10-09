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
			"viewQuery": "SELECT \n    p.uid as id,\n    CAST(CONCAT(p.surname, ', ', p.given_name) AS TEXT) AS name,\n    p.manager AS manager_uid,\n    CAST(CONCAT(mp.surname, ', ', mp.given_name) AS TEXT) AS manager,\n    ap.opening_date,\n    ap.opening_ov,\n    ap.opening_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_op,\n    CAST(MAX(CASE WHEN tt.code = 'OV' THEN te.date END) AS TEXT) AS last_ov,\n    CAST(MAX(CASE WHEN tt.code = 'OP' THEN te.date END) AS TEXT) AS last_op\nFROM \n    profiles p\nLEFT JOIN \n    admin_profiles ap ON p.uid = ap.uid\nLEFT JOIN \n    profiles mp ON p.manager = mp.uid\nLEFT JOIN \n    time_entries te ON p.uid = te.uid\nLEFT JOIN \n    time_types tt ON te.time_type = tt.id\nWHERE \n    te.week_ending > ap.opening_date\nGROUP BY \n    p.uid, p.surname, p.given_name, p.manager, mp.surname, mp.given_name, ap.opening_date, ap.opening_ov, ap.opening_op"
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_csH3")

		// remove field
		collection.Fields.RemoveById("_clone_BloK")

		// remove field
		collection.Fields.RemoveById("_clone_D2dh")

		// remove field
		collection.Fields.RemoveById("_clone_4AE0")

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(2, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_WDHA",
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
			"id": "_clone_fSWg",
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
			"id": "_clone_tqO9",
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
			"id": "_clone_orhh",
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
			"viewQuery": "SELECT \n    p.uid as id,\n    CAST(CONCAT(p.surname, ', ', p.given_name) AS TEXT) AS name,\n    p.manager AS manager_uid,\n    CAST(CONCAT(mp.surname, ', ', mp.given_name) AS TEXT) AS manager,\n    ap.opening_date,\n    ap.opening_ov,\n    ap.opening_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' THEN te.hours ELSE 0 END), 0) AS REAL) AS used_op,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OV' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_ov,\n    CAST(COALESCE(SUM(CASE WHEN tt.code = 'OP' AND te.tsid != '' THEN te.hours ELSE 0 END), 0) AS REAL) AS timesheet_op,\n    CAST(MAX(CASE WHEN tt.code = 'OV' THEN te.date END) AS TEXT) AS last_ov,\n    CAST(MAX(CASE WHEN tt.code = 'OP' THEN te.date END) AS TEXT) AS last_op\nFROM \n    profiles p\nLEFT JOIN \n    admin_profiles ap ON p.uid = ap.uid\nLEFT JOIN \n    profiles mp ON p.manager = mp.uid\nLEFT JOIN \n    time_entries te ON p.uid = te.uid\nLEFT JOIN \n    time_types tt ON te.time_type = tt.id\nWHERE \n    te.week_ending >= ap.opening_date\nGROUP BY \n    p.uid, p.surname, p.given_name, p.manager, mp.surname, mp.given_name, ap.opening_date, ap.opening_ov, ap.opening_op"
		}`), &collection); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(2, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_csH3",
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
			"id": "_clone_BloK",
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
			"id": "_clone_D2dh",
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
			"id": "_clone_4AE0",
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
		collection.Fields.RemoveById("_clone_WDHA")

		// remove field
		collection.Fields.RemoveById("_clone_fSWg")

		// remove field
		collection.Fields.RemoveById("_clone_tqO9")

		// remove field
		collection.Fields.RemoveById("_clone_orhh")

		return app.Save(collection)
	})
}
