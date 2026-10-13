package routes

import (
	"maps"
	"math"
	"strconv"
	"strings"

	"tybalt/notifications"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Project request and value-change emails. Each event has its own template and
// goes only to the person who must act on it or who is waiting for the result,
// never to the person who performed the action. Workflow handlers queue
// notices inside their transaction, so a failed action leaves none, and send
// them with sendWorkflowNotices once the transaction commits.

// queueWorkflowNotice queues one template for each distinct recipient other
// than the actor. dataFor returns the template data for a recipient.
func queueWorkflowNotice(app core.App, template, actor string, recipients []string, dataFor func(uid string) map[string]any) ([]string, error) {
	ids := []string{}
	seen := map[string]bool{actor: true}
	for _, uid := range recipients {
		if uid == "" || seen[uid] {
			continue
		}
		seen[uid] = true
		id, err := notifications.DispatchNotification(app, notifications.DispatchArgs{TemplateCode: template, RecipientUID: uid, System: true, ActorUID: actor, Mode: notifications.DeliveryDeferred, Data: dataFor(uid)})
		if err != nil {
			return nil, err
		}
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// sendWorkflowNotices sends notices queued by a committed workflow action, as
// purchase order notices are sent after their save. A failed send is logged;
// it never undoes the action.
func sendWorkflowNotices(app core.App, ids []string) {
	for _, id := range ids {
		if err := notifications.SendNotificationByID(app, id); err != nil {
			app.Logger().Error("failed to send project workflow notice", "notification_id", id, "error", err)
		}
	}
}

// noticeWholeCAD formats a whole-dollar amount as the emails show it, such as
// $12,345.
func noticeWholeCAD(value float64) string {
	digits := strconv.FormatInt(int64(math.Round(value)), 10)
	for i := len(digits) - 3; i > 0; i -= 3 {
		digits = digits[:i] + "," + digits[i:]
	}
	return "$" + digits
}

func noticeClientLabel(app core.App, clientID string) string {
	client, err := app.FindRecordById("clients", clientID)
	if err != nil {
		return ""
	}
	return utilities.ClientLabel(client)
}

// noticeTitle shortens a request's scope of work to its first line for email
// subjects, so requests for the same client are told apart.
func noticeTitle(description string) string {
	title := ""
	for _, line := range strings.Split(description, "\n") {
		if title = strings.Join(strings.Fields(line), " "); title != "" {
			break
		}
	}
	if runes := []rune(title); len(runes) > 60 {
		title = strings.TrimSpace(string(runes[:59])) + "…"
	}
	return title
}

// requestNoticeSentBefore reports whether recipient was already sent template
// about the request. A notice whose delivery failed does not count.
func requestNoticeSentBefore(app core.App, recipient, template, requestID string) bool {
	count, err := app.CountRecords("notifications",
		dbx.HashExp{"recipient": recipient},
		dbx.Not(dbx.HashExp{"status": "error"}),
		dbx.NewExp("template IN (SELECT id FROM notification_templates WHERE code = {:code})", dbx.Params{"code": template}),
		dbx.NewExp("json_extract(data, '$.RequestID') = {:request}", dbx.Params{"request": requestID}),
	)
	return err == nil && count > 0
}

// queueValueNotice queues the email for a value-change event: "submitted" to
// the selected approver, "approved" or "rejected" to the person who uploaded
// the change's PDF. revised marks a submission that replaces an earlier version
// with the same approver.
func queueValueNotice(app core.App, attachment *core.Record, event, actor string, revised bool) ([]string, error) {
	job, err := app.FindRecordById("jobs", attachment.GetString("job"))
	if err != nil {
		return nil, err
	}
	data := map[string]any{
		"JobNumber":       job.GetString("number"),
		"Description":     job.GetString("description"),
		"ClientName":      noticeClientLabel(app, job.GetString("client")),
		"CurrentValue":    noticeWholeCAD(job.GetFloat("project_value")),
		"ProposedValue":   noticeWholeCAD(attachment.GetFloat("project_value")),
		"PreviousValue":   noticeWholeCAD(attachment.GetFloat("previous_project_value")),
		"RejectionReason": attachment.GetString("rejection_reason"),
		"Revised":         revised,
		"ActionURL":       notifications.BuildActionURL(app, "/jobs/"+job.Id+"/details"),
	}
	recipient := attachment.GetString("uploader")
	if event == "submitted" {
		recipient = attachment.GetString("assigned_approver")
	}
	return queueWorkflowNotice(app, "job_value_"+event, actor, []string{recipient}, func(string) map[string]any { return data })
}

// queueRequestNotice queues the email for a request event: "submitted" to the
// selected approver, "approved" or "rejected" to the requester, and "created"
// to the requester and the new project's manager. job is the created project.
// withdrawn marks a rejection of an approved request.
func queueRequestNotice(app core.App, r *core.Record, event, actor string, job *core.Record, withdrawn bool) ([]string, error) {
	data := map[string]any{
		"RequestID":       r.Id,
		"ClientName":      noticeClientLabel(app, r.GetString("client")),
		"Title":           noticeTitle(r.GetString("description")),
		"Description":     r.GetString("description"),
		"ProjectValue":    noticeWholeCAD(r.GetFloat("project_value")),
		"RejectionReason": r.GetString("rejection_reason"),
		"Withdrawn":       withdrawn,
		"ActionURL":       notifications.BuildActionURL(app, "/job-requests/"+r.Id),
	}
	recipients := []string{r.GetString("creator")}
	switch event {
	case "submitted":
		recipients = []string{r.GetString("approver")}
		// Recall and rejection keep the request, so an approver can be sent it again.
		data["Revised"] = requestNoticeSentBefore(app, r.GetString("approver"), "job_request_submitted", r.Id)
	case "created":
		recipients = append(recipients, job.GetString("manager"))
		data["JobNumber"] = job.GetString("number")
		data["Active"] = job.GetString("status") == "Active"
		data["ActionURL"] = notifications.BuildActionURL(app, "/jobs/"+job.Id+"/details")
		data["RequesterName"] = ""
		if creator, err := app.FindRecordById("users", r.GetString("creator")); err == nil {
			data["RequesterName"] = workflowPerson(app, creator)["name"]
		}
	}
	return queueWorkflowNotice(app, "job_request_"+event, actor, recipients, func(uid string) map[string]any {
		values := maps.Clone(data)
		values["ForRequester"] = uid == r.GetString("creator")
		return values
	})
}
