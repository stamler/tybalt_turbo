package migrations

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"strings"
)

func init() {
	m.Register(func(app core.App) error {
		var spec struct {
			New     []json.RawMessage          `json:"new"`
			Updates map[string]json.RawMessage `json:"updates"`
		}
		// The counters below have separate purposes:
		// - job_requests.version identifies saved request data. The attachment's
		//   request_version records which request revision received approval.
		// - job_attachments.version identifies a revision of one attachment. Value
		//   edits, approval, and rejection advance it; edit/review/delete actions
		//   must compare the client's version with the stored version in a transaction.
		//   It detects stale actions; it does not order separate history entries.
		// - job_attachments.approval_sequence orders retained approvals for one job.
		//   Pending/rejected rows use zero. Initial job approval uses one; each later
		//   approval uses the largest retained sequence plus one in its transaction.
		//   The partial unique index protects this order. Only the latest later
		//   approval can be removed and its previous_project_value restored.
		// These fields are maintained by workflow code, not automatic timestamps.
		// PocketBase timestamps have millisecond precision and can tie; clocks can
		// move backward. IDs identify rows but cannot establish change or approval
		// order. Neither counter is a globally unique event ID or a saved snapshot.
		//
		// job_requests.submission_token identifies one submission for review. It is
		// a server-generated, cryptographically random 32-character lower-case
		// alphanumeric string, not a counter, password, or permission grant. Drafts
		// have an empty token. Every successful submit/resubmit generates a new one
		// in the workflow transaction; recall and rejection clear it with approval
		// data. Approval retains it until creation transfers the approval evidence
		// and removes the request. Do not autogenerate it on record creation/update:
		// reads and ordinary saves must not create a new submission identity.
		//
		// Clearing stored approvals alone cannot cancel a checked checklist still
		// held by an approver's old browser page. Without a new submission identity,
		// recall followed by resubmission of unchanged data/PDF would produce the
		// same review token and accept that old page's approval. requestReviewToken
		// includes submission_token in its digest; approval and creation compare
		// that digest in their transaction. The attachment retains the approved
		// digest, so the final evidence also identifies the reviewed submission.
		// Request version still identifies business edits, and last_edited still
		// controls abandonment age. Neither changes merely to issue this token.
		if err := json.Unmarshal([]byte(`{
  "new": [
    {
      "name": "project_authorization_props",
      "id": "pa3c109497089ae",
      "type": "base",
      "fields": [
        {
          "autogeneratePattern": "[a-z0-9]{15}",
          "hidden": false,
          "id": "text3208210256",
          "max": 15,
          "min": 15,
          "name": "id",
          "pattern": "^[a-z0-9]+$",
          "presentable": false,
          "primaryKey": true,
          "required": true,
          "system": true,
          "type": "text"
        },
        {
          "hidden": false,
          "id": "autodate2990389176",
          "name": "created",
          "onCreate": true,
          "onUpdate": false,
          "presentable": false,
          "system": false,
          "type": "autodate"
        },
        {
          "hidden": false,
          "id": "autodate3332085495",
          "name": "updated",
          "onCreate": true,
          "onUpdate": true,
          "presentable": false,
          "system": false,
          "type": "autodate"
        },
        {
          "name": "user_claim",
          "type": "relation",
          "id": "pa1382336ab1b3f",
          "collectionId": "pmxhrqhngh60icm",
          "maxSelect": 1,
          "required": true,
          "cascadeDelete": true
        },
        {
          "name": "max_value",
          "type": "number",
          "id": "pa444da5b0eb3c6",
          "onlyInt": true,
          "min": 1,
          "required": true,
          "max": 9007199254740991
        }
      ],
      "listRule": "@request.auth.id != '' && @request.auth.admin_profiles_via_uid.active = true && @request.auth.user_claims_via_uid.cid.name ?= 'admin'",
      "viewRule": "@request.auth.id != '' && @request.auth.admin_profiles_via_uid.active = true && @request.auth.user_claims_via_uid.cid.name ?= 'admin'",
      "createRule": "@request.auth.id != '' && @request.auth.admin_profiles_via_uid.active = true && @request.auth.user_claims_via_uid.cid.name ?= 'admin'",
      "updateRule": "@request.auth.id != '' && @request.auth.admin_profiles_via_uid.active = true && @request.auth.user_claims_via_uid.cid.name ?= 'admin'",
      "deleteRule": null,
      "indexes": [
        "CREATE UNIQUE INDEX idx_pa_props_claim ON project_authorization_props (user_claim)"
      ]
    },
    {
      "name": "job_requests",
      "id": "pa24826ebf13018",
      "type": "base",
      "fields": [
        {
          "autogeneratePattern": "[a-z0-9]{15}",
          "hidden": false,
          "id": "text3208210256",
          "max": 15,
          "min": 15,
          "name": "id",
          "pattern": "^[a-z0-9]+$",
          "presentable": false,
          "primaryKey": true,
          "required": true,
          "system": true,
          "type": "text"
        },
        {
          "hidden": false,
          "id": "autodate2990389176",
          "name": "created",
          "onCreate": true,
          "onUpdate": false,
          "presentable": false,
          "system": false,
          "type": "autodate"
        },
        {
          "hidden": false,
          "id": "autodate3332085495",
          "name": "updated",
          "onCreate": true,
          "onUpdate": true,
          "presentable": false,
          "system": false,
          "type": "autodate"
        },
        {
          "name": "creator",
          "type": "relation",
          "id": "paca0dd83451c62",
          "collectionId": "_pb_users_auth_",
          "maxSelect": 1,
          "required": true,
          "cascadeDelete": false
        },
        {
          "name": "status",
          "type": "select",
          "id": "pa48a3661d84647",
          "values": [
            "Draft",
            "Awaiting approval",
            "Awaiting creation",
            "Rejected",
            "Recalled"
          ],
          "maxSelect": 1,
          "required": true
        },
        {
          "name": "project_status",
          "type": "select",
          "id": "pa57856cedbec1f",
          "values": [
            "Active",
            "Closed"
          ],
          "maxSelect": 1
        },
        {
          "name": "version",
          "type": "number",
          "id": "pac692273deb277",
          "onlyInt": true,
          "min": 0,
          "required": false
        },
        {
          "name": "submission_token",
          "type": "text",
          "id": "pasubmissiontok",
          "max": 32,
          "pattern": "^[a-z0-9]{32}$",
          "required": false
        },
        {
          "name": "approver",
          "type": "relation",
          "id": "paca76a54021477",
          "collectionId": "_pb_users_auth_",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "submitted",
          "type": "date",
          "id": "pafda43783be8e0"
        },
        {
          "name": "rejected",
          "type": "date",
          "id": "pa1f087a5954f62"
        },
        {
          "name": "last_edited",
          "type": "date",
          "id": "pa6bd3475120b42"
        },
        {
          "name": "rejector",
          "type": "relation",
          "id": "pa0e6fbb29e3a62",
          "collectionId": "_pb_users_auth_",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "rejection_reason",
          "type": "text",
          "id": "pa63577eda3763f"
        },
        {
          "name": "client",
          "type": "relation",
          "id": "pad2a04d71301a8",
          "collectionId": "1v6i9rrpniuatcx",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "job_owner",
          "type": "relation",
          "id": "pa53f7ad120bbd1",
          "collectionId": "1v6i9rrpniuatcx",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "contact",
          "type": "relation",
          "id": "pa1a73af9e7ae00",
          "collectionId": "3v7wxidd2f9yhf9",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "manager",
          "type": "relation",
          "id": "pa1a8565a9dc720",
          "collectionId": "_pb_users_auth_",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "alternate_manager",
          "type": "relation",
          "id": "pab695cf4098395",
          "collectionId": "_pb_users_auth_",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "branch",
          "type": "relation",
          "id": "pa10d735e581f1e",
          "collectionId": "pbc_2536409462",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "rate_sheet",
          "type": "relation",
          "id": "pabf3f001d90d28",
          "collectionId": "pbc_126575313",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "parent",
          "type": "relation",
          "id": "pad8fd39d0bbdd2",
          "collectionId": "yovqzrnnomp0lkx",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "proposal",
          "type": "relation",
          "id": "pa426144f4dedc1",
          "collectionId": "yovqzrnnomp0lkx",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "invoicing_information",
          "type": "relation",
          "id": "pa1d42c050a86aa",
          "collectionId": "pa992bc769dcf8b",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "description",
          "type": "text",
          "id": "pacb329146a0dd0"
        },
        {
          "name": "location",
          "type": "text",
          "id": "pa6469ac84a8974"
        },
        {
          "name": "client_po",
          "type": "text",
          "id": "pa9065ed984a6ea"
        },
        {
          "name": "client_reference_number",
          "type": "text",
          "id": "pab4220650eaa63"
        },
        {
          "name": "project_award_date",
          "type": "text",
          "id": "paea56ec1a84d65"
        },
        {
          "name": "project_completion_date",
          "type": "text",
          "id": "pab9ae4f2f574f2"
        },
        {
          "name": "project_value",
          "type": "number",
          "id": "pacef2de974a5c0",
          "onlyInt": true,
          "min": 0,
          "required": false
        },
        {
          "name": "time_and_materials",
          "type": "bool",
          "id": "pa2cebafa0630d1"
        },
        {
          "name": "time_and_materials_set",
          "type": "bool",
          "id": "pac486b19db7132"
        },
        {
          "name": "fn_agreement_answer",
          "type": "select",
          "id": "pa3b8b49e3a1a1b",
          "values": [
            "Yes",
            "No"
          ],
          "maxSelect": 1
        }
      ],
      "listRule": "@request.auth.id != '' && @request.auth.admin_profiles_via_uid.active = true && (creator = @request.auth.id || @request.auth.user_claims_via_uid.cid.name ?= 'job' || (approver = @request.auth.id && @request.auth.user_claims_via_uid.cid.name ?= 'project_authorization'))",
      "viewRule": "@request.auth.id != '' && @request.auth.admin_profiles_via_uid.active = true && (creator = @request.auth.id || @request.auth.user_claims_via_uid.cid.name ?= 'job' || (approver = @request.auth.id && @request.auth.user_claims_via_uid.cid.name ?= 'project_authorization'))",
      "createRule": null,
      "updateRule": null,
      "deleteRule": null,
      "indexes": [
        "CREATE INDEX idx_requests_creator_status ON job_requests (creator,status)",
        "CREATE INDEX idx_requests_approver_status ON job_requests (approver,status)",
        "CREATE INDEX idx_requests_status_submitted ON job_requests (status,submitted)",
        "CREATE INDEX idx_requests_proposal ON job_requests (proposal)",
        "CREATE INDEX idx_requests_parent ON job_requests (parent)",
        "CREATE INDEX idx_requests_client ON job_requests (client)",
        "CREATE INDEX idx_requests_last_edited ON job_requests (last_edited)"
      ]
    },
    {
      "name": "job_attachments",
      "id": "pa252800ec80865",
      "type": "base",
      "fields": [
        {
          "autogeneratePattern": "[a-z0-9]{15}",
          "hidden": false,
          "id": "text3208210256",
          "max": 15,
          "min": 15,
          "name": "id",
          "pattern": "^[a-z0-9]+$",
          "presentable": false,
          "primaryKey": true,
          "required": true,
          "system": true,
          "type": "text"
        },
        {
          "hidden": false,
          "id": "autodate2990389176",
          "name": "created",
          "onCreate": true,
          "onUpdate": false,
          "presentable": false,
          "system": false,
          "type": "autodate"
        },
        {
          "hidden": false,
          "id": "autodate3332085495",
          "name": "updated",
          "onCreate": true,
          "onUpdate": true,
          "presentable": false,
          "system": false,
          "type": "autodate"
        },
        {
          "name": "request",
          "type": "relation",
          "id": "pa088e29b0ab007",
          "collectionId": "pa24826ebf13018",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "job",
          "type": "relation",
          "id": "pa2eda8ad1632af",
          "collectionId": "yovqzrnnomp0lkx",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "document",
          "type": "file",
          "id": "pa4f8278c89ad16",
          "maxSelect": 1,
          "maxSize": 20971520,
          "mimeTypes": [
            "application/pdf"
          ],
          "required": true,
          "protected": true
        },
        {
          "name": "document_hash",
          "type": "text",
          "id": "pa4b1600eae8f79",
          "required": true,
          "min": 64,
          "max": 64,
          "pattern": "^[a-f0-9]{64}$"
        },
        {
          "name": "uploader",
          "type": "relation",
          "id": "pa41ae8cf7f1ec7",
          "collectionId": "_pb_users_auth_",
          "maxSelect": 1,
          "required": true,
          "cascadeDelete": false
        },
        {
          "name": "uploaded",
          "type": "date",
          "id": "paf40369e790b9d"
        },
        {
          "name": "project_value",
          "type": "number",
          "id": "pacef2de974a5c0",
          "onlyInt": true,
          "min": 0,
          "required": false
        },
        {
          "name": "status",
          "type": "select",
          "id": "pa48a3661d84647",
          "values": [
            "Pending",
            "Approved",
            "Rejected"
          ],
          "maxSelect": 1,
          "required": true
        },
        {
          "name": "assigned_approver",
          "type": "relation",
          "id": "pa11ebb6dbcc2fd",
          "collectionId": "_pb_users_auth_",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "approver",
          "type": "relation",
          "id": "paca76a54021477",
          "collectionId": "_pb_users_auth_",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "approved",
          "type": "date",
          "id": "pac9560d94ee198"
        },
        {
          "name": "previous_project_value",
          "type": "number",
          "id": "paedef1280310d9",
          "onlyInt": true,
          "required": false
        },
        {
          "name": "approval_max_value",
          "type": "number",
          "id": "pac7854ab00057b",
          "onlyInt": true,
          "min": 0,
          "required": false
        },
        {
          "name": "request_version",
          "type": "number",
          "id": "pad4050089e9be3",
          "onlyInt": true,
          "min": 0,
          "required": false
        },
        {
          "name": "approval_sequence",
          "type": "number",
          "id": "pa7bce2b2c05580",
          "onlyInt": true,
          "min": 0,
          "required": false
        },
        {
          "name": "version",
          "type": "number",
          "id": "pac692273deb277",
          "onlyInt": true,
          "min": 0,
          "required": false
        },
        {
          "name": "initial_approval",
          "type": "bool",
          "id": "pa9b86c36a76af6"
        },
        {
          "name": "checklist_version",
          "type": "number",
          "id": "pa5167723227caa",
          "onlyInt": true,
          "min": 0,
          "required": false
        },
        {
          "name": "checklist_answers",
          "type": "json",
          "id": "pa4006f9ce47288"
        },
        {
          "name": "rejector",
          "type": "relation",
          "id": "pa0e6fbb29e3a62",
          "collectionId": "_pb_users_auth_",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "rejected",
          "type": "date",
          "id": "pa1f087a5954f62"
        },
        {
          "name": "rejection_reason",
          "type": "text",
          "id": "pa63577eda3763f"
        },
        {
          "name": "review_token",
          "id": "pareviewtoken001",
          "type": "text"
        }
      ],
      "listRule": "@request.auth.id != '' && @request.auth.admin_profiles_via_uid.active = true && ((request != '' && (request.creator = @request.auth.id || request.approver = @request.auth.id && @request.auth.user_claims_via_uid.cid.name ?= 'project_authorization' || @request.auth.user_claims_via_uid.cid.name ?= 'job')) || (job != '' && (@request.auth.user_claims_via_uid.cid.name ?= 'job' || @request.auth.user_claims_via_uid.cid.name ?= 'admin' || assigned_approver = @request.auth.id && @request.auth.user_claims_via_uid.cid.name ?= 'project_authorization')))",
      "viewRule": "@request.auth.id != '' && @request.auth.admin_profiles_via_uid.active = true && ((request != '' && (request.creator = @request.auth.id || request.approver = @request.auth.id && @request.auth.user_claims_via_uid.cid.name ?= 'project_authorization' || @request.auth.user_claims_via_uid.cid.name ?= 'job')) || (job != '' && (@request.auth.user_claims_via_uid.cid.name ?= 'job' || @request.auth.user_claims_via_uid.cid.name ?= 'admin' || assigned_approver = @request.auth.id && @request.auth.user_claims_via_uid.cid.name ?= 'project_authorization')))",
      "createRule": null,
      "updateRule": null,
      "deleteRule": null,
      "indexes": [
        "CREATE UNIQUE INDEX idx_attachment_hash ON job_attachments (document_hash)",
        "CREATE UNIQUE INDEX idx_attachment_request ON job_attachments (request) WHERE request != ''",
        "CREATE UNIQUE INDEX idx_attachment_pending_job ON job_attachments (job) WHERE job != '' AND status != 'Approved'",
        "CREATE UNIQUE INDEX idx_attachment_sequence ON job_attachments (job,approval_sequence) WHERE job != '' AND status = 'Approved'",
        "CREATE INDEX idx_attachment_queue ON job_attachments (assigned_approver,status)"
      ]
    },
    {
      "name": "job_request_allocations",
      "id": "pa56b39d10089a8",
      "type": "base",
      "fields": [
        {
          "autogeneratePattern": "[a-z0-9]{15}",
          "hidden": false,
          "id": "text3208210256",
          "max": 15,
          "min": 15,
          "name": "id",
          "pattern": "^[a-z0-9]+$",
          "presentable": false,
          "primaryKey": true,
          "required": true,
          "system": true,
          "type": "text"
        },
        {
          "hidden": false,
          "id": "autodate2990389176",
          "name": "created",
          "onCreate": true,
          "onUpdate": false,
          "presentable": false,
          "system": false,
          "type": "autodate"
        },
        {
          "hidden": false,
          "id": "autodate3332085495",
          "name": "updated",
          "onCreate": true,
          "onUpdate": true,
          "presentable": false,
          "system": false,
          "type": "autodate"
        },
        {
          "name": "request",
          "type": "relation",
          "id": "pa088e29b0ab007",
          "collectionId": "pa24826ebf13018",
          "maxSelect": 1,
          "required": true,
          "cascadeDelete": true
        },
        {
          "name": "division",
          "type": "relation",
          "id": "pa4fd95a301ae37",
          "collectionId": "3esdddggow6dykr",
          "maxSelect": 1,
          "required": true,
          "cascadeDelete": false
        },
        {
          "name": "hours",
          "type": "number",
          "id": "pa5aedee0964c6b",
          "min": 0
        }
      ],
      "listRule": "@request.auth.id != '' && @request.auth.admin_profiles_via_uid.active = true && (request.creator = @request.auth.id || @request.auth.user_claims_via_uid.cid.name ?= 'job' || (request.approver = @request.auth.id && @request.auth.user_claims_via_uid.cid.name ?= 'project_authorization'))",
      "viewRule": "@request.auth.id != '' && @request.auth.admin_profiles_via_uid.active = true && (request.creator = @request.auth.id || @request.auth.user_claims_via_uid.cid.name ?= 'job' || (request.approver = @request.auth.id && @request.auth.user_claims_via_uid.cid.name ?= 'project_authorization'))",
      "createRule": null,
      "updateRule": null,
      "deleteRule": null,
      "indexes": [
        "CREATE UNIQUE INDEX idx_request_division ON job_request_allocations (request,division)"
      ]
    }
  ],
  "updates": {
    "jobs": {
      "fields": [
        {
          "name": "request_creator",
          "type": "relation",
          "id": "paf06e4849ddce7",
          "collectionId": "_pb_users_auth_",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "created_by",
          "type": "relation",
          "id": "pa71a8aeb31196d",
          "collectionId": "_pb_users_auth_",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "request_created",
          "type": "date",
          "id": "pa10d6b07938981"
        },
        {
          "name": "request_submitted",
          "type": "date",
          "id": "pa18ad9b5878fd2"
        },
        {
          "name": "creation_completed",
          "type": "date",
          "id": "pab922266da57ad"
        }
      ],
      "updateRule": "@request.auth.id != '' && @request.auth.admin_profiles_via_uid.active = true && (@request.auth.user_claims_via_uid.cid.name ?= 'job' || (number ~ 'P%' && (manager = @request.auth.id || alternate_manager = @request.auth.id))) && @request.body.number:changed = false && (number ~ 'P%' || (@request.body.client:changed = false && @request.body.project_value:changed = false))"
    }
  }
}`), &spec); err != nil {
			return err
		}
		// Allocation writes must use the job save routes and their validation.
		allocations, err := app.FindCollectionByNameOrId("job_time_allocations")
		if err != nil {
			return err
		}
		allocations.CreateRule, allocations.UpdateRule, allocations.DeleteRule = nil, nil, nil
		if err := app.Save(allocations); err != nil {
			return err
		}

		for _, raw := range spec.New {
			c := core.NewBaseCollection("")
			if err := json.Unmarshal(raw, c); err != nil {
				return err
			}
			if err := app.Save(c); err != nil {
				return fmt.Errorf("create %s: %w", c.Name, err)
			}
		}

		for name, raw := range spec.Updates {
			c, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			var changes struct {
				Fields     core.FieldsList `json:"fields"`
				UpdateRule *string         `json:"updateRule"`
			}
			if err = json.Unmarshal(raw, &changes); err != nil {
				return err
			}
			for _, field := range changes.Fields {
				if c.Fields.GetByName(field.GetName()) == nil {
					c.Fields.Add(field)
				}
			}
			c.UpdateRule = changes.UpdateRule
			if c.DeleteRule != nil {
				c.DeleteRule = pointerString("(" + *c.DeleteRule + ") && @collection.job_attachments.job != id")
			}
			if err = app.Save(c); err != nil {
				return err
			}
		}

		// Client setup already exists. Broaden creation for requesters and add
		// request references only after their collections have been created.
		invoicing, err := app.FindCollectionByNameOrId("client_invoicing_information")
		if err != nil {
			return err
		}
		invoicing.CreateRule = pointerString("@request.auth.id != '' && @request.auth.admin_profiles_via_uid.active = true && (@request.auth.user_claims_via_uid.cid.name ?= 'job' || @request.auth.user_claims_via_uid.cid.name ?= 'time')")
		invoicing.DeleteRule = pointerString("@request.auth.id != '' && @request.auth.admin_profiles_via_uid.active = true && @request.auth.user_claims_via_uid.cid.name ?= 'job' && @collection.jobs.invoicing_information != id && @collection.job_requests.invoicing_information != id")
		if err = app.Save(invoicing); err != nil {
			return err
		}
		for name, relation := range map[string]string{"clients": "client", "client_contacts": "contact"} {
			c, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			if name == "client_contacts" && c.Fields.GetByName("no_email") == nil {
				// Email stays optional: many clients have none. Staff mark those contacts instead.
				c.Fields.Add(&core.BoolField{Id: "bool_no_email", Name: "no_email"})
			}
			if c.DeleteRule != nil {
				profileGuard := " && @collection.client_invoicing_information." + relation + " != id"
				c.DeleteRule = pointerString(strings.TrimSuffix(*c.DeleteRule, profileGuard) + " && @collection.job_requests." + relation + " != id" + profileGuard)
			}
			if err = app.Save(c); err != nil {
				return err
			}
		}

		claims, err := app.FindCollectionByNameOrId("claims")
		if err != nil {
			return err
		}
		claim := core.NewRecord(claims)
		claim.Id = "projectauth0001"
		claim.Set("name", "project_authorization")
		claim.Set("description", "Approve full project values within the assigned limit")
		if err = app.Save(claim); err != nil {
			return err
		}
		config, err := app.FindFirstRecordByData("app_config", "key", "jobs")
		if errors.Is(err, sql.ErrNoRows) {
			c, findErr := app.FindCollectionByNameOrId("app_config")
			if findErr != nil {
				return findErr
			}
			config = core.NewRecord(c)
			config.Set("key", "jobs")
			config.Set("value", map[string]any{})
		} else if err != nil {
			return err
		}
		value := map[string]any{}
		if err = json.Unmarshal([]byte(config.GetString("value")), &value); err != nil {
			return err
		}
		value["require_job_request_approval"] = false
		config.Set("value", value)
		config.Set("description", "create_edit_absorb (default true) enables job and proposal creation/editing, manual renumbering, and client/contact merges.\n\nrequire_job_request_approval (default false) requires approved requests for new projects and subjobs; when false, job claim holders can create projects directly. Proposals are exempt from request approval. Project-value changes on existing jobs require approval in either request-flag state.")
		if err = app.Save(config); err != nil {
			return err
		}

		notificationConfig, err := app.FindFirstRecordByData("app_config", "key", "notifications")
		if errors.Is(err, sql.ErrNoRows) {
			c, findErr := app.FindCollectionByNameOrId("app_config")
			if findErr != nil {
				return findErr
			}
			notificationConfig = core.NewRecord(c)
			notificationConfig.Set("key", "notifications")
			notificationConfig.Set("description", "Notification feature settings")
			notificationConfig.Set("value", map[string]any{})
		} else if err != nil {
			return err
		}
		notificationFlags := map[string]any{}
		if err = json.Unmarshal([]byte(notificationConfig.GetString("value")), &notificationFlags); err != nil {
			return err
		}
		notificationFlags["job_request_workflow"] = true
		notificationFlags["job_value_workflow"] = true
		notificationConfig.Set("value", notificationFlags)
		if err = app.Save(notificationConfig); err != nil {
			return err
		}
		templates, err := app.FindCollectionByNameOrId("notification_templates")
		if err != nil {
			return err
		}
		template := core.NewRecord(templates)
		template.Id = "panotification1"
		template.Set("code", "job_request_workflow")
		template.Set("description", "Job request and value approval notices")
		template.Set("subject", "Project authorization: {{.Action}}")
		template.Set("text_email", "Request {{.RequestID}}: {{.Description}}\n{{.Action}}\n\n{{.ActionURL}}")
		err = app.Save(template)
		if err != nil {
			return err
		}
		valueTemplate := core.NewRecord(templates)
		valueTemplate.Id = "pavaluenotice01"
		valueTemplate.Set("code", "job_value_workflow")
		valueTemplate.Set("description", "Project value approval notices")
		valueTemplate.Set("subject", "Project {{.JobNumber}}: {{.Action}}")
		valueTemplate.Set("text_email", "{{.Description}}\n{{.Action}}\n\n{{.ActionURL}}")
		err = app.Save(valueTemplate)

		return err
	}, func(app core.App) error {
		return fmt.Errorf("project request workflow cannot be reversed; restore a compatible backup")
	})
}
