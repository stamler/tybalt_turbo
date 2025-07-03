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
			"viewQuery": "SELECT \n  po.id,\n  po.po_number,\n  po.status,\n  po.uid,\n  po.type,\n  po.date,\n  po.end_date,\n  po.frequency,\n  po.division,\n  po.description,\n  po.total,\n  po.payment_type,\n  po.attachment,\n  po.rejector,\n  po.rejected,\n  po.rejection_reason,\n  po.approver,\n  po.approved,\n  po.second_approver,\n  po.second_approval,\n  po.canceller,\n  po.cancelled,\n  po.job,\n  po.category,\n  po.vendor,\n  po.parent_po,\n  po.created,\n  po.updated,\n  po.closer,\n  po.closed,\n  po.closed_by_system,\n  po.priority_second_approver,\n  po.approval_total,\n  COALESCE((SELECT MAX(threshold) \n    FROM po_approval_thresholds \n    WHERE threshold < po.approval_total), 0) AS lower_threshold,\n  COALESCE((SELECT MIN(threshold) \n    FROM po_approval_thresholds \n    WHERE threshold >= po.approval_total),1000000) AS upper_threshold,\n  (SELECT COUNT(*) \n    FROM expenses \n    WHERE expenses.purchase_order = po.id AND expenses.committed != \"\") AS committed_expenses_count,\n  (p0.given_name || \" \" || p0.surname) AS uid_name,\n  (p1.given_name || \" \" || p1.surname) AS approver_name,\n  (p2.given_name || \" \" || p2.surname) AS second_approver_name,\n  (p3.given_name || \" \" || p3.surname) AS priority_second_approver_name,\n  (p4.given_name || \" \" || p4.surname) AS rejector_name,\n  po2.po_number AS parent_po_number,\n  v.name AS vendor_name,\n  v.alias AS vendor_alias,\n  j.number AS job_number,\n  cl.name AS client_name,\n  cl.id AS client_id,\n  j.description AS job_description,\n  d.code AS division_code,\n  d.name AS division_name,\n  c.name AS category_name\nFROM purchase_orders AS po\nLEFT JOIN profiles AS p0 ON po.uid = p0.uid\nLEFT JOIN profiles AS p1 ON po.approver = p1.uid\nLEFT JOIN profiles AS p2 ON po.second_approver = p2.uid\nLEFT JOIN profiles AS p3 ON po.priority_second_approver = p3.uid\nLEFT JOIN profiles AS p4 ON po.rejector = p4.uid\nLEFT JOIN purchase_orders AS po2 ON po.parent_po = po2.id\nLEFT JOIN vendors AS v ON po.vendor = v.id\nLEFT JOIN jobs AS j ON po.job = j.id\nLEFT JOIN divisions AS d ON po.division = d.id\nLEFT JOIN categories AS c ON po.category = c.id\nLEFT JOIN clients AS cl ON j.client = cl.id;"
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_uk3n")

		// remove field
		collection.Fields.RemoveById("_clone_CxnO")

		// remove field
		collection.Fields.RemoveById("_clone_YhFu")

		// remove field
		collection.Fields.RemoveById("_clone_qDFT")

		// remove field
		collection.Fields.RemoveById("_clone_iCwp")

		// remove field
		collection.Fields.RemoveById("_clone_4G5w")

		// remove field
		collection.Fields.RemoveById("_clone_2tPy")

		// remove field
		collection.Fields.RemoveById("_clone_VxU1")

		// remove field
		collection.Fields.RemoveById("_clone_NNQb")

		// remove field
		collection.Fields.RemoveById("_clone_XFJx")

		// remove field
		collection.Fields.RemoveById("_clone_j5Bz")

		// remove field
		collection.Fields.RemoveById("_clone_Bepp")

		// remove field
		collection.Fields.RemoveById("_clone_gSQq")

		// remove field
		collection.Fields.RemoveById("_clone_S5P6")

		// remove field
		collection.Fields.RemoveById("_clone_nC7x")

		// remove field
		collection.Fields.RemoveById("_clone_eQeZ")

		// remove field
		collection.Fields.RemoveById("_clone_D26D")

		// remove field
		collection.Fields.RemoveById("_clone_GBLD")

		// remove field
		collection.Fields.RemoveById("_clone_ht1J")

		// remove field
		collection.Fields.RemoveById("_clone_VMTW")

		// remove field
		collection.Fields.RemoveById("_clone_Jwmx")

		// remove field
		collection.Fields.RemoveById("_clone_g8fL")

		// remove field
		collection.Fields.RemoveById("_clone_GLru")

		// remove field
		collection.Fields.RemoveById("_clone_KgGs")

		// remove field
		collection.Fields.RemoveById("_clone_fCL0")

		// remove field
		collection.Fields.RemoveById("_clone_oKiE")

		// remove field
		collection.Fields.RemoveById("_clone_8lE5")

		// remove field
		collection.Fields.RemoveById("_clone_rNrH")

		// remove field
		collection.Fields.RemoveById("_clone_DIZT")

		// remove field
		collection.Fields.RemoveById("_clone_xDZv")

		// remove field
		collection.Fields.RemoveById("_clone_B145")

		// remove field
		collection.Fields.RemoveById("_clone_j04J")

		// remove field
		collection.Fields.RemoveById("_clone_E3st")

		// remove field
		collection.Fields.RemoveById("_clone_DZhM")

		// remove field
		collection.Fields.RemoveById("_clone_CGGR")

		// remove field
		collection.Fields.RemoveById("_clone_M616")

		// remove field
		collection.Fields.RemoveById("_clone_pjnU")

		// remove field
		collection.Fields.RemoveById("_clone_lwad")

		// remove field
		collection.Fields.RemoveById("_clone_Vb2P")

		// remove field
		collection.Fields.RemoveById("_clone_vjqi")

		// remove field
		collection.Fields.RemoveById("_clone_Zo3g")

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_Mc1a",
			"max": 0,
			"min": 0,
			"name": "po_number",
			"pattern": "^([1-9]\\d{3})-(\\d{4})(?:-(0[1-9]|[1-9]\\d))?$",
			"presentable": true,
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
			"id": "_clone_me6E",
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
			"id": "_clone_hryG",
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
			"id": "_clone_EvbL",
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
			"id": "_clone_zQB6",
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
			"id": "_clone_MeMa",
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
			"id": "_clone_NOYW",
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
			"id": "_clone_l0q3",
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
			"id": "_clone_ks4I",
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
			"id": "_clone_XuOT",
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
			"id": "_clone_kspy",
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
			"id": "_clone_jbAG",
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
			"id": "_clone_pVvq",
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
			"id": "_clone_9z3M",
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
			"id": "_clone_4BfG",
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
			"id": "_clone_UTL3",
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
			"id": "_clone_MjGZ",
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
			"id": "_clone_Tk2a",
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
			"id": "_clone_YjpO",
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
			"id": "_clone_6age",
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
			"id": "_clone_b9O5",
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
			"id": "_clone_NeQl",
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
			"id": "_clone_U2CB",
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
			"id": "_clone_ZaYl",
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
			"id": "_clone_77cZ",
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
			"id": "_clone_a8QB",
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
			"id": "_clone_wMJf",
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
			"id": "_clone_wlmF",
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
			"id": "_clone_MFzW",
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
			"id": "_clone_5ibI",
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
			"id": "_clone_whiC",
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
			"id": "_clone_1biW",
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
		if err := collection.Fields.AddMarshaledJSONAt(41, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_WGSx",
			"max": 0,
			"min": 0,
			"name": "parent_po_number",
			"pattern": "^([1-9]\\d{3})-(\\d{4})(?:-(0[1-9]|[1-9]\\d))?$",
			"presentable": true,
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
			"id": "_clone_UsRo",
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
			"id": "_clone_slBo",
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
			"id": "_clone_sF75",
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
			"id": "_clone_xobI",
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
			"cascadeDelete": false,
			"collectionId": "1v6i9rrpniuatcx",
			"hidden": false,
			"id": "relation434858273",
			"maxSelect": 1,
			"minSelect": 0,
			"name": "client_id",
			"presentable": false,
			"required": false,
			"system": false,
			"type": "relation"
		}`)); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(47, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_CwH6",
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
		if err := collection.Fields.AddMarshaledJSONAt(48, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_lgZg",
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
		if err := collection.Fields.AddMarshaledJSONAt(49, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_Dl7r",
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
		if err := collection.Fields.AddMarshaledJSONAt(50, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_fkjQ",
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
			"viewQuery": "SELECT \n  po.id,\n  po.po_number,\n  po.status,\n  po.uid,\n  po.type,\n  po.date,\n  po.end_date,\n  po.frequency,\n  po.division,\n  po.description,\n  po.total,\n  po.payment_type,\n  po.attachment,\n  po.rejector,\n  po.rejected,\n  po.rejection_reason,\n  po.approver,\n  po.approved,\n  po.second_approver,\n  po.second_approval,\n  po.canceller,\n  po.cancelled,\n  po.job,\n  po.category,\n  po.vendor,\n  po.parent_po,\n  po.created,\n  po.updated,\n  po.closer,\n  po.closed,\n  po.closed_by_system,\n  po.priority_second_approver,\n  po.approval_total,\n  COALESCE((SELECT MAX(threshold) \n    FROM po_approval_thresholds \n    WHERE threshold < po.approval_total), 0) AS lower_threshold,\n  COALESCE((SELECT MIN(threshold) \n    FROM po_approval_thresholds \n    WHERE threshold >= po.approval_total),1000000) AS upper_threshold,\n  (SELECT COUNT(*) \n    FROM expenses \n    WHERE expenses.purchase_order = po.id AND expenses.committed != \"\") AS committed_expenses_count,\n  (p0.given_name || \" \" || p0.surname) AS uid_name,\n  (p1.given_name || \" \" || p1.surname) AS approver_name,\n  (p2.given_name || \" \" || p2.surname) AS second_approver_name,\n  (p3.given_name || \" \" || p3.surname) AS priority_second_approver_name,\n  (p4.given_name || \" \" || p4.surname) AS rejector_name,\n  po2.po_number AS parent_po_number,\n  v.name AS vendor_name,\n  v.alias AS vendor_alias,\n  j.number AS job_number,\n  cl.name AS client_name,\n  j.description AS job_description,\n  d.code AS division_code,\n  d.name AS division_name,\n  c.name AS category_name\nFROM purchase_orders AS po\nLEFT JOIN profiles AS p0 ON po.uid = p0.uid\nLEFT JOIN profiles AS p1 ON po.approver = p1.uid\nLEFT JOIN profiles AS p2 ON po.second_approver = p2.uid\nLEFT JOIN profiles AS p3 ON po.priority_second_approver = p3.uid\nLEFT JOIN profiles AS p4 ON po.rejector = p4.uid\nLEFT JOIN purchase_orders AS po2 ON po.parent_po = po2.id\nLEFT JOIN vendors AS v ON po.vendor = v.id\nLEFT JOIN jobs AS j ON po.job = j.id\nLEFT JOIN divisions AS d ON po.division = d.id\nLEFT JOIN categories AS c ON po.category = c.id\nLEFT JOIN clients AS cl ON j.client = cl.id;"
		}`), &collection); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_uk3n",
			"max": 0,
			"min": 0,
			"name": "po_number",
			"pattern": "^([1-9]\\d{3})-(\\d{4})(?:-(0[1-9]|[1-9]\\d))?$",
			"presentable": true,
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
			"id": "_clone_CxnO",
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
			"id": "_clone_YhFu",
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
			"id": "_clone_qDFT",
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
			"id": "_clone_iCwp",
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
			"id": "_clone_4G5w",
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
			"id": "_clone_2tPy",
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
			"id": "_clone_VxU1",
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
			"id": "_clone_NNQb",
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
			"id": "_clone_XFJx",
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
			"id": "_clone_j5Bz",
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
			"id": "_clone_Bepp",
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
			"id": "_clone_gSQq",
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
			"id": "_clone_S5P6",
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
			"id": "_clone_nC7x",
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
			"id": "_clone_eQeZ",
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
			"id": "_clone_D26D",
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
			"id": "_clone_GBLD",
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
			"id": "_clone_ht1J",
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
			"id": "_clone_VMTW",
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
			"id": "_clone_Jwmx",
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
			"id": "_clone_g8fL",
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
			"id": "_clone_GLru",
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
			"id": "_clone_KgGs",
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
			"id": "_clone_fCL0",
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
			"id": "_clone_oKiE",
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
			"id": "_clone_8lE5",
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
			"id": "_clone_rNrH",
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
			"id": "_clone_DIZT",
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
			"id": "_clone_xDZv",
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
			"id": "_clone_B145",
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
			"id": "_clone_j04J",
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
		if err := collection.Fields.AddMarshaledJSONAt(41, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_E3st",
			"max": 0,
			"min": 0,
			"name": "parent_po_number",
			"pattern": "^([1-9]\\d{3})-(\\d{4})(?:-(0[1-9]|[1-9]\\d))?$",
			"presentable": true,
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
			"id": "_clone_DZhM",
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
			"id": "_clone_CGGR",
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
			"id": "_clone_M616",
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
			"id": "_clone_pjnU",
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
			"id": "_clone_lwad",
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
			"id": "_clone_Vb2P",
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
			"id": "_clone_vjqi",
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
			"id": "_clone_Zo3g",
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

		// remove field
		collection.Fields.RemoveById("_clone_Mc1a")

		// remove field
		collection.Fields.RemoveById("_clone_me6E")

		// remove field
		collection.Fields.RemoveById("_clone_hryG")

		// remove field
		collection.Fields.RemoveById("_clone_EvbL")

		// remove field
		collection.Fields.RemoveById("_clone_zQB6")

		// remove field
		collection.Fields.RemoveById("_clone_MeMa")

		// remove field
		collection.Fields.RemoveById("_clone_NOYW")

		// remove field
		collection.Fields.RemoveById("_clone_l0q3")

		// remove field
		collection.Fields.RemoveById("_clone_ks4I")

		// remove field
		collection.Fields.RemoveById("_clone_XuOT")

		// remove field
		collection.Fields.RemoveById("_clone_kspy")

		// remove field
		collection.Fields.RemoveById("_clone_jbAG")

		// remove field
		collection.Fields.RemoveById("_clone_pVvq")

		// remove field
		collection.Fields.RemoveById("_clone_9z3M")

		// remove field
		collection.Fields.RemoveById("_clone_4BfG")

		// remove field
		collection.Fields.RemoveById("_clone_UTL3")

		// remove field
		collection.Fields.RemoveById("_clone_MjGZ")

		// remove field
		collection.Fields.RemoveById("_clone_Tk2a")

		// remove field
		collection.Fields.RemoveById("_clone_YjpO")

		// remove field
		collection.Fields.RemoveById("_clone_6age")

		// remove field
		collection.Fields.RemoveById("_clone_b9O5")

		// remove field
		collection.Fields.RemoveById("_clone_NeQl")

		// remove field
		collection.Fields.RemoveById("_clone_U2CB")

		// remove field
		collection.Fields.RemoveById("_clone_ZaYl")

		// remove field
		collection.Fields.RemoveById("_clone_77cZ")

		// remove field
		collection.Fields.RemoveById("_clone_a8QB")

		// remove field
		collection.Fields.RemoveById("_clone_wMJf")

		// remove field
		collection.Fields.RemoveById("_clone_wlmF")

		// remove field
		collection.Fields.RemoveById("_clone_MFzW")

		// remove field
		collection.Fields.RemoveById("_clone_5ibI")

		// remove field
		collection.Fields.RemoveById("_clone_whiC")

		// remove field
		collection.Fields.RemoveById("_clone_1biW")

		// remove field
		collection.Fields.RemoveById("_clone_WGSx")

		// remove field
		collection.Fields.RemoveById("_clone_UsRo")

		// remove field
		collection.Fields.RemoveById("_clone_slBo")

		// remove field
		collection.Fields.RemoveById("_clone_sF75")

		// remove field
		collection.Fields.RemoveById("_clone_xobI")

		// remove field
		collection.Fields.RemoveById("relation434858273")

		// remove field
		collection.Fields.RemoveById("_clone_CwH6")

		// remove field
		collection.Fields.RemoveById("_clone_lgZg")

		// remove field
		collection.Fields.RemoveById("_clone_Dl7r")

		// remove field
		collection.Fields.RemoveById("_clone_fkjQ")

		return app.Save(collection)
	})
}
