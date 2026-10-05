/*
Client setup: first stage of the project_authorization rollout.

Schema changes:
  - Add client_invoicing_information with columns id, created, updated, client,
    contact, creator, name, invoicing_instructions, fax. Client and contact are
    required single relations; creator is optional. Store reusable invoice
    recipients and instructions, with creator and timestamp audit fields.
  - Add client_invoicing_information indexes idx_invoice_client (client) and
    idx_invoice_contact (contact). Support client lists and contact references.
  - Add jobs.invoicing_information as an optional single relation, with index
    idx_jobs_invoicing_information. Link jobs to profiles without changing old
    job data. Project create/edit requirements are enforced by application code.
  - Add missing clients and client_contacts columns address, city, province_state,
    postal_code, country, phone as optional text. Store client and recipient
    address details. Keep existing fields and values when a column already exists.
  - Remove jobs.project_authorization_doc, project_authorization_doc_hash,
    pa_reviewer, pa_reviewed, pa_uploader, pa_uploaded, pa_rejector, pa_rejected,
    pa_rejection_reason, and index idx_jobs_project_authorization_doc_hash.
    Discard the retired authorization document and review data; do not archive it.

Collection rules:
  - Allow active users to read profiles; require the job claim to create, update,
    or delete them. Block deletion of profiles referenced by jobs. Keep shared
    billing details under client maintainer control and preserve job references.
  - Require an active account for client and contact writes, as for profiles.
    Extend client/contact delete rules to block deletion while profiles
    reference them. Prevent profiles from losing their client or invoice contact.
  - Remove retired authorization field guards from jobs.updateRule. Keep the
    remaining job permissions so ordinary job edits no longer depend on that flow.

Data and settings changes:
  - Remove enforce_project_authorization from app_config.value where key = jobs;
    replace that row's description with the create_edit_absorb setting description.
    Remove project_authorization_rejected from app_config.value where
    key = notifications. Keep other settings; skip either row if absent.
    Remove settings for the retired workflow without changing unrelated flags.
  - Delete notifications for the project_authorization_rejected template, then
    delete that notification_templates row. Remove obsolete rejection messages.
  - Do not create profiles or assign them to existing jobs. Staff must choose the
    correct invoice contact and instructions when they create or edit a project.

Rollback is refused because the removed authorization data cannot be recovered
by a down migration. Restore a compatible backup to return to the old schema.
*/
package migrations

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// This stage can ship without the request workflow collections.
		var spec struct {
			New     []json.RawMessage          `json:"new"`
			Updates map[string]json.RawMessage `json:"updates"`
		}
		if err := json.Unmarshal([]byte(`{
  "new": [
    {
      "name": "client_invoicing_information",
      "id": "pa992bc769dcf8b",
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
          "name": "client",
          "type": "relation",
          "id": "pad2a04d71301a8",
          "collectionId": "1v6i9rrpniuatcx",
          "maxSelect": 1,
          "required": true,
          "cascadeDelete": false
        },
        {
          "name": "contact",
          "type": "relation",
          "id": "pa1a73af9e7ae00",
          "collectionId": "3v7wxidd2f9yhf9",
          "maxSelect": 1,
          "required": true,
          "cascadeDelete": false
        },
        {
          "name": "creator",
          "type": "relation",
          "id": "painvoiceauthor",
          "collectionId": "_pb_users_auth_",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        },
        {
          "name": "name",
          "type": "text",
          "id": "painvoicename01",
          "max": 120
        },
        {
          "name": "invoicing_instructions",
          "type": "text",
          "id": "pa121e197a0d07a"
        },
        {
          "name": "fax",
          "type": "text",
          "id": "pada41abc0db70b"
        }
      ],
      "listRule": "@request.auth.id != '' && @request.auth.admin_profiles_via_uid.active = true",
      "viewRule": "@request.auth.id != '' && @request.auth.admin_profiles_via_uid.active = true",
      "createRule": "@request.auth.id != '' && @request.auth.admin_profiles_via_uid.active = true && @request.auth.user_claims_via_uid.cid.name ?= 'job'",
      "updateRule": "@request.auth.id != '' && @request.auth.admin_profiles_via_uid.active = true && @request.auth.user_claims_via_uid.cid.name ?= 'job'",
      "deleteRule": "@request.auth.id != '' && @request.auth.admin_profiles_via_uid.active = true && @request.auth.user_claims_via_uid.cid.name ?= 'job' && @collection.jobs.invoicing_information != id",
      "indexes": [
        "CREATE INDEX idx_invoice_client ON client_invoicing_information (client)",
        "CREATE INDEX idx_invoice_contact ON client_invoicing_information (contact)"
      ]
    }
  ],
  "updates": {
    "jobs": {
      "fields": [
        {
          "name": "invoicing_information",
          "type": "relation",
          "id": "pa1d42c050a86aa",
          "collectionId": "pa992bc769dcf8b",
          "maxSelect": 1,
          "required": false,
          "cascadeDelete": false
        }
      ]
    },
    "clients": {
      "fields": [
        {
          "name": "address",
          "type": "text",
          "id": "pac662180230cad"
        },
        {
          "name": "city",
          "type": "text",
          "id": "pa2c54892c40a17"
        },
        {
          "name": "province_state",
          "type": "text",
          "id": "pad2953b0a25758"
        },
        {
          "name": "postal_code",
          "type": "text",
          "id": "pab3407f5ecb6ee"
        },
        {
          "name": "country",
          "type": "text",
          "id": "pa8e68b3e5af636"
        },
        {
          "name": "phone",
          "type": "text",
          "id": "paf6be6ca910984"
        }
      ]
    },
    "client_contacts": {
      "fields": [
        {
          "name": "address",
          "type": "text",
          "id": "pac662180230cad"
        },
        {
          "name": "city",
          "type": "text",
          "id": "pa2c54892c40a17"
        },
        {
          "name": "province_state",
          "type": "text",
          "id": "pad2953b0a25758"
        },
        {
          "name": "postal_code",
          "type": "text",
          "id": "pab3407f5ecb6ee"
        },
        {
          "name": "country",
          "type": "text",
          "id": "pa8e68b3e5af636"
        },
        {
          "name": "phone",
          "type": "text",
          "id": "paf6be6ca910984"
        }
      ]
    }
  }
}`), &spec); err != nil {
			return err
		}
		for _, raw := range spec.New {
			c := core.NewBaseCollection("")
			if err := json.Unmarshal(raw, c); err != nil {
				return err
			}
			// The jobs relation must exist before this rule can be validated.
			c.DeleteRule = nil
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
				Fields core.FieldsList `json:"fields"`
			}
			if err = json.Unmarshal(raw, &changes); err != nil {
				return err
			}
			for _, field := range changes.Fields {
				if c.Fields.GetByName(field.GetName()) == nil {
					c.Fields.Add(field)
				}
			}
			if name == "jobs" {
				for _, field := range []string{"project_authorization_doc", "project_authorization_doc_hash", "pa_reviewer", "pa_reviewed", "pa_uploader", "pa_uploaded", "pa_rejector", "pa_rejected", "pa_rejection_reason"} {
					c.Fields.RemoveByName(field)
				}
				c.RemoveIndex("idx_jobs_project_authorization_doc_hash")
				c.AddIndex("idx_jobs_invoicing_information", false, "invoicing_information", "")
				// Remove only the retired field guards; keep the existing job permissions.
				if c.UpdateRule != nil {
					c.UpdateRule = pointerString(unwrapProjectAuthorizationJobsUpdateRule(unwrapProjectAuthorizationAuditJobsUpdateRule(*c.UpdateRule)))
				}
			} else {
				// Client maintenance, like profile maintenance, requires an active account.
				const active = " && @request.auth.admin_profiles_via_uid.active = true"
				relation := "client"
				if name == "client_contacts" {
					relation = "contact"
				}
				if c.CreateRule != nil {
					c.CreateRule = pointerString(*c.CreateRule + active)
				}
				if c.UpdateRule != nil {
					c.UpdateRule = pointerString(*c.UpdateRule + active)
				}
				if c.DeleteRule != nil {
					c.DeleteRule = pointerString("(" + *c.DeleteRule + ")" + active + " && @collection.client_invoicing_information." + relation + " != id")
				}
			}
			if err = app.Save(c); err != nil {
				return err
			}
		}
		for _, raw := range spec.New {
			var desired core.Collection
			if err := json.Unmarshal(raw, &desired); err != nil {
				return err
			}
			c, err := app.FindCollectionByNameOrId(desired.Name)
			if err != nil {
				return err
			}
			c.DeleteRule = desired.DeleteRule
			if err = app.Save(c); err != nil {
				return err
			}
		}
		retiredSettings := map[string]string{
			"jobs":          "enforce_project_authorization",
			"notifications": "project_authorization_rejected",
		}
		for key, setting := range retiredSettings {
			config, err := app.FindFirstRecordByData("app_config", "key", key)
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			if err != nil {
				return err
			}
			value := map[string]any{}
			if err := json.Unmarshal([]byte(config.GetString("value")), &value); err != nil {
				return err
			}
			delete(value, setting)
			config.Set("value", value)
			if key == "jobs" {
				config.Set("description", "create_edit_absorb (default true) enables job and proposal creation/editing, manual renumbering, and client/contact merges.")
			}
			// The retired setting may have been the only key. An empty object is
			// valid here because missing settings use their application defaults.
			if err := app.SaveNoValidate(config); err != nil {
				return err
			}
		}
		if _, err := app.DB().NewQuery("DELETE FROM notifications WHERE template = (SELECT id FROM notification_templates WHERE code = {:code})").Bind(dbx.Params{"code": "project_authorization_rejected"}).Execute(); err != nil {
			return err
		}
		_, err := app.DB().NewQuery("DELETE FROM notification_templates WHERE code = {:code}").Bind(dbx.Params{"code": "project_authorization_rejected"}).Execute()
		return err
	}, func(app core.App) error {
		return fmt.Errorf("legacy project authorization cleanup cannot be reversed; restore a compatible backup")
	})
}
