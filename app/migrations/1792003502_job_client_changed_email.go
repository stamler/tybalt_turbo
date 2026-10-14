package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// jobClientChangedEmail tells the managers of a project and its subjobs, and the
// people who approved their values, that a holder of job moved the project to
// another client, and why.
var jobClientChangedEmail = []projectWorkflowEmail{
	{"jobclientchg000", "job_client_changed", "Tells project managers and value approvers that a project moved to another client",
		"Client changed: project {{.JobNumber}}",
		`Hello {{.RecipientName}},

{{.UserName}} changed the client of project {{.JobNumber}} from {{.OldClientName}} to {{.NewClientName}}.{{if .SubjobNumbers}} Its subjobs moved with it: {{.SubjobNumbers}}.{{end}}

Reason:
{{.Reason}}

Project / scope of work:
{{.Description}}

{{if .ForApprover}}You approved a value for this project when its client was {{.OldClientName}}.{{else}}You manage this project or one of its subjobs.{{end}} The project now uses the contact and invoicing profile chosen for {{.NewClientName}}. View the project here:

{{.ActionURL}}`},
}

// Add the client change email and switch it on.
func init() {
	m.Register(func(app core.App) error {
		return replaceProjectWorkflowEmails(app, nil, jobClientChangedEmail)
	}, func(app core.App) error {
		return replaceProjectWorkflowEmails(app, jobClientChangedEmail, nil)
	})
}
