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
			"listRule": "// copy the listRule and viewRule from the purchase_orders collection api rules\n// Active purchase_orders can be viewed by any authenticated user\n(status = \"Active\" && @request.auth.id != \"\") ||\n\n// Cancelled and Closed purchase_orders can be viewed by uid, approver, second_approver, and 'report' claim holder\n(\n  (status = \"Cancelled\" || status = \"Closed\") &&\n  (\n    @request.auth.id = uid || \n    @request.auth.id = approver || \n    @request.auth.id = second_approver || \n    @request.auth.user_claims_via_uid.cid.name ?= 'report'\n  )\n) ||\n\n// TODO: We may also later allow Closed purchase_orders to be viewed by uid, approver, and committer of corresponding expenses, if any, in a rule here\n\n// Unapproved purchase_orders can be viewed by uid, approver, priority_second_approver and, if updated is more than 24 hours ago, any holder of the po_approver claim whose po_approver_props.max_amount >= approval_amount and <= the upper_threshold of the tier.\n(\n  status = \"Unapproved\" &&\n  (\n    @request.auth.id = uid || \n    @request.auth.id = approver || \n    @request.auth.id = priority_second_approver \n  ) || \n  (\n    // updated more than 24 hours ago\n    updated < @yesterday && \n    \n    // caller has the po_approver claim\n    @request.auth.user_claims_via_uid.cid.name ?= \"po_approver\" &&\n\n    // caller max_amount for the po_approver claim >= approval_total\n    @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.max_amount >= approval_total &&\n\n    // caller user_claims.payload.divisions = null OR includes division\n    (\n      @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.divisions:length = 0 ||\n      @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.divisions:each ?= division\n    ) &&\n    (\n      @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.max_amount >= approval_total &&\n      @collection.purchase_orders_augmented.id ?= id &&\n      @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.max_amount ?<= @collection.purchase_orders_augmented.upper_threshold\n    )\n  )\n)",
			"viewRule": "// copy the listRule and viewRule from the purchase_orders collection api rules\n// Active purchase_orders can be viewed by any authenticated user\n(status = \"Active\" && @request.auth.id != \"\") ||\n\n// Cancelled and Closed purchase_orders can be viewed by uid, approver, second_approver, and 'report' claim holder\n(\n  (status = \"Cancelled\" || status = \"Closed\") &&\n  (\n    @request.auth.id = uid || \n    @request.auth.id = approver || \n    @request.auth.id = second_approver || \n    @request.auth.user_claims_via_uid.cid.name ?= 'report'\n  )\n) ||\n\n// TODO: We may also later allow Closed purchase_orders to be viewed by uid, approver, and committer of corresponding expenses, if any, in a rule here\n\n// Unapproved purchase_orders can be viewed by uid, approver, priority_second_approver and, if updated is more than 24 hours ago, any holder of the po_approver claim whose po_approver_props.max_amount >= approval_amount and <= the upper_threshold of the tier.\n(\n  status = \"Unapproved\" &&\n  (\n    @request.auth.id = uid || \n    @request.auth.id = approver || \n    @request.auth.id = priority_second_approver \n  ) || \n  (\n    // updated more than 24 hours ago\n    updated < @yesterday && \n    \n    // caller has the po_approver claim\n    @request.auth.user_claims_via_uid.cid.name ?= \"po_approver\" &&\n\n    // caller max_amount for the po_approver claim >= approval_total\n    @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.max_amount >= approval_total &&\n\n    // caller user_claims.payload.divisions = null OR includes division\n    (\n      @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.divisions:length = 0 ||\n      @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.divisions:each ?= division\n    ) &&\n    (\n      @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.max_amount >= approval_total &&\n      @collection.purchase_orders_augmented.id ?= id &&\n      @request.auth.user_claims_via_uid.po_approver_props_via_user_claim.max_amount ?<= @collection.purchase_orders_augmented.upper_threshold\n    )\n  )\n)"
		}`), &collection); err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("_clone_aOsm")

		// remove field
		collection.Fields.RemoveById("_clone_bTLo")

		// remove field
		collection.Fields.RemoveById("_clone_iQq2")

		// remove field
		collection.Fields.RemoveById("_clone_OjPc")

		// remove field
		collection.Fields.RemoveById("_clone_nT9p")

		// remove field
		collection.Fields.RemoveById("_clone_Mp8U")

		// remove field
		collection.Fields.RemoveById("_clone_dVOi")

		// remove field
		collection.Fields.RemoveById("_clone_47KW")

		// remove field
		collection.Fields.RemoveById("_clone_Q78H")

		// remove field
		collection.Fields.RemoveById("_clone_AAjg")

		// remove field
		collection.Fields.RemoveById("_clone_25Is")

		// remove field
		collection.Fields.RemoveById("_clone_CJCc")

		// remove field
		collection.Fields.RemoveById("_clone_auzZ")

		// remove field
		collection.Fields.RemoveById("_clone_cPQe")

		// remove field
		collection.Fields.RemoveById("_clone_8mVP")

		// remove field
		collection.Fields.RemoveById("_clone_8lId")

		// remove field
		collection.Fields.RemoveById("_clone_5iYr")

		// remove field
		collection.Fields.RemoveById("_clone_uqfj")

		// remove field
		collection.Fields.RemoveById("_clone_tZtm")

		// remove field
		collection.Fields.RemoveById("_clone_6Knq")

		// remove field
		collection.Fields.RemoveById("_clone_254y")

		// remove field
		collection.Fields.RemoveById("_clone_E2MJ")

		// remove field
		collection.Fields.RemoveById("_clone_6DDI")

		// remove field
		collection.Fields.RemoveById("_clone_aZtV")

		// remove field
		collection.Fields.RemoveById("_clone_uE7E")

		// remove field
		collection.Fields.RemoveById("_clone_UrXg")

		// remove field
		collection.Fields.RemoveById("_clone_MuzC")

		// remove field
		collection.Fields.RemoveById("_clone_wH7A")

		// remove field
		collection.Fields.RemoveById("_clone_ysRm")

		// remove field
		collection.Fields.RemoveById("_clone_oWdi")

		// remove field
		collection.Fields.RemoveById("_clone_bdhL")

		// remove field
		collection.Fields.RemoveById("_clone_5sUD")

		// remove field
		collection.Fields.RemoveById("_clone_LazZ")

		// remove field
		collection.Fields.RemoveById("_clone_xfgi")

		// remove field
		collection.Fields.RemoveById("_clone_yieP")

		// remove field
		collection.Fields.RemoveById("_clone_Ucrj")

		// remove field
		collection.Fields.RemoveById("_clone_SSid")

		// remove field
		collection.Fields.RemoveById("_clone_lrLz")

		// remove field
		collection.Fields.RemoveById("_clone_AFko")

		// remove field
		collection.Fields.RemoveById("_clone_lnhl")

		// remove field
		collection.Fields.RemoveById("_clone_HhEv")

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_R3li",
			"max": 0,
			"min": 0,
			"name": "po_number",
			"pattern": "^(20[2-9]\\d)-(0{3}[1-9]|0{2}[1-9]\\d|0[1-9]\\d{2}|[1-3]\\d{3}|4[0-8]\\d{2}|49[0-9]{2})(?:-(0[1-9]|[1-9]\\d))?$",
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
			"id": "_clone_tESh",
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
			"id": "_clone_vuME",
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
			"id": "_clone_Q9LM",
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
			"id": "_clone_pqFy",
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
			"id": "_clone_7fYi",
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
			"id": "_clone_DBdc",
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
			"id": "_clone_dL0i",
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
			"id": "_clone_Y7i5",
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
			"id": "_clone_74e8",
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
			"id": "_clone_mHtj",
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
			"id": "_clone_205J",
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
			"id": "_clone_W7eM",
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
			"id": "_clone_FK9y",
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
			"id": "_clone_06Wl",
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
			"id": "_clone_t9FA",
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
			"id": "_clone_aosz",
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
			"id": "_clone_nWUL",
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
			"id": "_clone_RZw3",
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
			"id": "_clone_TvOP",
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
			"id": "_clone_3zTD",
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
			"id": "_clone_vpW8",
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
			"id": "_clone_CL1y",
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
			"id": "_clone_v4TM",
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
			"id": "_clone_uJbY",
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
			"id": "_clone_9o8S",
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
			"id": "_clone_j3k2",
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
			"id": "_clone_wnae",
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
			"id": "_clone_1ORX",
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
			"id": "_clone_ecQs",
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
			"id": "_clone_YTso",
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
			"id": "_clone_SmID",
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
			"id": "_clone_F3ZF",
			"max": 0,
			"min": 0,
			"name": "parent_po_number",
			"pattern": "^(20[2-9]\\d)-(0{3}[1-9]|0{2}[1-9]\\d|0[1-9]\\d{2}|[1-3]\\d{3}|4[0-8]\\d{2}|49[0-9]{2})(?:-(0[1-9]|[1-9]\\d))?$",
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
			"id": "_clone_xPaF",
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
			"id": "_clone_2urH",
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
			"id": "_clone_olwO",
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
			"id": "_clone_8b2O",
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
			"id": "_clone_YZbx",
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
			"id": "_clone_AeHR",
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
			"id": "_clone_0dBJ",
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
			"id": "_clone_jFtr",
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
			"listRule": "@request.auth.id != \"\"",
			"viewRule": "@request.auth.id != \"\""
		}`), &collection); err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(1, []byte(`{
			"autogeneratePattern": "",
			"hidden": false,
			"id": "_clone_aOsm",
			"max": 0,
			"min": 0,
			"name": "po_number",
			"pattern": "^(20[2-9]\\d)-(0{3}[1-9]|0{2}[1-9]\\d|0[1-9]\\d{2}|[1-3]\\d{3}|4[0-8]\\d{2}|49[0-9]{2})(?:-(0[1-9]|[1-9]\\d))?$",
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
			"id": "_clone_bTLo",
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
			"id": "_clone_iQq2",
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
			"id": "_clone_OjPc",
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
			"id": "_clone_nT9p",
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
			"id": "_clone_Mp8U",
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
			"id": "_clone_dVOi",
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
			"id": "_clone_47KW",
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
			"id": "_clone_Q78H",
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
			"id": "_clone_AAjg",
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
			"id": "_clone_25Is",
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
			"id": "_clone_CJCc",
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
			"id": "_clone_auzZ",
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
			"id": "_clone_cPQe",
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
			"id": "_clone_8mVP",
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
			"id": "_clone_8lId",
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
			"id": "_clone_5iYr",
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
			"id": "_clone_uqfj",
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
			"id": "_clone_tZtm",
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
			"id": "_clone_6Knq",
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
			"id": "_clone_254y",
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
			"id": "_clone_E2MJ",
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
			"id": "_clone_6DDI",
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
			"id": "_clone_aZtV",
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
			"id": "_clone_uE7E",
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
			"id": "_clone_UrXg",
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
			"id": "_clone_MuzC",
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
			"id": "_clone_wH7A",
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
			"id": "_clone_ysRm",
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
			"id": "_clone_oWdi",
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
			"id": "_clone_bdhL",
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
			"id": "_clone_5sUD",
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
			"id": "_clone_LazZ",
			"max": 0,
			"min": 0,
			"name": "parent_po_number",
			"pattern": "^(20[2-9]\\d)-(0{3}[1-9]|0{2}[1-9]\\d|0[1-9]\\d{2}|[1-3]\\d{3}|4[0-8]\\d{2}|49[0-9]{2})(?:-(0[1-9]|[1-9]\\d))?$",
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
			"id": "_clone_xfgi",
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
			"id": "_clone_yieP",
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
			"id": "_clone_Ucrj",
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
			"id": "_clone_SSid",
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
			"id": "_clone_lrLz",
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
			"id": "_clone_AFko",
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
			"id": "_clone_lnhl",
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
			"id": "_clone_HhEv",
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
		collection.Fields.RemoveById("_clone_R3li")

		// remove field
		collection.Fields.RemoveById("_clone_tESh")

		// remove field
		collection.Fields.RemoveById("_clone_vuME")

		// remove field
		collection.Fields.RemoveById("_clone_Q9LM")

		// remove field
		collection.Fields.RemoveById("_clone_pqFy")

		// remove field
		collection.Fields.RemoveById("_clone_7fYi")

		// remove field
		collection.Fields.RemoveById("_clone_DBdc")

		// remove field
		collection.Fields.RemoveById("_clone_dL0i")

		// remove field
		collection.Fields.RemoveById("_clone_Y7i5")

		// remove field
		collection.Fields.RemoveById("_clone_74e8")

		// remove field
		collection.Fields.RemoveById("_clone_mHtj")

		// remove field
		collection.Fields.RemoveById("_clone_205J")

		// remove field
		collection.Fields.RemoveById("_clone_W7eM")

		// remove field
		collection.Fields.RemoveById("_clone_FK9y")

		// remove field
		collection.Fields.RemoveById("_clone_06Wl")

		// remove field
		collection.Fields.RemoveById("_clone_t9FA")

		// remove field
		collection.Fields.RemoveById("_clone_aosz")

		// remove field
		collection.Fields.RemoveById("_clone_nWUL")

		// remove field
		collection.Fields.RemoveById("_clone_RZw3")

		// remove field
		collection.Fields.RemoveById("_clone_TvOP")

		// remove field
		collection.Fields.RemoveById("_clone_3zTD")

		// remove field
		collection.Fields.RemoveById("_clone_vpW8")

		// remove field
		collection.Fields.RemoveById("_clone_CL1y")

		// remove field
		collection.Fields.RemoveById("_clone_v4TM")

		// remove field
		collection.Fields.RemoveById("_clone_uJbY")

		// remove field
		collection.Fields.RemoveById("_clone_9o8S")

		// remove field
		collection.Fields.RemoveById("_clone_j3k2")

		// remove field
		collection.Fields.RemoveById("_clone_wnae")

		// remove field
		collection.Fields.RemoveById("_clone_1ORX")

		// remove field
		collection.Fields.RemoveById("_clone_ecQs")

		// remove field
		collection.Fields.RemoveById("_clone_YTso")

		// remove field
		collection.Fields.RemoveById("_clone_SmID")

		// remove field
		collection.Fields.RemoveById("_clone_F3ZF")

		// remove field
		collection.Fields.RemoveById("_clone_xPaF")

		// remove field
		collection.Fields.RemoveById("_clone_2urH")

		// remove field
		collection.Fields.RemoveById("_clone_olwO")

		// remove field
		collection.Fields.RemoveById("_clone_8b2O")

		// remove field
		collection.Fields.RemoveById("_clone_YZbx")

		// remove field
		collection.Fields.RemoveById("_clone_AeHR")

		// remove field
		collection.Fields.RemoveById("_clone_0dBJ")

		// remove field
		collection.Fields.RemoveById("_clone_jFtr")

		return app.Save(collection)
	})
}
