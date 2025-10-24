package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_3637308620")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"viewQuery": "SELECT \n  ta.id,\n  ta.division,\n  ta.uid,\n  ta.hours,\n  ta.description,\n  ta.time_type,\n  ta.meals_hours,\n  ta.job,\n  ta.work_record,\n  ta.payout_request_amount,\n  ta.date,\n  ta.week_ending,\n  ta.tsid,\n  ta.category,\n  ta.creator,\n  ta.committed,\n  ta.committer,\n  ta.committed_week_ending,\n  ta.skip_tsid_check,\n  (p0.given_name || ' ' || p0.surname) as uid_name,\n  (p1.given_name || ' ' || p1.surname) as creator_name,\n  (p2.given_name || ' ' || p2.surname) as committer_name,\n  tt.code as time_type_code,\n  tt.name as time_type_name,\n  j.number as job_number,\n  j.description as job_description,\n  c.name as category_name,\n  d.code as division_code,\n  d.name as division_name\nFROM time_amendments ta\nLEFT JOIN time_types tt ON ta.time_type = tt.id\nLEFT JOIN jobs j ON ta.job = j.id\nLEFT JOIN categories c ON ta.category = c.id\nLEFT JOIN divisions d ON ta.division = d.id\nLEFT JOIN profiles p0 ON ta.uid = p0.uid\nLEFT JOIN profiles p1 ON ta.creator = p1.uid\nLEFT JOIN profiles p2 ON ta.committer = p2.uid;"
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_1Cz2")

		// remove field
		collection.Fields.RemoveById("_clone_wrgH")

		// remove field
		collection.Fields.RemoveById("_clone_lgdK")

		// remove field
		collection.Fields.RemoveById("_clone_bB6v")

		// remove field
		collection.Fields.RemoveById("_clone_Ai7p")

		// remove field
		collection.Fields.RemoveById("_clone_BxVm")

		// remove field
		collection.Fields.RemoveById("_clone_hkVy")

		// remove field
		collection.Fields.RemoveById("_clone_Awre")

		// remove field
		collection.Fields.RemoveById("_clone_KHd6")

		// remove field
		collection.Fields.RemoveById("_clone_ckRq")

		// remove field
		collection.Fields.RemoveById("_clone_KTYC")

		// remove field
		collection.Fields.RemoveById("_clone_gIeX")

		// remove field
		collection.Fields.RemoveById("_clone_Isy2")

		// remove field
		collection.Fields.RemoveById("_clone_WDS9")

		// remove field
		collection.Fields.RemoveById("_clone_7hGz")

		// remove field
		collection.Fields.RemoveById("_clone_M9iZ")

		// remove field
		collection.Fields.RemoveById("_clone_rcp3")

		// remove field
		collection.Fields.RemoveById("_clone_RzaS")

		// remove field
		collection.Fields.RemoveById("_clone_wFnX")

		// remove field
		collection.Fields.RemoveById("_clone_GV48")

		// remove field
		collection.Fields.RemoveById("_clone_8pQt")

		// remove field
		collection.Fields.RemoveById("_clone_3k5l")

		// remove field
		collection.Fields.RemoveById("_clone_Jxty")

		// remove field
		collection.Fields.RemoveById("_clone_RCB6")

		// remove field
		collection.Fields.RemoveById("_clone_mP6o")

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"cascadeDelete": false,
			"collectionId": "3esdddggow6dykr",
			"hidden": false,
			"id": "_clone_mAcc",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "division",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(2, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_UF8z",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "uid",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(3, []byte(`{
			"hidden": false,
			"id": "_clone_RXRl",
			"max": 18,
			"min": -18,
			"name": "hours",
			"onlyInt": false,
			"presentable": false,
			"required": true,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(4, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_822i",
			"max": 0,
			"min": 0,
			"name": "description",
			"pattern": "",
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
			"cascadeDelete": false,
			"collectionId": "cnqv0wm8hly7r3n",
			"hidden": false,
			"id": "_clone_kCQq",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "time_type",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(6, []byte(`{
			"hidden": false,
			"id": "_clone_N5S7",
			"max": 3,
			"min": 0,
			"name": "meals_hours",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(7, []byte(`{
			"cascadeDelete": false,
			"collectionId": "yovqzrnnomp0lkx",
			"hidden": false,
			"id": "_clone_Mstw",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "job",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(8, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_8R9b",
			"max": 0,
			"min": 0,
			"name": "work_record",
			"pattern": "^[FKQ][0-9]{2}-[0-9]{3,}(-[0-9]+)?$",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(9, []byte(`{
			"hidden": false,
			"id": "_clone_BYwu",
			"max": null,
			"min": null,
			"name": "payout_request_amount",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(10, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_2Cax",
			"max": 0,
			"min": 0,
			"name": "date",
			"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
			"presentable": true,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(11, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_fwrZ",
			"max": 0,
			"min": 0,
			"name": "week_ending",
			"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
			"presentable": true,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(12, []byte(`{
			"cascadeDelete": false,
			"collectionId": "fpri53nrr2xgoov",
			"hidden": false,
			"id": "_clone_AxzD",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "tsid",
			"presentable": true,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(13, []byte(`{
			"cascadeDelete": false,
			"collectionId": "nrwhbwowokwu6cr",
			"hidden": false,
			"id": "_clone_a8YZ",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "category",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(14, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_NQOl",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "creator",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(15, []byte(`{
			"hidden": false,
			"id": "_clone_x3pm",
			"max": "",
			"min": "",
			"name": "committed",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "date"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(16, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_NCSW",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "committer",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(17, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_skE4",
			"max": 0,
			"min": 0,
			"name": "committed_week_ending",
			"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
			"presentable": true,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(18, []byte(`{
			"hidden": false,
			"id": "_clone_rnA1",
			"name": "skip_tsid_check",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(22, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_aopl",
			"max": 0,
			"min": 1,
			"name": "time_type_code",
			"pattern": "",
			"presentable": true,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(23, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_dF2Y",
			"max": 0,
			"min": 2,
			"name": "time_type_name",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(24, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_1gJb",
			"max": 0,
			"min": 0,
			"name": "job_number",
			"pattern": "^(P)?[0-9]{2}-[0-9]{3,4}(-[0-9]{1,2})?(-[0-9])?$",
			"presentable": true,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(25, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_iT4S",
			"max": 0,
			"min": 3,
			"name": "job_description",
			"pattern": "",
			"presentable": true,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(26, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_Cq0n",
			"max": 0,
			"min": 3,
			"name": "category_name",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(27, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_xBYK",
			"max": 0,
			"min": 1,
			"name": "division_code",
			"pattern": "",
			"presentable": true,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(28, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_Ab8t",
			"max": 0,
			"min": 2,
			"name": "division_name",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_3637308620")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"viewQuery": "SELECT \n  ta.id,\n  ta.division,\n  ta.uid,\n  ta.hours,\n  ta.description,\n  ta.time_type,\n  ta.meals_hours,\n  ta.job,\n  ta.work_record,\n  ta.payout_request_amount,\n  ta.date,\n  ta.week_ending,\n  ta.tsid,\n  ta.category,\n  ta.creator,\n  ta.committed,\n  ta.committer,\n  ta.committed_week_ending,\n  ta.skip_tsid_check,\n  (p0.given_name || ' ' || p0.surname) as uid_name,\n  (p1.given_name || ' ' || p1.surname) as creator_name,\n  (p2.given_name || ' ' || p2.surname) as committer_name,\n  tt.code as time_type_code,\n  tt.name as time_type_name,\n  j.number as job_number,\n  j.description as job_description,\n  c.name as category_name,\n  d.code as division_code,\n  d.name as division_name\nFROM time_amendments ta\nLEFT JOIN time_types tt ON ta.time_type = tt.id\nLEFT JOIN jobs j ON ta.job = j.id\nLEFT JOIN categories c ON ta.category = c.id\nLEFT JOIN divisions d ON ta.division = d.id\nLEFT JOIN profiles p0 ON ta.uid = p0.uid\nLEFT JOIN profiles p1 ON ta.creator = p1.uid\nLEFT JOIN profiles p2 ON ta.committer = p2.uid\nORDER BY ta.date DESC;"
		}`), &collection); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"cascadeDelete": false,
			"collectionId": "3esdddggow6dykr",
			"hidden": false,
			"id": "_clone_1Cz2",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "division",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(2, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_wrgH",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "uid",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(3, []byte(`{
			"hidden": false,
			"id": "_clone_lgdK",
			"max": 18,
			"min": -18,
			"name": "hours",
			"onlyInt": false,
			"presentable": false,
			"required": true,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(4, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_bB6v",
			"max": 0,
			"min": 0,
			"name": "description",
			"pattern": "",
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
			"cascadeDelete": false,
			"collectionId": "cnqv0wm8hly7r3n",
			"hidden": false,
			"id": "_clone_Ai7p",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "time_type",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(6, []byte(`{
			"hidden": false,
			"id": "_clone_BxVm",
			"max": 3,
			"min": 0,
			"name": "meals_hours",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(7, []byte(`{
			"cascadeDelete": false,
			"collectionId": "yovqzrnnomp0lkx",
			"hidden": false,
			"id": "_clone_hkVy",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "job",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(8, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_Awre",
			"max": 0,
			"min": 0,
			"name": "work_record",
			"pattern": "^[FKQ][0-9]{2}-[0-9]{3,}(-[0-9]+)?$",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(9, []byte(`{
			"hidden": false,
			"id": "_clone_KHd6",
			"max": null,
			"min": null,
			"name": "payout_request_amount",
			"onlyInt": false,
			"presentable": false,
			"required": false,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(10, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_ckRq",
			"max": 0,
			"min": 0,
			"name": "date",
			"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
			"presentable": true,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(11, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_KTYC",
			"max": 0,
			"min": 0,
			"name": "week_ending",
			"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
			"presentable": true,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(12, []byte(`{
			"cascadeDelete": false,
			"collectionId": "fpri53nrr2xgoov",
			"hidden": false,
			"id": "_clone_gIeX",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "tsid",
			"presentable": true,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(13, []byte(`{
			"cascadeDelete": false,
			"collectionId": "nrwhbwowokwu6cr",
			"hidden": false,
			"id": "_clone_Isy2",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "category",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(14, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_WDS9",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "creator",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(15, []byte(`{
			"hidden": false,
			"id": "_clone_7hGz",
			"max": "",
			"min": "",
			"name": "committed",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "date"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(16, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_M9iZ",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "committer",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(17, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_rcp3",
			"max": 0,
			"min": 0,
			"name": "committed_week_ending",
			"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
			"presentable": true,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(18, []byte(`{
			"hidden": false,
			"id": "_clone_RzaS",
			"name": "skip_tsid_check",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(22, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_wFnX",
			"max": 0,
			"min": 1,
			"name": "time_type_code",
			"pattern": "",
			"presentable": true,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(23, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_GV48",
			"max": 0,
			"min": 2,
			"name": "time_type_name",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(24, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_8pQt",
			"max": 0,
			"min": 0,
			"name": "job_number",
			"pattern": "^(P)?[0-9]{2}-[0-9]{3,4}(-[0-9]{1,2})?(-[0-9])?$",
			"presentable": true,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(25, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_3k5l",
			"max": 0,
			"min": 3,
			"name": "job_description",
			"pattern": "",
			"presentable": true,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(26, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_Jxty",
			"max": 0,
			"min": 3,
			"name": "category_name",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(27, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_RCB6",
			"max": 0,
			"min": 1,
			"name": "division_code",
			"pattern": "",
			"presentable": true,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(28, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_mP6o",
			"max": 0,
			"min": 2,
			"name": "division_name",
			"pattern": "",
			"presentable": false,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_mAcc")

		// remove field
		collection.Fields.RemoveById("_clone_UF8z")

		// remove field
		collection.Fields.RemoveById("_clone_RXRl")

		// remove field
		collection.Fields.RemoveById("_clone_822i")

		// remove field
		collection.Fields.RemoveById("_clone_kCQq")

		// remove field
		collection.Fields.RemoveById("_clone_N5S7")

		// remove field
		collection.Fields.RemoveById("_clone_Mstw")

		// remove field
		collection.Fields.RemoveById("_clone_8R9b")

		// remove field
		collection.Fields.RemoveById("_clone_BYwu")

		// remove field
		collection.Fields.RemoveById("_clone_2Cax")

		// remove field
		collection.Fields.RemoveById("_clone_fwrZ")

		// remove field
		collection.Fields.RemoveById("_clone_AxzD")

		// remove field
		collection.Fields.RemoveById("_clone_a8YZ")

		// remove field
		collection.Fields.RemoveById("_clone_NQOl")

		// remove field
		collection.Fields.RemoveById("_clone_x3pm")

		// remove field
		collection.Fields.RemoveById("_clone_NCSW")

		// remove field
		collection.Fields.RemoveById("_clone_skE4")

		// remove field
		collection.Fields.RemoveById("_clone_rnA1")

		// remove field
		collection.Fields.RemoveById("_clone_aopl")

		// remove field
		collection.Fields.RemoveById("_clone_dF2Y")

		// remove field
		collection.Fields.RemoveById("_clone_1gJb")

		// remove field
		collection.Fields.RemoveById("_clone_iT4S")

		// remove field
		collection.Fields.RemoveById("_clone_Cq0n")

		// remove field
		collection.Fields.RemoveById("_clone_xBYK")

		// remove field
		collection.Fields.RemoveById("_clone_Ab8t")

		return app.Save(collection)
	})
}
