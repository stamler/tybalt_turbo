package routes

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"unicode/utf8"

	"tybalt/errs"
	"tybalt/hooks"
	"tybalt/notifications"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// The reason for moving a project to another client must explain the change.
// The client notes that record it also name both clients and every subjob, and
// must fit client_notes.note.
const (
	jobClientChangeReasonMinLength = 20
	jobClientChangeReasonMaxLength = 500
	clientNoteMaxLength            = 1000
)

type jobChangeClientRequest struct {
	Client               string `json:"client"`
	Contact              string `json:"contact"`
	InvoicingInformation string `json:"invoicing_information"`
	Reason               string `json:"reason"`
	// Subjobs lists the subjobs the user confirmed will move with the project.
	Subjobs []string `json:"subjobs"`
}

func jobChangeClientError(status int, field, code, message string) *errs.HookError {
	err := &errs.HookError{Status: status, Message: message}
	if field != "" {
		err.Data = map[string]errs.CodeError{field: {Code: code, Message: message}}
	}
	return err
}

// createChangeJobClientHandler moves a project, and its subjobs, to another
// client when the client was chosen in error or the work now belongs to a
// different client. A project's client is otherwise fixed. Active holders of
// job may do it, and must give a reason. In one transaction it:
//
//   - sets the new client, contact, and invoicing profile on the project and
//     its subjobs, since a contact and profile must belong to the job's client;
//   - moves the client notes about these jobs to the new client;
//   - records the reason as a note on the project under the new client, and a
//     note under the old client saying where the project went;
//   - queues an email to the managers of the moved jobs and the people who
//     approved their values, so a change made by one person is seen by others.
//
// Approved requests and value changes keep the client they were approved for.
func createChangeJobClientHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}
		jobID := e.Request.PathValue("id")
		var req jobChangeClientRequest
		if err := e.BindBody(&req); err != nil {
			return e.Error(http.StatusBadRequest, "invalid request body", err)
		}
		req.Reason = strings.TrimSpace(req.Reason)

		var notices []string
		err := app.RunInTransaction(func(tx core.App) error {
			var err error
			notices, err = changeJobClient(tx, e, jobID, req)
			return err
		})
		if err != nil {
			var hookErr *errs.HookError
			if errors.As(err, &hookErr) {
				status := hookErr.Status
				if status == 0 {
					status = http.StatusBadRequest
				}
				return e.JSON(status, hookErr)
			}
			return e.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
		}
		sendWorkflowNotices(app, notices)
		return e.JSON(http.StatusOK, map[string]string{"client": req.Client})
	}
}

func changeJobClient(tx core.App, e *core.RequestEvent, jobID string, req jobChangeClientRequest) ([]string, error) {
	active, err := utilities.IsUserActive(tx, e.Auth.Id)
	if err != nil {
		return nil, err
	}
	hasJob, err := utilities.HasClaim(tx, e.Auth, "job")
	if err != nil {
		return nil, err
	}
	if !active || !hasJob {
		return nil, jobChangeClientError(http.StatusForbidden, "", "", "An active user with the job claim is required to change a project's client.")
	}
	editing, err := utilities.IsJobsEditingEnabled(tx)
	if err != nil {
		return nil, err
	}
	if !editing {
		return nil, utilities.ErrJobsEditingDisabled
	}

	job, err := tx.FindRecordById("jobs", jobID)
	if err != nil {
		return nil, jobChangeClientError(http.StatusNotFound, "", "", "Project not found.")
	}
	if strings.HasPrefix(job.GetString("number"), "P") {
		return nil, jobChangeClientError(http.StatusBadRequest, "", "", "A proposal's client can be changed in the job editor.")
	}
	if job.GetString("parent") != "" {
		return nil, jobChangeClientError(http.StatusBadRequest, "", "", "A subjob uses its parent project's client. Change the parent project's client instead.")
	}

	// Check the reason before the selections, so the form reports every
	// missing field the user can see at once.
	fieldErrors := map[string]errs.CodeError{}
	if req.Client == "" {
		fieldErrors["client"] = errs.CodeError{Code: "required", Message: "Select the new client."}
	}
	if req.Contact == "" {
		fieldErrors["contact"] = errs.CodeError{Code: "required", Message: "Select a contact for the new client."}
	}
	if req.InvoicingInformation == "" {
		fieldErrors["invoicing_information"] = errs.CodeError{Code: "required", Message: "Select an invoicing profile for the new client."}
	}
	if n := utf8.RuneCountInString(req.Reason); n < jobClientChangeReasonMinLength {
		fieldErrors["reason"] = errs.CodeError{Code: "too_short", Message: fmt.Sprintf("Explain why the client is changing, in at least %d characters.", jobClientChangeReasonMinLength)}
	} else if n > jobClientChangeReasonMaxLength {
		fieldErrors["reason"] = errs.CodeError{Code: "too_long", Message: fmt.Sprintf("Keep the reason to %d characters or fewer.", jobClientChangeReasonMaxLength)}
	}
	if len(fieldErrors) > 0 {
		return nil, &errs.HookError{Status: http.StatusBadRequest, Message: "Complete the highlighted fields.", Data: fieldErrors}
	}

	oldClientID := job.GetString("client")
	if req.Client == oldClientID {
		return nil, jobChangeClientError(http.StatusBadRequest, "client", "unchanged", "Select a different client.")
	}
	newClient, err := tx.FindRecordById("clients", req.Client)
	if err != nil {
		return nil, jobChangeClientError(http.StatusBadRequest, "client", "invalid_reference", "The selected client was not found.")
	}

	children, err := tx.FindRecordsByFilter("jobs", "parent = {:parent}", "number", 0, 0, dbx.Params{"parent": job.Id})
	if err != nil {
		return nil, err
	}
	if err := checkSubjobsConfirmed(children, req.Subjobs); err != nil {
		return nil, err
	}
	jobs := append([]*core.Record{job}, children...)
	ids := make([]any, len(jobs))
	for i, j := range jobs {
		ids[i] = j.Id
	}

	if err := checkJobClientChangeBlockers(tx, job, ids); err != nil {
		return nil, err
	}
	// Notes are found before the jobs move, while they still match the old client.
	notes, err := tx.FindAllRecords("client_notes", dbx.In("job", ids...), dbx.HashExp{"client": oldClientID})
	if err != nil {
		return nil, err
	}
	if err := checkPendingClientMerges(tx, req, ids, notes); err != nil {
		return nil, err
	}

	oldLabel := noticeClientLabel(tx, oldClientID)
	if oldLabel == "" {
		oldLabel = "no client"
	}
	newLabel := utilities.ClientLabel(newClient)
	subjobs := make([]string, len(children))
	for i, child := range children {
		subjobs[i] = child.GetString("number")
	}
	moved := ""
	if len(subjobs) > 0 {
		moved = " Subjobs moved with it: " + strings.Join(subjobs, ", ") + "."
	}
	changeNote := fmt.Sprintf("Client changed from %s to %s.%s Reason: %s", oldLabel, newLabel, moved, req.Reason)
	departureNote := fmt.Sprintf("Project %s moved to %s.%s Reason: %s", job.GetString("number"), newLabel, moved, req.Reason)
	if over := max(utf8.RuneCountInString(changeNote), utf8.RuneCountInString(departureNote)) - clientNoteMaxLength; over > 0 {
		return nil, jobChangeClientError(http.StatusBadRequest, "reason", "too_long",
			fmt.Sprintf("With the client names and subjobs, the notes recording this change would be too long. Shorten the reason by %d characters.", over))
	}

	ctx := hooks.WithProjectWorkflowMutation(e.Request.Context(), hooks.ProjectWorkflowMutationChangeClient)

	// Every job takes the new selections. Only these relations change, so only
	// they are validated, and the jobs are saved without checking their other
	// fields: a legacy project or an older Closed subjob may not meet today's
	// rules for fields this action does not touch, such as a zero value or a
	// missing branch. The project workflow guard still runs.
	for _, j := range jobs {
		j.Set("client", req.Client)
		j.Set("contact", req.Contact)
		j.Set("invoicing_information", req.InvoicingInformation)
		if err := utilities.ValidateClientReferences(tx, j); err != nil {
			return nil, err
		}
		// Moved jobs are written back to the legacy system.
		utilities.MarkImportedFalseIfChanged(j)
		if err := tx.SaveNoValidateWithContext(ctx, j); err != nil {
			return nil, fmt.Errorf("moving job %s: %w", j.GetString("number"), err)
		}
	}

	for _, note := range notes {
		note.Set("client", req.Client)
		if err := tx.SaveWithContext(ctx, note); err != nil {
			return nil, err
		}
	}

	if err := createJobAuditNote(tx, job, e.Auth.Id, changeNote, ""); err != nil {
		return nil, err
	}
	// The old client keeps no job to attach its note to.
	if oldClientID != "" {
		if err := createClientNote(tx, oldClientID, "", e.Auth.Id, departureNote, ""); err != nil {
			return nil, err
		}
	}

	return queueClientChangeNotice(tx, job, jobs, ids, e.Auth.Id, oldLabel, newLabel, subjobs, req.Reason)
}

// checkSubjobsConfirmed refuses the change unless the user confirmed exactly
// the subjobs that will move. A subjob added since the dialog opened, or a
// confirmation that was never given, would move a job the user did not see.
func checkSubjobsConfirmed(children []*core.Record, confirmed []string) error {
	want := map[string]bool{}
	for _, child := range children {
		want[child.Id] = true
	}
	got := map[string]bool{}
	for _, id := range confirmed {
		got[id] = true
	}
	if !maps.Equal(want, got) {
		return jobChangeClientError(http.StatusConflict, "subjobs", "unconfirmed", "The subjobs confirmed do not match this project's subjobs, which may have changed. Review the current list and confirm again.")
	}
	return nil
}

// checkJobClientChangeBlockers refuses the change while work for the old
// client is still in progress on these jobs. A pending or rejected value change
// would be reviewed or corrected without its approver seeing the new client,
// and an open request for a subjob must use its parent's client.
func checkJobClientChangeBlockers(tx core.App, job *core.Record, ids []any) error {
	open, err := tx.CountRecords("job_attachments", dbx.In("job", ids...), dbx.Not(dbx.HashExp{"status": "Approved"}))
	if err != nil {
		return err
	}
	if open > 0 {
		return jobChangeClientError(http.StatusConflict, "", "", "This project or a subjob has a value change that is awaiting approval or was rejected. Have it approved, or ask an admin to delete it, before changing the client.")
	}
	requests, err := tx.CountRecords("job_requests", dbx.HashExp{"parent": job.Id})
	if err != nil {
		return err
	}
	if requests > 0 {
		return jobChangeClientError(http.StatusConflict, "", "", "A subjob request for this project is still open. Create or delete it before changing the client.")
	}
	return nil
}

// checkPendingClientMerges refuses the change while a client or contact merge
// that moved these jobs or notes, or the chosen contact or profile, can still be
// undone. Undo would restore the old relations over the new ones, or move the
// chosen contact or profile away from the new client and then refuse to finish.
func checkPendingClientMerges(tx core.App, req jobChangeClientRequest, ids []any, notes []*core.Record) error {
	tracked := map[string]bool{req.Contact: true, req.InvoicingInformation: true}
	for _, id := range ids {
		tracked[id.(string)] = true
	}
	for _, note := range notes {
		tracked[note.Id] = true
	}
	actions, err := tx.FindRecordsByFilter("absorb_actions", "collection_name = 'clients' || collection_name = 'client_contacts'", "", 0, 0)
	if err != nil {
		return err
	}
	for _, action := range actions {
		var updates map[string]map[string]map[string]string
		if err := action.UnmarshalJSONField("updated_references", &updates); err != nil {
			return err
		}
		for _, fields := range updates {
			for _, records := range fields {
				for id := range records {
					if tracked[id] {
						return jobChangeClientError(http.StatusConflict, "", "", "A client or contact merge that involves this project or the chosen contact or invoicing profile is waiting to be committed or undone. Finish it before changing the client.")
					}
				}
			}
		}
	}
	return nil
}

// queueClientChangeNotice emails the managers of the moved jobs and the people
// who approved their values, other than those who have left.
func queueClientChangeNotice(tx core.App, job *core.Record, jobs []*core.Record, ids []any, actor, oldLabel, newLabel string, subjobs []string, reason string) ([]string, error) {
	candidates := []string{}
	for _, j := range jobs {
		candidates = append(candidates, j.GetString("manager"))
	}
	approved, err := tx.FindAllRecords("job_attachments", dbx.In("job", ids...), dbx.HashExp{"status": "Approved"})
	if err != nil {
		return nil, err
	}
	approvers := map[string]bool{}
	for _, a := range approved {
		uid := a.GetString("approver")
		approvers[uid] = true
		candidates = append(candidates, uid)
	}
	recipients := []string{}
	for _, uid := range candidates {
		active, err := utilities.IsUserActive(tx, uid)
		if err != nil {
			return nil, err
		}
		if active {
			recipients = append(recipients, uid)
		}
	}
	data := map[string]any{
		"JobNumber":     job.GetString("number"),
		"Description":   job.GetString("description"),
		"OldClientName": oldLabel,
		"NewClientName": newLabel,
		"SubjobNumbers": strings.Join(subjobs, ", "),
		"Reason":        reason,
		"ActionURL":     notifications.BuildActionURL(tx, "/jobs/"+job.Id+"/details"),
	}
	return queueWorkflowNotice(tx, "job_client_changed", actor, recipients, func(uid string) map[string]any {
		values := maps.Clone(data)
		values["ForApprover"] = approvers[uid]
		return values
	})
}
