package routes

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"tybalt/errs"
	"tybalt/hooks"
	"tybalt/notifications"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

const JOB_REQUEST_ABANDONMENT_DAYS = utilities.JOB_REQUEST_ABANDONMENT_DAYS
const creationQueueFilter = "status = 'Awaiting creation'"

var requestBusinessFields = []string{"project_status", "approver", "client", "job_owner", "contact", "manager", "alternate_manager", "branch", "rate_sheet", "parent", "proposal", "invoicing_information", "description", "location", "client_po", "client_reference_number", "project_award_date", "project_completion_date", "project_value", "time_and_materials", "fn_agreement_answer"}
var approvalChecklistKeys = []string{"fn_agreement_reviewed", "no_known_conflict", "no_executive_restriction", "no_duplicate_work", "pdf_sufficient", "allocations_complete"}

type requestWrite struct {
	Request     map[string]any        `json:"request"`
	Allocations []JobAllocationUpdate `json:"allocations"`
	Version     int                   `json:"version"`
}
type workflowAction struct {
	Version              int            `json:"version"`
	ReviewToken          string         `json:"review_token"`
	DocumentHash         string         `json:"document_hash"`
	ChecklistVersion     int            `json:"checklist_version"`
	ChecklistAnswers     map[string]any `json:"checklist_answers"`
	RejectionReason      string         `json:"rejection_reason"`
	DuplicateWorkChecked bool           `json:"duplicate_work_checked"`
}

func registerJobRequestRoutes(se *core.ServeEvent, app core.App) {
	r := se.Router.Group("/api/job_requests")
	r.Bind(apis.RequireAuth("users"))
	r.BindFunc(workflowResponseErrors)
	r.GET("", requestListHandler(app))
	r.GET("/options", requestOptionsHandler(app))
	r.POST("", requestSaveHandler(app))
	r.GET("/{id}", requestDetailHandler(app))
	r.PATCH("/{id}", requestSaveHandler(app))
	r.GET("/{id}/print", requestDetailHandler(app))
	r.GET("/{id}/duplicates", requestDuplicatesHandler(app))
	r.POST("/{id}/document", requestDocumentHandler(app, false))
	r.DELETE("/{id}/document", requestDocumentHandler(app, true))
	for _, action := range []string{"submit", "recall", "reject", "approve", "create"} {
		r.POST("/{id}/"+action, requestActionHandler(app, action))
	}
	r.DELETE("/{id}", requestDeleteHandler(app))
	se.Router.GET("/api/project_authorization/approvers", authorizationApproversHandler(app)).Bind(apis.RequireAuth("users"))
	a := se.Router.Group("/api/job_attachments")
	a.Bind(apis.RequireAuth("users"))
	a.BindFunc(workflowResponseErrors)
	a.GET("", attachmentListHandler(app))
	a.GET("/context", attachmentContextHandler(app))
	a.POST("", attachmentSaveHandler(app))
	a.PATCH("/{id}", attachmentSaveHandler(app))
	a.GET("/{id}/document", attachmentDocumentHandler(app))
	a.DELETE("/{id}", attachmentDeleteHandler(app))
	a.POST("/{id}/approve", attachmentActionHandler(app, true))
	a.POST("/{id}/reject", attachmentActionHandler(app, false))
}

func workflowResponseErrors(e *core.RequestEvent) error {
	err := e.Next()
	var h *errs.HookError
	if errors.As(err, &h) {
		return e.JSON(h.Status, h)
	}
	var v validation.Errors
	if errors.As(err, &v) {
		return apis.NewBadRequestError("Check the request fields.", v)
	}
	return err
}

func workflowError(message string) error { return apis.NewBadRequestError(message, nil) }
func workflowContext(ctx context.Context) context.Context {
	return hooks.WithProjectWorkflowMutation(ctx, hooks.ProjectWorkflowMutationRequest)
}
func workflowActive(app core.App, auth *core.Record) error {
	if auth == nil {
		return apis.NewUnauthorizedError("Sign in to continue.", nil)
	}
	active, err := utilities.IsUserActive(app, auth.Id)
	if err != nil {
		return err
	}
	if !active {
		return apis.NewForbiddenError("An active employee is required.", nil)
	}
	return nil
}
func workflowClaim(app core.App, auth *core.Record, claim string) error {
	if err := workflowActive(app, auth); err != nil {
		return err
	}
	ok, err := utilities.HasClaim(app, auth, claim)
	if err != nil {
		return err
	}
	if !ok {
		return apis.NewForbiddenError("The "+claim+" claim is required.", nil)
	}
	return nil
}
func requestEnabled(app core.App) error {
	enabled, err := utilities.IsJobRequestApprovalRequired(app)
	if err != nil {
		return err
	}
	if !enabled {
		return apis.NewForbiddenError("Project requests are disabled.", nil)
	}
	return nil
}
func abandonedRequest(r *core.Record) bool {
	return !r.GetDateTime("last_edited").Time().After(time.Now().UTC().AddDate(0, 0, -JOB_REQUEST_ABANDONMENT_DAYS))
}
func requestCanView(app core.App, r, auth *core.Record) (bool, error) {
	if err := workflowActive(app, auth); err != nil {
		return false, err
	}
	if r.GetString("creator") == auth.Id {
		return true, nil
	}
	for _, claim := range []string{"job", "project_authorization", "admin"} {
		ok, err := utilities.HasClaim(app, auth, claim)
		if err != nil {
			return false, err
		}
		if ok && (claim == "job" || (claim == "project_authorization" && r.GetString("approver") == auth.Id) || (claim == "admin" && abandonedRequest(r))) {
			return true, nil
		}
	}
	return false, nil
}
func loadVisibleRequest(app core.App, id string, auth *core.Record) (*core.Record, error) {
	r, err := app.FindRecordById("job_requests", id)
	if err != nil {
		return nil, apis.NewNotFoundError("Project request not found.", err)
	}
	ok, err := requestCanView(app, r, auth)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apis.NewForbiddenError("You cannot view this request.", nil)
	}
	return r, nil
}
func requestEditable(app core.App, r, auth *core.Record) error {
	if err := requestEnabled(app); err != nil {
		return err
	}
	if err := workflowActive(app, auth); err != nil {
		return err
	}
	if r.GetString("creator") != auth.Id {
		return apis.NewForbiddenError("Only the creator can edit this request.", nil)
	}
	switch r.GetString("status") {
	case "Draft", "Recalled", "Rejected":
		return nil
	}
	return workflowError("Recall this request before editing it.")
}
func requestAttachment(app core.App, id string) (*core.Record, error) {
	r, err := app.FindFirstRecordByData("job_attachments", "request", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return r, err
}
func requestAllocations(app core.App, id string) ([]*core.Record, error) {
	return app.FindRecordsByFilter("job_request_allocations", "request={:id}", "division", 0, 0, dbx.Params{"id": id})
}

// requestVersion rejects actions based on older saved request data. Compare the
// revision, not updated: two writes can share a millisecond timestamp.
func requestVersion(r *core.Record, version int) error {
	if version != r.GetInt("version") {
		return apis.NewApiError(http.StatusConflict, "This request changed. Reload it before continuing.", nil)
	}
	return nil
}
func clearAttachmentApproval(a *core.Record) {
	for _, key := range []string{"approver", "approved", "review_token", "rejector", "rejected", "rejection_reason"} {
		a.Set(key, "")
	}
	for _, key := range []string{"approval_max_value", "request_version", "checklist_version", "approval_sequence"} {
		a.Set(key, 0)
	}
	a.Set("checklist_answers", nil)
	a.Set("status", "Pending")
}
func clearRequestRejection(r *core.Record) {
	for _, key := range []string{"rejector", "rejected", "rejection_reason"} {
		r.Set(key, "")
	}
}
func markRequestEdited(app core.App, ctx context.Context, r *core.Record) error {
	r.Set("version", r.GetInt("version")+1)
	r.Set("last_edited", time.Now().UTC())
	// Editable requests have no live submission. Only submit may issue a token;
	// business edits cannot reuse an earlier submission's approval identity.
	r.Set("submission_token", "")
	if r.GetString("status") == "Rejected" {
		r.Set("status", "Draft")
	}
	clearRequestRejection(r)
	a, err := requestAttachment(app, r.Id)
	if err != nil {
		return err
	}
	if a != nil {
		clearAttachmentApproval(a)
		a.Set("project_value", r.Get("project_value"))
		a.Set("version", a.GetInt("version")+1)
		if err = app.SaveWithContext(workflowContext(ctx), a); err != nil {
			return err
		}
	}
	return app.SaveWithContext(workflowContext(ctx), r)
}
func validWholeAmount(value any, positive bool) bool {
	v, ok := value.(float64)
	return ok && !math.IsNaN(v) && !math.IsInf(v, 0) && v == math.Trunc(v) && v <= 9007199254740991 && ((positive && v > 0) || (!positive && v >= 0))
}
func applyRequestData(r *core.Record, data map[string]any) error {
	allowed := map[string]bool{}
	for _, key := range requestBusinessFields {
		allowed[key] = true
	}
	for key, value := range data {
		if !allowed[key] {
			return workflowError("Field is not editable: " + key)
		}
		if key == "project_value" && !validWholeAmount(value, false) {
			return workflowError("Project value must be a whole CAD dollar amount.")
		}
		if key == "time_and_materials" {
			if _, ok := value.(bool); !ok {
				return workflowError("Select a fee basis.")
			}
			r.Set("time_and_materials_set", true)
		}
		r.Set(key, value)
	}
	return nil
}
func validateRequestDraft(app core.App, r *core.Record) error {
	return validateRequestDraftAuthority(app, r, true)
}
func validateRequestDraftAuthority(app core.App, r *core.Record, checkAuthority bool) error {
	if status := r.GetString("project_status"); status != "Active" && status != "Closed" {
		return workflowError("Select Active or Closed for the project status.")
	}
	if err := utilities.ValidateClientReferences(app, r); err != nil {
		return err
	}
	if id := r.GetString("approver"); id != "" && checkAuthority {
		if _, err := utilities.ProjectAuthorizationLimit(app, id, r.GetFloat("project_value")); err != nil {
			return err
		}
	}
	if id := r.GetString("parent"); id != "" {
		p, err := app.FindRecordById("jobs", id)
		if err != nil {
			return workflowError("Select a valid parent project.")
		}
		if !regexp.MustCompile(`^\d{2}-\d{3,4}$`).MatchString(p.GetString("number")) {
			return workflowError("Only a top-level project can be a parent.")
		}
		if r.GetString("client") != p.GetString("client") {
			return workflowError("The subjob must use the parent project's client.")
		}
	}
	if id := r.GetString("proposal"); id != "" {
		p, err := app.FindRecordById("jobs", id)
		if err != nil {
			return workflowError("Select a valid proposal.")
		}
		if !strings.HasPrefix(p.GetString("number"), "P") || p.GetString("client") != r.GetString("client") {
			return workflowError("Select a proposal for this client.")
		}
		switch p.GetString("status") {
		case "In Progress", "Submitted", "Awarded":
		default:
			return workflowError("Select an In Progress, Submitted, or Awarded proposal.")
		}
	}
	return nil
}
func validateRequestComplete(app core.App, r *core.Record, checkAuthority bool) (*core.Record, error) {
	if err := validateRequestDraftAuthority(app, r, checkAuthority); err != nil {
		return nil, err
	}
	for _, key := range []string{"description", "location", "client", "contact", "manager", "branch", "rate_sheet", "project_award_date", "invoicing_information", "approver", "fn_agreement_answer"} {
		if strings.TrimSpace(r.GetString(key)) == "" {
			return nil, workflowError("Required field: " + key)
		}
	}
	if !r.GetBool("time_and_materials_set") {
		return nil, workflowError("Select a fee basis.")
	}
	if r.GetFloat("project_value") <= 0 {
		return nil, workflowError("Project value must be positive.")
	}
	if id := r.GetString("proposal"); id != "" {
		p, err := app.FindRecordById("jobs", id)
		if err != nil {
			return nil, err
		}
		if p.GetString("status") != "Awarded" {
			return nil, workflowError("Set the linked proposal to Awarded before continuing.")
		}
	}
	if err := utilities.ValidateClientReferences(app, r); err != nil {
		return nil, err
	}
	if err := utilities.ValidateRequestContactEmails(app, r); err != nil {
		return nil, err
	}
	allocs, err := requestAllocations(app, r.Id)
	if err != nil {
		return nil, err
	}
	inputs := []JobAllocationUpdate{}
	for _, a := range allocs {
		inputs = append(inputs, JobAllocationUpdate{a.GetString("division"), a.GetFloat("hours")})
	}
	if err = validateRequestAllocationInputs(app, inputs, true); err != nil {
		return nil, err
	}
	if err := hooks.ValidateProjectRequestData(app, r); err != nil {
		return nil, err
	}
	a, err := requestAttachment(app, r.Id)
	if err != nil {
		return nil, err
	}
	if a == nil || a.GetString("document") == "" || a.GetString("document_hash") == "" {
		return nil, workflowError("Upload an authorization PDF.")
	}
	if err := ensureAuthorizationFile(app, a, nil); err != nil {
		return nil, err
	}
	return a, nil
}
func validateRequestAllocationInputs(app core.App, allocs []JobAllocationUpdate, required bool) error {
	if len(allocs) == 0 && !required {
		return nil
	}
	if err := validateAllocationInputs(allocs); err != nil {
		return err
	}
	for _, a := range allocs {
		if math.IsNaN(a.Hours) || math.IsInf(a.Hours, 0) {
			return workflowError("Invalid allocation hours.")
		}
		d, err := app.FindRecordById("divisions", a.Division)
		if err != nil {
			return workflowError("Select a valid division.")
		}
		if !d.GetBool("active") {
			return workflowError("Select an active division.")
		}
	}
	return nil
}
func requestSaveHandler(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var input requestWrite
		if err := e.BindBody(&input); err != nil {
			return workflowError("Invalid request data.")
		}
		var id string
		err := app.RunInTransaction(func(tx core.App) error {
			if err := requestEnabled(tx); err != nil {
				return err
			}
			var r *core.Record
			var err error
			creating := e.Request.PathValue("id") == ""
			if creating {
				if err = workflowClaim(tx, e.Auth, "time"); err != nil {
					return err
				}
				c, err := tx.FindCollectionByNameOrId("job_requests")
				if err != nil {
					return err
				}
				r = core.NewRecord(c)
				r.Set("creator", e.Auth.Id)
				r.Set("status", "Draft")
				r.Set("project_status", "Active")
				r.Set("version", 1)
				r.Set("last_edited", time.Now().UTC())
			} else {
				r, err = loadVisibleRequest(tx, e.Request.PathValue("id"), e.Auth)
				if err != nil {
					return err
				}
				if err = requestEditable(tx, r, e.Auth); err != nil {
					return err
				}
				if err = requestVersion(r, input.Version); err != nil {
					return err
				}
			}
			before := map[string]any{}
			for _, key := range requestBusinessFields {
				before[key] = r.Get(key)
			}
			before["time_and_materials_set"] = r.GetBool("time_and_materials_set")
			if err = applyRequestData(r, input.Request); err != nil {
				return err
			}
			if err = validateRequestDraft(tx, r); err != nil {
				return err
			}
			old, err := requestAllocations(tx, r.Id)
			if err != nil {
				return err
			}
			existing := []existingJobAllocation{}
			for _, a := range old {
				existing = append(existing, existingJobAllocation{a.GetString("division"), a.GetFloat("hours")})
			}
			allocationChanged := input.Allocations != nil && allocationsDiffer(existing, input.Allocations)
			if input.Allocations != nil {
				if err = validateRequestAllocationInputs(tx, input.Allocations, false); err != nil {
					return err
				}
			}
			changed := allocationChanged
			for key, value := range before {
				if !reflect.DeepEqual(value, r.Get(key)) {
					changed = true
				}
			}
			if !creating && !changed {
				id = r.Id
				return nil
			}
			if creating {
				err = tx.SaveWithContext(workflowContext(e.Request.Context()), r)
			} else {
				err = markRequestEdited(tx, e.Request.Context(), r)
			}
			if err != nil {
				return err
			}
			if allocationChanged {
				for _, a := range old {
					if err = tx.DeleteWithContext(workflowContext(e.Request.Context()), a); err != nil {
						return err
					}
				}
				c, err := tx.FindCollectionByNameOrId("job_request_allocations")
				if err != nil {
					return err
				}
				for _, a := range input.Allocations {
					row := core.NewRecord(c)
					row.Set("request", r.Id)
					row.Set("division", a.Division)
					row.Set("hours", a.Hours)
					if err = tx.SaveWithContext(workflowContext(e.Request.Context()), row); err != nil {
						return err
					}
				}
			}
			id = r.Id
			return nil
		})
		if err != nil {
			return err
		}
		return respondRequestDetail(app, e, id)
	}
}
func validateChecklist(input workflowAction) error {
	if input.ChecklistVersion != 1 {
		return workflowError("Checklist version 1 is required.")
	}
	if len(input.ChecklistAnswers) != len(approvalChecklistKeys) {
		return workflowError("Use only the six checklist answers for version 1.")
	}
	for _, key := range approvalChecklistKeys {
		v, ok := input.ChecklistAnswers[key].(bool)
		if !ok || !v {
			return workflowError("Confirm: " + key)
		}
	}
	return nil
}
func requestActionHandler(app core.App, action string) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var input workflowAction
		if err := e.BindBody(&input); err != nil {
			return workflowError("Invalid action data.")
		}
		var result map[string]any
		err := app.RunInTransaction(func(tx core.App) error {
			if err := requestEnabled(tx); err != nil {
				return err
			}
			r, err := loadVisibleRequest(tx, e.Request.PathValue("id"), e.Auth)
			if err != nil {
				return err
			}
			if err = requestVersion(r, input.Version); err != nil {
				return err
			}
			ctx := workflowContext(e.Request.Context())
			state := r.GetString("status")
			a, err := requestAttachment(tx, r.Id)
			if err != nil {
				return err
			}
			switch action {
			case "submit":
				if err = workflowClaim(tx, e.Auth, "time"); err != nil {
					return err
				}
				if err = requestEditable(tx, r, e.Auth); err != nil {
					return err
				}
				if state == "Rejected" {
					return workflowError("Edit the rejected request before submission.")
				}
				if a, err = validateRequestComplete(tx, r, true); err != nil {
					return err
				}
				clearAttachmentApproval(a)
				a.Set("project_value", r.Get("project_value"))
				r.Set("status", "Awaiting approval")
				r.Set("submitted", time.Now().UTC())
				// Issue a new identity even when the creator resubmits unchanged data
				// and the same PDF. Clearing database approval fields cannot invalidate
				// checklist answers held in an old browser page. requestReviewToken
				// binds that page to this token, so a later submission rejects it.
				// Generate here, after validation, in the same transaction as the
				// status change. A failed save/notification rolls the token back too.
				// Do not change version or last_edited: no business edit occurred.
				r.Set("submission_token", security.RandomStringWithAlphabet(32, "abcdefghijklmnopqrstuvwxyz0123456789"))
			case "recall", "reject":
				if state != "Awaiting approval" && state != "Awaiting creation" {
					return workflowError("This action requires a submitted or approved request.")
				}
				if action == "recall" {
					if r.GetString("creator") != e.Auth.Id {
						return apis.NewForbiddenError("Only the creator can recall this request.", nil)
					}
					r.Set("status", "Recalled")
					clearRequestRejection(r)
				} else {
					if err = workflowClaim(tx, e.Auth, "project_authorization"); err != nil {
						return err
					}
					if r.GetString("approver") != e.Auth.Id {
						return apis.NewForbiddenError("Only the selected approver can reject this request.", nil)
					}
					if strings.TrimSpace(input.RejectionReason) == "" {
						return workflowError("Enter a rejection reason.")
					}
					r.Set("status", "Rejected")
					r.Set("rejected", time.Now().UTC())
					r.Set("rejector", e.Auth.Id)
					r.Set("rejection_reason", strings.TrimSpace(input.RejectionReason))
				}
				// End this submission together with its saved approval/checklist.
				// Resubmission must issue a new token, never revive this one.
				r.Set("submission_token", "")
				if a != nil {
					clearAttachmentApproval(a)
					if action == "reject" {
						a.Set("status", "Rejected")
						for _, key := range []string{"rejector", "rejected", "rejection_reason"} {
							a.Set(key, r.Get(key))
						}
					}
				}
			case "approve":
				if state != "Awaiting approval" || r.GetString("rejected") != "" {
					return workflowError("Only a submitted request can be approved.")
				}
				// A missing token must not fall back to a data-only review digest.
				// The digest below binds the browser's approval to this submission
				// and these fields/PDF; it is not an authorization credential.
				if r.GetString("submission_token") == "" {
					return apis.NewApiError(http.StatusConflict, "This request has no submission token. Recall and submit it again.", nil)
				}
				if r.GetString("approver") != e.Auth.Id {
					return apis.NewForbiddenError("Only the selected approver can approve this request.", nil)
				}
				limit, err := utilities.ProjectAuthorizationLimit(tx, e.Auth.Id, r.GetFloat("project_value"))
				if err != nil {
					return err
				}
				if err = validateChecklist(input); err != nil {
					return err
				}
				a, err = validateRequestComplete(tx, r, true)
				if err != nil {
					return err
				}
				token, err := requestReviewToken(tx, r)
				if err != nil {
					return err
				}
				if input.ReviewToken != token || input.DocumentHash != a.GetString("document_hash") {
					return apis.NewApiError(http.StatusConflict, "The reviewed data changed. Reload and review it again.", nil)
				}
				a.Set("status", "Approved")
				a.Set("approver", e.Auth.Id)
				a.Set("approved", time.Now().UTC())
				a.Set("project_value", r.Get("project_value"))
				a.Set("approval_max_value", limit)
				a.Set("request_version", r.GetInt("version"))
				a.Set("checklist_version", 1)
				a.Set("checklist_answers", input.ChecklistAnswers)
				a.Set("review_token", token)
				r.Set("status", "Awaiting creation")
			case "create":
				if err = workflowClaim(tx, e.Auth, "job"); err != nil {
					return err
				}
				if state != "Awaiting creation" {
					return workflowError("This request is not awaiting creation.")
				}
				// Retain the submission identity through creation. Comparing both
				// the attachment's approved digest and the caller's digest below
				// prevents creation using approval from a different submission.
				if r.GetString("submission_token") == "" {
					return apis.NewApiError(http.StatusConflict, "This request has no submission token. Recall and submit it again.", nil)
				}
				if !input.DuplicateWorkChecked {
					return workflowError("Check existing jobs and pending requests for duplicate work.")
				}
				a, err = validateRequestComplete(tx, r, false)
				if err != nil {
					return err
				}
				token, err := requestReviewToken(tx, r)
				if err != nil {
					return err
				}
				if a.GetString("status") != "Approved" || a.GetInt("request_version") != r.GetInt("version") || a.GetFloat("project_value") != r.GetFloat("project_value") || a.GetString("review_token") != token || input.ReviewToken != token || input.DocumentHash != a.GetString("document_hash") {
					return apis.NewApiError(http.StatusConflict, "The approval is stale. Recall the request for a new approval.", nil)
				}
				if a.GetString("approver") != r.GetString("approver") || a.GetString("approved") == "" || a.GetFloat("approval_max_value") < r.GetFloat("project_value") || a.GetString("rejected") != "" || r.GetString("rejected") != "" {
					return workflowError("The saved approval is incomplete. Recall the request for a new approval.")
				}
				var retainedAnswers map[string]any
				if err := a.UnmarshalJSONField("checklist_answers", &retainedAnswers); err != nil {
					return workflowError("The saved checklist is invalid.")
				}
				if err := validateChecklist(workflowAction{ChecklistVersion: a.GetInt("checklist_version"), ChecklistAnswers: retainedAnswers}); err != nil {
					return err
				}
				id, err := createJobFromRequest(tx, e, r, a)
				if err != nil {
					return err
				}
				result = map[string]any{"id": id, "job_id": id}
				return nil
			}
			if a != nil {
				if err = tx.SaveWithContext(ctx, a); err != nil {
					return err
				}
			}
			if err = tx.SaveWithContext(ctx, r); err != nil {
				return err
			}
			if action == "submit" || action == "approve" || action == "reject" {
				if err = queueRequestNotice(tx, r, action, e.Auth.Id); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		if result != nil {
			return e.JSON(http.StatusOK, result)
		}
		return respondRequestDetail(app, e, e.Request.PathValue("id"))
	}
}
func createJobFromRequest(app core.App, e *core.RequestEvent, r, a *core.Record) (string, error) {
	c, err := app.FindCollectionByNameOrId("jobs")
	if err != nil {
		return "", err
	}
	job := core.NewRecord(c)
	for _, key := range requestBusinessFields {
		if c.Fields.GetByName(key) != nil && key != "approver" {
			job.Set(key, r.Get(key))
		}
	}
	job.Set("status", r.GetString("project_status"))
	job.Set("fn_agreement", r.GetString("fn_agreement_answer") == "Yes")
	job.Set("authorizing_document", "PA")
	if err = hooks.ProcessJobWorkflow(app, job, e.Auth, hooks.ProjectWorkflowMutationCreateJob); err != nil {
		return "", err
	}
	job.Set("request_creator", r.Get("creator"))
	job.Set("request_created", r.Get("created"))
	job.Set("request_submitted", r.Get("submitted"))
	job.Set("created_by", e.Auth.Id)
	job.Set("creation_completed", time.Now().UTC())
	ctx := hooks.WithProjectWorkflowMutation(e.Request.Context(), hooks.ProjectWorkflowMutationCreateJob)
	if err = app.SaveWithContext(ctx, job); err != nil {
		return "", err
	}
	allocs, err := requestAllocations(app, r.Id)
	if err != nil {
		return "", err
	}
	ac, err := app.FindCollectionByNameOrId("job_time_allocations")
	if err != nil {
		return "", err
	}
	for _, alloc := range allocs {
		n := core.NewRecord(ac)
		n.Set("job", job.Id)
		n.Set("division", alloc.Get("division"))
		n.Set("hours", alloc.Get("hours"))
		if err = app.SaveWithContext(ctx, n); err != nil {
			return "", err
		}
		if err = app.DeleteWithContext(ctx, alloc); err != nil {
			return "", err
		}
	}
	a.Set("request", "")
	a.Set("job", job.Id)
	a.Set("initial_approval", true)
	// Transferring the request establishes the first approval in this job's
	// history. Later value approvals extend this order; attachment version is
	// a separate counter for changes to this one record.
	a.Set("approval_sequence", 1)
	if err = app.SaveWithContext(ctx, a); err != nil {
		return "", err
	}
	if err = transferRequestMergeReferences(app, r, job); err != nil {
		return "", err
	}
	if err = app.DeleteWithContext(ctx, r); err != nil {
		return "", err
	}
	return job.Id, nil
}
func queueRequestNotice(app core.App, r *core.Record, action, actor string) error {
	seen := map[string]bool{}
	for _, uid := range []string{r.GetString("approver"), r.GetString("creator")} {
		if uid == "" || seen[uid] {
			continue
		}
		seen[uid] = true
		_, err := notifications.DispatchNotification(app, notifications.DispatchArgs{TemplateCode: "job_request_workflow", RecipientUID: uid, System: true, ActorUID: actor, Mode: notifications.DeliveryDeferred, Data: map[string]any{"RequestID": r.Id, "Description": r.GetString("description"), "Action": action, "ActionURL": notifications.BuildActionURL(app, "/job-requests/"+r.Id)}})
		if err != nil {
			return err
		}
	}
	return nil
}
func requestDeleteHandler(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		err := app.RunInTransaction(func(tx core.App) error {
			if err := workflowClaim(tx, e.Auth, "admin"); err != nil {
				return err
			}
			r, err := tx.FindRecordById("job_requests", e.Request.PathValue("id"))
			if err != nil {
				return err
			}
			if !abandonedRequest(r) {
				return workflowError("Only abandoned requests can be deleted.")
			}
			ctx := workflowContext(e.Request.Context())
			a, err := requestAttachment(tx, r.Id)
			if err != nil {
				return err
			}
			if a != nil {
				if err = tx.DeleteWithContext(ctx, a); err != nil {
					return err
				}
			}
			allocs, err := requestAllocations(tx, r.Id)
			if err != nil {
				return err
			}
			for _, a := range allocs {
				if err = tx.DeleteWithContext(ctx, a); err != nil {
					return err
				}
			}
			return tx.DeleteWithContext(ctx, r)
		})
		if err != nil {
			return err
		}
		return e.NoContent(http.StatusNoContent)
	}
}
