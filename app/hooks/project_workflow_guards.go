package hooks

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"reflect"
	"strings"
	"tybalt/errs"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type ProjectWorkflowMutation string

const (
	ProjectWorkflowMutationRequest      ProjectWorkflowMutation = "request"
	ProjectWorkflowMutationCreateJob    ProjectWorkflowMutation = "create_job"
	ProjectWorkflowMutationApproveValue ProjectWorkflowMutation = "approve_value"
	ProjectWorkflowMutationRestoreValue ProjectWorkflowMutation = "restore_value"
	ProjectWorkflowMutationMerge        ProjectWorkflowMutation = "merge"
	ProjectWorkflowMutationValue        ProjectWorkflowMutation = "value"
	ProjectWorkflowMutationDelete       ProjectWorkflowMutation = "delete"
	// ProjectWorkflowMutationChangeClient moves a project and its subjobs to
	// another client, with a recorded reason.
	ProjectWorkflowMutationChangeClient ProjectWorkflowMutation = "change_client"
)

type projectWorkflowContextKey struct{}

func WithProjectWorkflowMutation(ctx context.Context, operation ProjectWorkflowMutation) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, projectWorkflowContextKey{}, operation)
}
func ProjectWorkflowMutationFromContext(ctx context.Context) ProjectWorkflowMutation {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(projectWorkflowContextKey{}).(ProjectWorkflowMutation)
	return v
}

const projectWorkflowRecordOperation = "@projectWorkflowOperation"

// These private types cannot be supplied through a JSON request body.
type projectWorkflowOperationState struct{ operation ProjectWorkflowMutation }
type projectWorkflowValidatedState struct{ fingerprint string }

func projectWorkflowFingerprint(record *core.Record) string {
	data := map[string]any{}
	for _, field := range record.Collection().Fields {
		name := field.GetName()
		if name != "id" && name != "created" && name != "updated" {
			data[name] = record.Get(name)
		}
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func projectWorkflowOperation(record *core.Record) ProjectWorkflowMutation {
	state, ok := record.Get(projectWorkflowRecordOperation).(projectWorkflowOperationState)
	if !ok {
		return ""
	}
	return state.operation
}

// ProcessJobWorkflow applies normal validation within a controlled transition.
func ProcessJobWorkflow(app core.App, job, actor *core.Record, operation ProjectWorkflowMutation) error {
	job.Set(projectWorkflowRecordOperation, projectWorkflowOperationState{operation: operation})
	defer job.Set(projectWorkflowRecordOperation, "")
	return ProcessJobCoreStrict(app, job, actor)
}
func workflowError(message string) error {
	return &errs.HookError{Status: http.StatusBadRequest, Message: message}
}
func activeClaim(app core.App, actor *core.Record, claims ...string) error {
	if actor == nil {
		return &errs.HookError{Status: http.StatusForbidden, Message: "authentication is required"}
	}
	active, err := utilities.IsUserActive(app, actor.Id)
	if err != nil {
		return err
	}
	has := false
	for _, claim := range claims {
		allowed, err := utilities.HasClaim(app, actor, claim)
		if err != nil {
			return err
		}
		has = has || allowed
	}
	if !active || !has {
		return &errs.HookError{Status: http.StatusForbidden, Message: "an active user with the " + strings.Join(claims, " or ") + " claim is required"}
	}
	return nil
}
func validateProjectJobWrite(app core.App, job, actor *core.Record) error {
	if isProposalRecord(job) {
		return nil
	}
	operation := projectWorkflowOperation(job)
	if operation != ProjectWorkflowMutationApproveValue && operation != ProjectWorkflowMutationRestoreValue {
		if err := activeClaim(app, actor, "job"); err != nil {
			return err
		}
	}
	if job.GetFloat("project_value") != math.Trunc(job.GetFloat("project_value")) {
		return workflowError("project value must use whole dollars")
	}
	if job.IsNew() {
		enabled, err := utilities.IsJobRequestApprovalRequired(app)
		if err != nil {
			return err
		}
		if enabled && operation != ProjectWorkflowMutationCreateJob {
			return workflowError("create projects through an approved job request")
		}
	}
	if !job.IsNew() {
		if job.GetString("client") != job.Original().GetString("client") && operation != ProjectWorkflowMutationMerge {
			return workflowError("project client can change only through a client merge")
		}
		if job.GetFloat("project_value") != job.Original().GetFloat("project_value") && operation != ProjectWorkflowMutationApproveValue && operation != ProjectWorkflowMutationRestoreValue {
			return workflowError("project value changes require approval")
		}
	}
	// Required billing details and client relations are checked in validateJob.
	return nil
}

func AddProjectWorkflowHooks(app core.App) {
	app.OnRecordDelete("users").BindFunc(func(e *core.RecordEvent) error {
		for _, table := range []string{"job_requests", "job_attachments", "jobs"} {
			fields := map[string][]string{"job_requests": {"creator", "approver", "rejector"}, "job_attachments": {"uploader", "approver", "rejector"}, "jobs": {"request_creator", "created_by"}}[table]
			var conditions []dbx.Expression
			for _, field := range fields {
				conditions = append(conditions, dbx.HashExp{field: e.Record.Id})
			}
			count, err := e.App.CountRecords(table, dbx.Or(conditions...))
			if err != nil {
				return err
			}
			if count > 0 {
				return workflowError("user identity is retained in project authorization records")
			}
		}
		return e.Next()
	})
	// Generic and model writes cannot change workflow-owned state.
	for _, collection := range []string{"job_requests", "job_request_allocations", "job_attachments"} {
		guard := func(e *core.RecordEvent) error {
			op := ProjectWorkflowMutationFromContext(e.Context)
			if op == "" {
				return workflowError("use the project workflow actions")
			}
			if e.Record.Collection().Name == "job_attachments" {
				r := e.Record
				if (r.GetString("request") == "") == (r.GetString("job") == "") {
					return workflowError("an attachment must belong to one request or job")
				}
				if !r.IsNew() && r.Original().GetString("job") != "" && r.Original().GetString("status") == "Approved" {
					return workflowError("approved project evidence cannot be edited")
				}
				for _, f := range []string{"project_value", "previous_project_value", "approval_max_value"} {
					v := r.GetFloat(f)
					if math.Trunc(v) != v {
						return workflowError(f + " must use whole dollars")
					}
				}
			}
			return e.Next()
		}
		app.OnRecordCreate(collection).BindFunc(guard)
		app.OnRecordUpdate(collection).BindFunc(guard)
		app.OnRecordDelete(collection).BindFunc(func(e *core.RecordEvent) error {
			if ProjectWorkflowMutationFromContext(e.Context) == "" {
				return workflowError("use the project workflow delete action")
			}
			if e.Record.Collection().Name == "job_attachments" && e.Record.GetString("job") != "" && e.Record.GetString("status") == "Approved" {
				if e.Record.GetBool("initial_approval") {
					return workflowError("initial project approval cannot be deleted")
				}
				// Enforce the retained approval order for model deletes too. A larger
				// sequence means a later approved value still depends on this history.
				count, err := e.App.CountRecords("job_attachments", dbx.And(dbx.HashExp{"job": e.Record.GetString("job")}, dbx.Or(dbx.Not(dbx.HashExp{"status": "Approved"}), dbx.NewExp("approval_sequence > {:sequence}", dbx.Params{"sequence": e.Record.GetInt("approval_sequence")}))))
				if err != nil {
					return err
				}
				if count > 0 {
					return workflowError("delete pending or later project changes first")
				}
			}
			return e.Next()
		})
	}
	jobGuard := func(e *core.RecordEvent) error {
		r := e.Record
		op := ProjectWorkflowMutationFromContext(e.Context)
		// The job type follows the stored number. A switch between project and
		// proposal numbers would remove the project guards and clear the value.
		if !r.IsNew() && typeFromNumber(r.GetString("number")) != typeFromNumber(r.Original().GetString("number")) {
			return workflowError("a job number cannot switch between project and proposal")
		}
		if isProposalRecord(r) {
			return e.Next()
		}
		state, validated := r.Get("@projectWorkflowValidated").(projectWorkflowValidatedState)
		validated = validated && state.fingerprint != "" && state.fingerprint == projectWorkflowFingerprint(r)
		if op == "" && !validated {
			return workflowError("project writes require the validated job action")
		}
		defer r.Set("@projectWorkflowValidated", false)

		if r.GetFloat("project_value") != math.Trunc(r.GetFloat("project_value")) {
			return workflowError("project value must use whole dollars")
		}
		if op == ProjectWorkflowMutationRestoreValue {
			if err := validateHistoricalRestoration(e.App, r); err != nil {
				return err
			}
		}

		if r.IsNew() {
			enabled, err := utilities.IsJobRequestApprovalRequired(e.App)
			if err != nil {
				return err
			}
			if enabled && op != ProjectWorkflowMutationCreateJob {
				return workflowError("create projects through an approved job request")
			}
		}
		if !r.IsNew() {
			// The change client action validates only the client relations it
			// sets, so validateProjectJobWrite does not accept it.
			if r.GetString("client") != r.Original().GetString("client") && op != ProjectWorkflowMutationMerge && op != ProjectWorkflowMutationChangeClient {
				return workflowError("project client is immutable")
			}
			if r.GetFloat("project_value") != r.Original().GetFloat("project_value") && op != ProjectWorkflowMutationApproveValue && op != ProjectWorkflowMutationRestoreValue {
				return workflowError("project value changes require approval")
			}
		}
		for _, f := range []string{"request_creator", "request_created", "request_submitted", "created_by", "creation_completed"} {
			if !reflect.DeepEqual(r.Get(f), r.Original().Get(f)) && op != ProjectWorkflowMutationCreateJob {
				return workflowError("project creation history is protected")
			}
		}
		return e.Next()
	}
	app.OnRecordCreate("jobs").BindFunc(jobGuard)
	app.OnRecordUpdate("jobs").BindFunc(jobGuard)
	app.OnRecordDelete("jobs").BindFunc(func(e *core.RecordEvent) error {
		return e.App.RunInTransaction(func(tx core.App) error {
			count, err := tx.CountRecords("job_attachments", dbx.HashExp{"job": e.Record.Id})
			if err != nil {
				return err
			}
			if count > 0 {
				return workflowError("delete permitted project attachments before deleting this job")
			}
			e.App = tx
			return e.Next()
		})
	})
	propsGuard := func(e *core.RecordEvent) error {
		r := e.Record
		v := r.GetFloat("max_value")
		if v <= 0 || v > utilities.MaxProjectAuthorizationValue || math.IsNaN(v) || math.IsInf(v, 0) || math.Trunc(v) != v {
			return workflowError("approval limit must be positive whole dollars no greater than 9007199254740991")
		}
		uc, err := e.App.FindRecordById("user_claims", r.GetString("user_claim"))
		if err != nil {
			return err
		}
		claim, err := e.App.FindRecordById("claims", uc.GetString("cid"))
		if err != nil {
			return err
		}
		if claim.GetString("name") != "project_authorization" {
			return workflowError("properties require a project_authorization assignment")
		}
		return e.Next()
	}
	app.OnRecordCreate("project_authorization_props").BindFunc(propsGuard)
	app.OnRecordUpdate("project_authorization_props").BindFunc(propsGuard)
}

// ValidateProjectRequestData checks the future job without assigning its number.
func ValidateProjectRequestData(app core.App, request *core.Record) error {
	collection, err := app.FindCollectionByNameOrId("jobs")
	if err != nil {
		return err
	}
	job := core.NewRecord(collection)
	for _, field := range []string{"description", "client", "job_owner", "contact", "manager", "alternate_manager", "branch", "rate_sheet", "parent", "proposal", "invoicing_information", "location", "client_po", "client_reference_number", "project_award_date", "project_completion_date", "project_value", "time_and_materials"} {
		job.Set(field, request.Get(field))
	}
	job.Set("status", request.GetString("project_status"))
	job.Set("authorizing_document", "PA")
	if request.GetString("fn_agreement_answer") != "Yes" && request.GetString("fn_agreement_answer") != "No" {
		return workflowError("First Nations agreement answer is required")
	}
	job.Set("fn_agreement", request.GetString("fn_agreement_answer") == "Yes")
	if err := validateRateSheetIsActive(app, job, nil); err != nil {
		return err
	}
	branch, err := app.FindRecordById("branches", job.GetString("branch"))
	if err != nil {
		return workflowError("branch is required")
	}
	if branch.Collection().Fields.GetByName("active") != nil && !branch.GetBool("active") {
		return workflowError("branch must be active")
	}
	_, err = validateJob(app, job, true)
	return err
}

// validateHistoricalRestoration permits only the latest stored historical value.
func validateHistoricalRestoration(app core.App, job *core.Record) error {
	// Sequence, not timestamp or record ID, identifies the approval whose saved
	// previous value can be restored. Never restore through an earlier approval.
	rows, err := app.FindRecordsByFilter("job_attachments", "job={:job} && status='Approved'", "-approval_sequence", 1, 0, dbx.Params{"job": job.Id})
	if err != nil {
		return err
	}
	if len(rows) != 1 || rows[0].GetBool("initial_approval") || rows[0].GetFloat("previous_project_value") != job.GetFloat("project_value") || rows[0].GetFloat("project_value") != job.Original().GetFloat("project_value") {
		return workflowError("restore only the latest approved change's stored previous value")
	}
	pending, err := app.CountRecords("job_attachments", dbx.And(dbx.HashExp{"job": job.Id}, dbx.Not(dbx.HashExp{"status": "Approved"})))
	if err != nil {
		return err
	}
	if pending > 0 {
		return workflowError("delete pending project value changes before restoration")
	}
	return nil
}
