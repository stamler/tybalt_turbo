package migrations

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const projectAuthorizationFieldsGuard = "@request.body.project_authorization_doc:changed = false &&\n@request.body.project_authorization_doc_hash:changed = false &&\n@request.body.pa_reviewer:changed = false &&\n@request.body.pa_reviewed:changed = false"

func init() {
	m.Register(func(app core.App) error {
		branches, err := app.FindCollectionByNameOrId("branches")
		if err != nil {
			return err
		}
		if branches.Fields.GetByName("manager") == nil {
			if err := branches.Fields.AddMarshaledJSON([]byte(`{
				"cascadeDelete": false,
				"collectionId": "_pb_users_auth_",
				"hidden": false,
				"id": "rel1780417194a",
				"maxSelect": 1,
				"minSelect": 0,
				"name": "manager",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "relation"
			}`)); err != nil {
				return err
			}
		}
		if err := app.Save(branches); err != nil {
			return err
		}

		jobs, err := app.FindCollectionByNameOrId("jobs")
		if err != nil {
			return err
		}
		if jobs.Fields.GetByName("project_authorization_doc") == nil {
			if err := jobs.Fields.AddMarshaledJSON([]byte(`{
				"hidden": false,
				"id": "file1780417194a",
				"maxSelect": 1,
				"maxSize": 20971520,
				"mimeTypes": ["application/pdf"],
				"name": "project_authorization_doc",
				"presentable": false,
				"protected": false,
				"required": false,
				"system": false,
				"thumbs": null,
				"type": "file"
			}`)); err != nil {
				return err
			}
		}
		if jobs.Fields.GetByName("project_authorization_doc_hash") == nil {
			if err := jobs.Fields.AddMarshaledJSON([]byte(`{
				"autogeneratePattern": "",
				"hidden": false,
				"id": "text1780417194a",
				"max": 64,
				"min": 0,
				"name": "project_authorization_doc_hash",
				"pattern": "^[a-f0-9]{64}$|^$",
				"presentable": false,
				"primaryKey": false,
				"required": false,
				"system": false,
				"type": "text"
			}`)); err != nil {
				return err
			}
		}
		if jobs.Fields.GetByName("pa_reviewer") == nil {
			if err := jobs.Fields.AddMarshaledJSON([]byte(`{
				"cascadeDelete": false,
				"collectionId": "_pb_users_auth_",
				"hidden": false,
				"id": "rel1780417194b",
				"maxSelect": 1,
				"minSelect": 0,
				"name": "pa_reviewer",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "relation"
			}`)); err != nil {
				return err
			}
		}
		if jobs.Fields.GetByName("pa_reviewed") == nil {
			if err := jobs.Fields.AddMarshaledJSON([]byte(`{
				"hidden": false,
				"id": "date1780417194a",
				"max": "",
				"min": "",
				"name": "pa_reviewed",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "date"
			}`)); err != nil {
				return err
			}
		}
		jobs.AddIndex("idx_jobs_project_authorization_doc_hash", true, "`project_authorization_doc_hash`", "`project_authorization_doc_hash` != ''")
		jobs.UpdateRule = pointerString(wrapProjectAuthorizationJobsUpdateRule(pointerValue(jobs.UpdateRule)))
		return app.Save(jobs)
	}, func(app core.App) error {
		branches, err := app.FindCollectionByNameOrId("branches")
		if err != nil {
			return err
		}
		branches.Fields.RemoveByName("manager")
		if err := app.Save(branches); err != nil {
			return err
		}

		jobs, err := app.FindCollectionByNameOrId("jobs")
		if err != nil {
			return err
		}
		jobs.RemoveIndex("idx_jobs_project_authorization_doc_hash")
		jobs.Fields.RemoveByName("project_authorization_doc")
		jobs.Fields.RemoveByName("project_authorization_doc_hash")
		jobs.Fields.RemoveByName("pa_reviewer")
		jobs.Fields.RemoveByName("pa_reviewed")
		jobs.UpdateRule = pointerString(unwrapProjectAuthorizationJobsUpdateRule(pointerValue(jobs.UpdateRule)))
		return app.Save(jobs)
	})
}

func wrapProjectAuthorizationJobsUpdateRule(rule string) string {
	rule = strings.TrimSpace(rule)
	if strings.Contains(rule, projectAuthorizationFieldsGuard) {
		return rule
	}
	if rule == "" {
		return projectAuthorizationFieldsGuard
	}
	return projectAuthorizationFieldsGuard + " &&\n(\n" + rule + "\n)"
}

func unwrapProjectAuthorizationJobsUpdateRule(rule string) string {
	rule = strings.TrimSpace(rule)
	prefix := projectAuthorizationFieldsGuard + " &&\n(\n"
	if strings.HasPrefix(rule, prefix) && strings.HasSuffix(rule, "\n)") {
		return strings.TrimSuffix(strings.TrimPrefix(rule, prefix), "\n)")
	}
	if rule == projectAuthorizationFieldsGuard {
		return ""
	}
	return rule
}

func pointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func pointerString(value string) *string {
	return &value
}
