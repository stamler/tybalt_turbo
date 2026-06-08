package migrations

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const projectAuthorizationAuditFieldsGuard = "@request.body.pa_uploader:changed = false &&\n@request.body.pa_uploaded:changed = false &&\n@request.body.pa_rejector:changed = false &&\n@request.body.pa_rejected:changed = false &&\n@request.body.pa_rejection_reason:changed = false"

func init() {
	m.Register(func(app core.App) error {
		jobs, err := app.FindCollectionByNameOrId("jobs")
		if err != nil {
			return err
		}
		if jobs.Fields.GetByName("pa_uploader") == nil {
			if err := jobs.Fields.AddMarshaledJSON([]byte(`{
				"cascadeDelete": false,
				"collectionId": "_pb_users_auth_",
				"hidden": false,
				"id": "rel1780930403a",
				"maxSelect": 1,
				"minSelect": 0,
				"name": "pa_uploader",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "relation"
			}`)); err != nil {
				return err
			}
		}
		if jobs.Fields.GetByName("pa_uploaded") == nil {
			if err := jobs.Fields.AddMarshaledJSON([]byte(`{
				"hidden": false,
				"id": "date1780930403a",
				"max": "",
				"min": "",
				"name": "pa_uploaded",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "date"
			}`)); err != nil {
				return err
			}
		}
		if jobs.Fields.GetByName("pa_rejector") == nil {
			if err := jobs.Fields.AddMarshaledJSON([]byte(`{
				"cascadeDelete": false,
				"collectionId": "_pb_users_auth_",
				"hidden": false,
				"id": "rel1780930403b",
				"maxSelect": 1,
				"minSelect": 0,
				"name": "pa_rejector",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "relation"
			}`)); err != nil {
				return err
			}
		}
		if jobs.Fields.GetByName("pa_rejected") == nil {
			if err := jobs.Fields.AddMarshaledJSON([]byte(`{
				"hidden": false,
				"id": "date1780930403b",
				"max": "",
				"min": "",
				"name": "pa_rejected",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "date"
			}`)); err != nil {
				return err
			}
		}
		if jobs.Fields.GetByName("pa_rejection_reason") == nil {
			if err := jobs.Fields.AddMarshaledJSON([]byte(`{
				"autogeneratePattern": "",
				"hidden": false,
				"id": "text1780930403a",
				"max": 2000,
				"min": 0,
				"name": "pa_rejection_reason",
				"pattern": "",
				"presentable": false,
				"primaryKey": false,
				"required": false,
				"system": false,
				"type": "text"
			}`)); err != nil {
				return err
			}
		}
		jobs.UpdateRule = pointerString(wrapProjectAuthorizationAuditJobsUpdateRule(pointerValue(jobs.UpdateRule)))
		return app.Save(jobs)
	}, func(app core.App) error {
		jobs, err := app.FindCollectionByNameOrId("jobs")
		if err != nil {
			return err
		}
		jobs.Fields.RemoveByName("pa_uploader")
		jobs.Fields.RemoveByName("pa_uploaded")
		jobs.Fields.RemoveByName("pa_rejector")
		jobs.Fields.RemoveByName("pa_rejected")
		jobs.Fields.RemoveByName("pa_rejection_reason")
		jobs.UpdateRule = pointerString(unwrapProjectAuthorizationAuditJobsUpdateRule(pointerValue(jobs.UpdateRule)))
		return app.Save(jobs)
	})
}

func wrapProjectAuthorizationAuditJobsUpdateRule(rule string) string {
	rule = strings.TrimSpace(rule)
	if strings.Contains(rule, projectAuthorizationAuditFieldsGuard) {
		return rule
	}
	if rule == "" {
		return projectAuthorizationAuditFieldsGuard
	}
	return projectAuthorizationAuditFieldsGuard + " &&\n(\n" + rule + "\n)"
}

func unwrapProjectAuthorizationAuditJobsUpdateRule(rule string) string {
	rule = strings.TrimSpace(rule)
	prefix := projectAuthorizationAuditFieldsGuard + " &&\n(\n"
	if strings.HasPrefix(rule, prefix) && strings.HasSuffix(rule, "\n)") {
		return strings.TrimSuffix(strings.TrimPrefix(rule, prefix), "\n)")
	}
	if rule == projectAuthorizationAuditFieldsGuard {
		return ""
	}
	return rule
}
