package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1245168108")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"viewQuery": "SELECT \n  po.id,\n  po.po_number,\n  po.status,\n  po.uid,\n  po.type,\n  po.date,\n  po.end_date,\n  po.frequency,\n  po.division,\n  po.description,\n  po.total,\n  po.payment_type,\n  po.attachment,\n  po.rejector,\n  po.rejected,\n  po.rejection_reason,\n  po.approver,\n  po.approved,\n  po.second_approver,\n  po.second_approval,\n  po.canceller,\n  po.cancelled,\n  po.job,\n  po.category,\n  po.vendor,\n  po.parent_po,\n  po.created,\n  po.updated,\n  po.closer,\n  po.closed,\n  po.closed_by_system,\n  po.priority_second_approver,\n  po.approval_total,\n  COALESCE((SELECT MAX(threshold) \n    FROM po_approval_thresholds \n    WHERE threshold < po.approval_total), 0) AS lower_threshold,\n  COALESCE((SELECT MIN(threshold) \n    FROM po_approval_thresholds \n    WHERE threshold >= po.approval_total),1000000) AS upper_threshold,\n  (SELECT COUNT(*) \n    FROM expenses \n    WHERE expenses.purchase_order = po.id AND expenses.committed != \"\") AS committed_expenses_count,\n  (p0.given_name || \" \" || p0.surname) AS uid_name,\n  (p1.given_name || \" \" || p1.surname) AS approver_name,\n  (p2.given_name || \" \" || p2.surname) AS second_approver_name,\n  (p3.given_name || \" \" || p3.surname) AS priority_second_approver_name,\n  (p4.given_name || \" \" || p4.surname) AS rejector_name,\n  po2.po_number AS parent_po_number,\n  v.name AS vendor_name,\n  v.alias AS vendor_alias,\n  j.number AS job_number,\n  cl.name AS client_name,\n  j.description AS job_description,\n  d.code AS division_code,\n  d.name AS division_name,\n  c.name AS category_name\nFROM purchase_orders AS po\nLEFT JOIN profiles AS p0 ON po.uid = p0.uid\nLEFT JOIN profiles AS p1 ON po.approver = p1.uid\nLEFT JOIN profiles AS p2 ON po.second_approver = p2.uid\nLEFT JOIN profiles AS p3 ON po.priority_second_approver = p3.uid\nLEFT JOIN profiles AS p4 ON po.rejector = p4.uid\nLEFT JOIN purchase_orders AS po2 ON po.parent_po = po2.id\nLEFT JOIN vendors AS v ON po.vendor = v.id\nLEFT JOIN jobs AS j ON po.job = j.id\nLEFT JOIN divisions AS d ON po.division = d.id\nLEFT JOIN categories AS c ON po.category = c.id\nLEFT JOIN clients AS cl ON j.client = cl.id;"
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_BXAd")

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_JsNp",
			"max": 0,
			"min": 0,
			"name": "po_number",
			"pattern": "^(20[2-9]\\d)-(0{3}[1-9]|0{2}[1-9]\\d|0[1-9]\\d{2}|[1-3]\\d{3}|4[0-8]\\d{2}|49[0-9]{2})(?:-(0[1-9]|[1-9]\\d))?$",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(2, []byte(`{
			"hidden": false,
			"id": "_clone_h3uC",
			"maxSelect": 1,
			"name": "status",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "select",
			"values": [
				"Unapproved",
				"Active",
				"Cancelled",
				"Closed"
			]
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(3, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_Y9sq",
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
		if err := collection.Fields.AddMarshaledJSONAt(4, []byte(`{
			"hidden": false,
			"id": "_clone_QvlD",
			"maxSelect": 1,
			"name": "type",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "select",
			"values": [
				"Normal",
				"Cumulative",
				"Recurring"
			]
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(5, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_cewU",
			"max": 0,
			"min": 0,
			"name": "date",
			"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
			"presentable": false,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(6, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_jQyI",
			"max": 0,
			"min": 0,
			"name": "end_date",
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
		if err := collection.Fields.AddMarshaledJSONAt(7, []byte(`{
			"hidden": false,
			"id": "_clone_Nep6",
			"maxSelect": 1,
			"name": "frequency",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "select",
			"values": [
				"Weekly",
				"Biweekly",
				"Monthly"
			]
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(8, []byte(`{
			"cascadeDelete": false,
			"collectionId": "3esdddggow6dykr",
			"hidden": false,
			"id": "_clone_g3GP",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "division",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(9, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_1ZGJ",
			"max": 0,
			"min": 5,
			"name": "description",
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
		if err := collection.Fields.AddMarshaledJSONAt(10, []byte(`{
			"hidden": false,
			"id": "_clone_Daqy",
			"max": null,
			"min": 0,
			"name": "total",
			"onlyInt": false,
			"presentable": false,
			"required": true,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(11, []byte(`{
			"hidden": false,
			"id": "_clone_tvQp",
			"maxSelect": 1,
			"name": "payment_type",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "select",
			"values": [
				"OnAccount",
				"Expense",
				"CorporateCreditCard"
			]
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(12, []byte(`{
			"hidden": false,
			"id": "_clone_mnlu",
			"maxSelect": 1,
			"maxSize": 5242880,
			"mimeTypes": [
				"application/pdf",
				"image/jpeg",
				"image/png",
				"image/heic"
			],
			"name": "attachment",
			"presentable": false,
			"protected": false,
			"required": false,
			"system": false,
			"thumbs": null,
			"type": "file"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(13, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_pqFu",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "rejector",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(14, []byte(`{
			"hidden": false,
			"id": "_clone_UUJb",
			"max": "",
			"min": "",
			"name": "rejected",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "date"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(15, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_aZbX",
			"max": 0,
			"min": 5,
			"name": "rejection_reason",
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
		if err := collection.Fields.AddMarshaledJSONAt(16, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_Wm1z",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "approver",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(17, []byte(`{
			"hidden": false,
			"id": "_clone_HMhE",
			"max": "",
			"min": "",
			"name": "approved",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "date"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(18, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_MeRJ",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "second_approver",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(19, []byte(`{
			"hidden": false,
			"id": "_clone_HpXg",
			"max": "",
			"min": "",
			"name": "second_approval",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "date"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(20, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_Mguk",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "canceller",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(21, []byte(`{
			"hidden": false,
			"id": "_clone_P0uI",
			"max": "",
			"min": "",
			"name": "cancelled",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "date"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(22, []byte(`{
			"cascadeDelete": false,
			"collectionId": "yovqzrnnomp0lkx",
			"hidden": false,
			"id": "_clone_rZai",
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
		if err := collection.Fields.AddMarshaledJSONAt(23, []byte(`{
			"cascadeDelete": false,
			"collectionId": "nrwhbwowokwu6cr",
			"hidden": false,
			"id": "_clone_Hnto",
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
		if err := collection.Fields.AddMarshaledJSONAt(24, []byte(`{
			"cascadeDelete": false,
			"collectionId": "y0xvnesailac971",
			"hidden": false,
			"id": "_clone_iVcw",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "vendor",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(25, []byte(`{
			"cascadeDelete": false,
			"collectionId": "m19q72syy0e3lvm",
			"hidden": false,
			"id": "_clone_e3jJ",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "parent_po",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(26, []byte(`{
			"hidden": false,
			"id": "_clone_gi3n",
			"name": "created",
			"onCreate": true,
			"onUpdate": false,
			"presentable": false,
			"system": false,
			"type": "autodate"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(27, []byte(`{
			"hidden": false,
			"id": "_clone_eVt1",
			"name": "updated",
			"onCreate": true,
			"onUpdate": true,
			"presentable": false,
			"system": false,
			"type": "autodate"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(28, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_8uK5",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "closer",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(29, []byte(`{
			"hidden": false,
			"id": "_clone_IjfC",
			"max": "",
			"min": "",
			"name": "closed",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "date"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(30, []byte(`{
			"hidden": false,
			"id": "_clone_i3NI",
			"name": "closed_by_system",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(31, []byte(`{
			"cascadeDelete": false,
			"collectionId": "_pb_users_auth_",
			"hidden": false,
			"id": "_clone_bMPp",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "priority_second_approver",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(32, []byte(`{
			"hidden": false,
			"id": "_clone_E6SG",
			"max": null,
			"min": null,
			"name": "approval_total",
			"onlyInt": false,
			"presentable": false,
			"required": true,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(36, []byte(`{
			"hidden": false,
			"id": "json3388866023",
			"maxSize": 1,
			"name": "uid_name",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "json"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(37, []byte(`{
			"hidden": false,
			"id": "json1197482769",
			"maxSize": 1,
			"name": "approver_name",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "json"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(38, []byte(`{
			"hidden": false,
			"id": "json1788389956",
			"maxSize": 1,
			"name": "second_approver_name",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "json"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(39, []byte(`{
			"hidden": false,
			"id": "json2279208833",
			"maxSize": 1,
			"name": "priority_second_approver_name",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "json"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(40, []byte(`{
			"hidden": false,
			"id": "json1398595088",
			"maxSize": 1,
			"name": "rejector_name",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "json"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(41, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_GZu6",
			"max": 0,
			"min": 0,
			"name": "parent_po_number",
			"pattern": "^(20[2-9]\\d)-(0{3}[1-9]|0{2}[1-9]\\d|0[1-9]\\d{2}|[1-3]\\d{3}|4[0-8]\\d{2}|49[0-9]{2})(?:-(0[1-9]|[1-9]\\d))?$",
			"presentable": false,
			"primaryKey": false,
			"required": false,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(42, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_DGW0",
			"max": 0,
			"min": 3,
			"name": "vendor_name",
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
		if err := collection.Fields.AddMarshaledJSONAt(43, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_dhFu",
			"max": 0,
			"min": 3,
			"name": "vendor_alias",
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
		if err := collection.Fields.AddMarshaledJSONAt(44, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_Mzfn",
			"max": 0,
			"min": 0,
			"name": "job_number",
			"pattern": "^(P)?[0-9]{2}-[0-9]{3,4}(-[0-9]{1,2})?(-[0-9])?$",
			"presentable": true,
			"primaryKey": false,
			"required": true,
			"system": false,
			"type": "text"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(45, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_z1xf",
			"max": 0,
			"min": 2,
			"name": "client_name",
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
		if err := collection.Fields.AddMarshaledJSONAt(46, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_IIij",
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
		if err := collection.Fields.AddMarshaledJSONAt(47, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_oah9",
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
		if err := collection.Fields.AddMarshaledJSONAt(48, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_x1II",
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

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(49, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_Pi9e",
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

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1245168108")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"viewQuery": "SELECT \n    po.id, po.approval_total,\n    COALESCE((SELECT MAX(threshold) \n     FROM po_approval_thresholds \n     WHERE threshold < po.approval_total), 0) AS lower_threshold,\n    COALESCE((SELECT MIN(threshold) \n     FROM po_approval_thresholds \n     WHERE threshold >= po.approval_total),1000000) AS upper_threshold,\n    (SELECT COUNT(*) \n     FROM expenses \n     WHERE expenses.purchase_order = po.id AND expenses.committed != \"\") AS committed_expenses_count\nFROM purchase_orders AS po;"
		}`), &collection); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"hidden": false,
			"id": "_clone_BXAd",
			"max": null,
			"min": null,
			"name": "approval_total",
			"onlyInt": false,
			"presentable": false,
			"required": true,
			"system": false,
			"type": "number"
		}`)); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_JsNp")

		// remove field
		collection.Fields.RemoveById("_clone_h3uC")

		// remove field
		collection.Fields.RemoveById("_clone_Y9sq")

		// remove field
		collection.Fields.RemoveById("_clone_QvlD")

		// remove field
		collection.Fields.RemoveById("_clone_cewU")

		// remove field
		collection.Fields.RemoveById("_clone_jQyI")

		// remove field
		collection.Fields.RemoveById("_clone_Nep6")

		// remove field
		collection.Fields.RemoveById("_clone_g3GP")

		// remove field
		collection.Fields.RemoveById("_clone_1ZGJ")

		// remove field
		collection.Fields.RemoveById("_clone_Daqy")

		// remove field
		collection.Fields.RemoveById("_clone_tvQp")

		// remove field
		collection.Fields.RemoveById("_clone_mnlu")

		// remove field
		collection.Fields.RemoveById("_clone_pqFu")

		// remove field
		collection.Fields.RemoveById("_clone_UUJb")

		// remove field
		collection.Fields.RemoveById("_clone_aZbX")

		// remove field
		collection.Fields.RemoveById("_clone_Wm1z")

		// remove field
		collection.Fields.RemoveById("_clone_HMhE")

		// remove field
		collection.Fields.RemoveById("_clone_MeRJ")

		// remove field
		collection.Fields.RemoveById("_clone_HpXg")

		// remove field
		collection.Fields.RemoveById("_clone_Mguk")

		// remove field
		collection.Fields.RemoveById("_clone_P0uI")

		// remove field
		collection.Fields.RemoveById("_clone_rZai")

		// remove field
		collection.Fields.RemoveById("_clone_Hnto")

		// remove field
		collection.Fields.RemoveById("_clone_iVcw")

		// remove field
		collection.Fields.RemoveById("_clone_e3jJ")

		// remove field
		collection.Fields.RemoveById("_clone_gi3n")

		// remove field
		collection.Fields.RemoveById("_clone_eVt1")

		// remove field
		collection.Fields.RemoveById("_clone_8uK5")

		// remove field
		collection.Fields.RemoveById("_clone_IjfC")

		// remove field
		collection.Fields.RemoveById("_clone_i3NI")

		// remove field
		collection.Fields.RemoveById("_clone_bMPp")

		// remove field
		collection.Fields.RemoveById("_clone_E6SG")

		// remove field
		collection.Fields.RemoveById("json3388866023")

		// remove field
		collection.Fields.RemoveById("json1197482769")

		// remove field
		collection.Fields.RemoveById("json1788389956")

		// remove field
		collection.Fields.RemoveById("json2279208833")

		// remove field
		collection.Fields.RemoveById("json1398595088")

		// remove field
		collection.Fields.RemoveById("_clone_GZu6")

		// remove field
		collection.Fields.RemoveById("_clone_DGW0")

		// remove field
		collection.Fields.RemoveById("_clone_dhFu")

		// remove field
		collection.Fields.RemoveById("_clone_Mzfn")

		// remove field
		collection.Fields.RemoveById("_clone_z1xf")

		// remove field
		collection.Fields.RemoveById("_clone_IIij")

		// remove field
		collection.Fields.RemoveById("_clone_oah9")

		// remove field
		collection.Fields.RemoveById("_clone_x1II")

		// remove field
		collection.Fields.RemoveById("_clone_Pi9e")

		return app.Save(collection)
	})
}
