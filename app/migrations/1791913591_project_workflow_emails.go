package migrations

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

type projectWorkflowEmail struct {
	id, code, description, subject, text string
}

// projectWorkflowEmails are the project request and value-change emails, one
// per event. Each goes to the person who must act or who is waiting for the
// result, and names the project, client, values and link they need. Subjects
// use the same placeholders as the body.
var projectWorkflowEmails = []projectWorkflowEmail{
	{"pavalsubmitted0", "job_value_submitted", "Asks the selected approver to review a project value change",
		"Approval needed: project {{.JobNumber}} value change",
		`Hello {{.RecipientName}},

{{.UserName}} has asked you to approve a new project value.{{if .Revised}} This replaces a version of this change sent to you earlier.{{end}}

Project: {{.JobNumber}}
Client: {{.ClientName}}
Current value: {{.CurrentValue}}
Proposed value: {{.ProposedValue}}

Description:
{{.Description}}

Review the change and its authorization PDF, then approve or reject it here:

{{.ActionURL}}`},
	{"pavalapproved00", "job_value_approved", "Tells the uploader of a project value change that it was approved",
		"Approved: project {{.JobNumber}} value change",
		`Hello {{.RecipientName}},

{{.UserName}} approved the new value for project {{.JobNumber}} ({{.ClientName}}).

Previous value: {{.PreviousValue}}
New value: {{.ProposedValue}}

The new value is now in effect. View the project here:

{{.ActionURL}}`},
	{"pavalrejected00", "job_value_rejected", "Tells the uploader of a project value change that it was rejected and why",
		"Rejected: project {{.JobNumber}} value change",
		`Hello {{.RecipientName}},

{{.UserName}} rejected the proposed value of {{.ProposedValue}} for project {{.JobNumber}} ({{.ClientName}}) for the following reason:

{{.RejectionReason}}

The project value remains {{.CurrentValue}}. You can correct the change and send it for approval again here:

{{.ActionURL}}`},
	{"pareqsubmitted0", "job_request_submitted", "Asks the selected approver to review a project request",
		"Approval needed: project request for {{.ClientName}} — {{.Title}}",
		`Hello {{.RecipientName}},

{{.UserName}} has asked you to approve a project request for {{.ClientName}} with a project value of {{.ProjectValue}}.{{if .Revised}} This replaces a version of this request sent to you earlier.{{end}}

Project / scope of work:
{{.Description}}

Review the request and its authorization PDF, then approve or reject it here:

{{.ActionURL}}`},
	{"pareqapproved00", "job_request_approved", "Tells the requester that a project request was approved",
		"Approved: project request for {{.ClientName}} — {{.Title}}",
		`Hello {{.RecipientName}},

{{.UserName}} approved your project request for {{.ClientName}} with a project value of {{.ProjectValue}}.

Project / scope of work:
{{.Description}}

The project will now be created and given a number. You will receive another email with the project number when that happens. View the request here:

{{.ActionURL}}`},
	{"pareqrejected00", "job_request_rejected", "Tells the requester that a project request was rejected and why",
		"Rejected: project request for {{.ClientName}} — {{.Title}}",
		`Hello {{.RecipientName}},

{{.UserName}} {{if .Withdrawn}}withdrew approval of{{else}}rejected{{end}} your project request for {{.ClientName}} for the following reason:

{{.RejectionReason}}

Project / scope of work:
{{.Description}}

{{if .Withdrawn}}The project will not be created. {{end}}You can correct the request and submit it again here:

{{.ActionURL}}`},
	{"pareqcreated000", "job_request_created", "Gives the requester and project manager the number of a project created from a request",
		"Project {{.JobNumber}} created for {{.ClientName}}",
		`Hello {{.RecipientName}},

{{.UserName}} created project {{.JobNumber}} for {{.ClientName}} from {{if .ForRequester}}your project request{{else}}a project request by {{.RequesterName}}. You are its project manager{{end}}.

Project / scope of work:
{{.Description}}

{{if .Active}}Time, expenses, and purchase orders can now be recorded against {{.JobNumber}}.{{else}}The project is Closed, so time, expenses, and purchase orders cannot be recorded against it.{{end}} View the project here:

{{.ActionURL}}`},
}

// The request workflow migration added one template for all request events and
// one for all value-change events. Its subjects and wording were placeholders.
var replacedProjectWorkflowEmails = []projectWorkflowEmail{
	{"panotification1", "job_request_workflow", "Job request and value approval notices",
		"Project authorization: {{.Action}}", "Request {{.RequestID}}: {{.Description}}\n{{.Action}}\n\n{{.ActionURL}}"},
	{"pavaluenotice01", "job_value_workflow", "Project value approval notices",
		"Project {{.JobNumber}}: {{.Action}}", "{{.Description}}\n{{.Action}}\n\n{{.ActionURL}}"},
}

// Replace the request workflow's two notice templates with one email per event,
// and switch the new emails on.
func init() {
	m.Register(func(app core.App) error {
		return replaceProjectWorkflowEmails(app, replacedProjectWorkflowEmails, projectWorkflowEmails)
	}, func(app core.App) error {
		return replaceProjectWorkflowEmails(app, projectWorkflowEmails, replacedProjectWorkflowEmails)
	})
}

func replaceProjectWorkflowEmails(app core.App, remove, add []projectWorkflowEmail) error {
	config, err := app.FindFirstRecordByData("app_config", "key", "notifications")
	if err != nil {
		return err
	}
	flags := map[string]any{}
	if err = json.Unmarshal([]byte(config.GetString("value")), &flags); err != nil {
		return err
	}
	for _, email := range remove {
		delete(flags, email.code)
		template, err := app.FindFirstRecordByData("notification_templates", "code", email.code)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		// Sent notifications keep their template. Without its flag a kept
		// template is never used again.
		used, err := app.CountRecords("notifications", dbx.HashExp{"template": template.Id})
		if err != nil {
			return err
		}
		if used > 0 {
			continue
		}
		if err = app.Delete(template); err != nil {
			return err
		}
	}
	templates, err := app.FindCollectionByNameOrId("notification_templates")
	if err != nil {
		return err
	}
	for _, email := range add {
		flags[email.code] = true
		// A template kept above because notifications used it is updated in place.
		template, err := app.FindFirstRecordByData("notification_templates", "code", email.code)
		if errors.Is(err, sql.ErrNoRows) {
			template = core.NewRecord(templates)
			template.Id = email.id
		} else if err != nil {
			return err
		}
		template.Set("code", email.code)
		template.Set("description", email.description)
		template.Set("subject", email.subject)
		template.Set("text_email", email.text)
		if err = app.Save(template); err != nil {
			return err
		}
	}
	config.Set("value", flags)
	return app.Save(config)
}
