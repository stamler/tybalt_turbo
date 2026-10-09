package routes

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"tybalt/hooks"
	"tybalt/notifications"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func readAuthorizationUpload(e *core.RequestEvent, required bool) (*filesystem.File, string, error) {
	e.Request.Body = http.MaxBytesReader(e.Response, e.Request.Body, 21<<20)
	files, err := e.FindUploadedFiles("document")
	if err != nil {
		return nil, "", workflowError("Cannot read the authorization PDF.")
	}
	if len(files) == 0 && !required {
		return nil, "", nil
	}
	if len(files) != 1 {
		return nil, "", workflowError("Upload one authorization PDF.")
	}
	f := files[0]
	reader, err := f.Reader.Open()
	if err != nil {
		return nil, "", err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, (20<<20)+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > 20<<20 {
		return nil, "", workflowError("The PDF must be at most 20 MB.")
	}
	if !strings.HasPrefix(string(data), "%PDF-") {
		return nil, "", workflowError("Upload a valid PDF file.")
	}
	sum := sha256.Sum256(data)
	return f, hex.EncodeToString(sum[:]), nil
}
func checkAuthorizationHash(app core.App, hash, except string) error {
	existing, err := app.FindFirstRecordByFilter("job_attachments", "document_hash={:hash} && id!={:id}", dbx.Params{"hash": hash, "id": except})
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if id := existing.GetString("request"); id != "" {
		r, err := app.FindRecordById("job_requests", id)
		if err != nil {
			return err
		}
		if abandonedRequest(r) {
			return workflowError("This PDF belongs to abandoned job request " + id + ". Give this request ID to an admin to resolve the duplicate.")
		}
	}
	return workflowError("This PDF already belongs to a project request or job approval.")
}
func requestDocumentHandler(app core.App, remove bool) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var file *filesystem.File
		var hash string
		var version int
		var err error
		if remove {
			var action workflowAction
			if err = e.BindBody(&action); err != nil {
				return workflowError("Invalid action data.")
			}
			version = action.Version
		} else {
			file, hash, err = readAuthorizationUpload(e, true)
			if err != nil {
				return err
			}
			version, _ = strconv.Atoi(e.Request.FormValue("version"))
		}
		err = app.RunInTransaction(func(tx core.App) error {
			r, err := loadVisibleRequest(tx, e.Request.PathValue("id"), e.Auth)
			if err != nil {
				return err
			}
			if err = requestEditable(tx, r, e.Auth); err != nil {
				return err
			}
			if err = requestVersion(r, version); err != nil {
				return err
			}
			a, err := requestAttachment(tx, r.Id)
			if err != nil {
				return err
			}
			ctx := workflowContext(e.Request.Context())
			if remove {
				if a == nil {
					return nil
				}
				if err = tx.DeleteWithContext(ctx, a); err != nil {
					return err
				}
			} else {
				if a != nil && a.GetString("document_hash") == hash {
					return ensureAuthorizationFile(tx, a, file)
				}
				if a == nil {
					c, err := tx.FindCollectionByNameOrId("job_attachments")
					if err != nil {
						return err
					}
					a = core.NewRecord(c)
					a.Set("request", r.Id)
				}
				if err = checkAuthorizationHash(tx, hash, a.Id); err != nil {
					return err
				}
				clearAttachmentApproval(a)
				a.Set("document", file)
				a.Set("document_hash", hash)
				a.Set("uploader", e.Auth.Id)
				a.Set("uploaded", time.Now().UTC())
				a.Set("project_value", r.Get("project_value"))
				a.Set("version", a.GetInt("version")+1)
				if err = tx.SaveWithContext(ctx, a); err != nil {
					return err
				}
			}
			return markRequestEdited(tx, e.Request.Context(), r)
		})
		if err != nil {
			return err
		}
		return respondRequestDetail(app, e, e.Request.PathValue("id"))
	}
}
func attachmentCanView(app core.App, a, auth *core.Record) (bool, error) {
	if err := workflowActive(app, auth); err != nil {
		return false, err
	}
	if id := a.GetString("request"); id != "" {
		r, err := app.FindRecordById("job_requests", id)
		if err != nil {
			return false, err
		}
		return requestCanView(app, r, auth)
	}
	for _, claim := range []string{"job", "admin"} {
		ok, err := utilities.HasClaim(app, auth, claim)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	ok, err := utilities.HasClaim(app, auth, "project_authorization")
	if err != nil {
		return false, err
	}
	return ok && (a.GetString("assigned_approver") == auth.Id || a.GetString("approver") == auth.Id), nil
}
func attachmentDocumentHandler(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		a, err := app.FindRecordById("job_attachments", e.Request.PathValue("id"))
		if err != nil {
			return apis.NewNotFoundError("Attachment not found.", err)
		}
		ok, err := attachmentCanView(app, a, e.Auth)
		if err != nil {
			return err
		}
		if !ok {
			return apis.NewForbiddenError("You cannot view this PDF.", nil)
		}
		fs, err := app.NewFilesystem()
		if err != nil {
			return err
		}
		defer fs.Close()
		e.Response.Header().Set("Cache-Control", "private, no-store")
		return fs.Serve(e.Response, e.Request, a.BaseFilesPath()+"/"+a.GetString("document"), a.GetString("document"))
	}
}
func attachmentListHandler(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := workflowActive(app, e.Auth); err != nil {
			return err
		}
		query := e.Request.URL.Query()
		filter := "job!=''"
		params := dbx.Params{}
		if query.Get("queue") == "approval" {
			if err := workflowClaim(app, e.Auth, "project_authorization"); err != nil {
				return err
			}
			filter += " && assigned_approver={:uid} && status='Pending'"
			params["uid"] = e.Auth.Id
		} else {
			if query.Get("job") == "" {
				return workflowError("Select a job.")
			}
			filter += " && job={:job}"
			params["job"] = query.Get("job")
		}
		// Read each change, its job, and the token from one snapshot so a shown
		// row and its review token always describe the same data.
		items := []map[string]any{}
		err := app.RunInTransaction(func(tx core.App) error {
			// Approval sequence orders retained approvals per job. Creation time only
			// breaks ties, including pending/rejected rows whose sequence is zero.
			rows, err := tx.FindRecordsByFilter("job_attachments", filter, "-approval_sequence,-created", 0, 0, params)
			if err != nil {
				return err
			}
			for _, a := range rows {
				ok, err := attachmentCanView(tx, a, e.Auth)
				if err != nil {
					return err
				}
				if !ok {
					continue
				}
				row := recordPublic(a)
				row["document_url"] = "/api/job_attachments/" + a.Id + "/document"
				j, err := tx.FindRecordById("jobs", a.GetString("job"))
				if err != nil {
					return err
				}
				row["job_number"] = j.GetString("number")
				row["job_description"] = j.GetString("description")
				row["effective_project_value"] = j.GetFloat("project_value")
				row["job_record"] = recordPublic(j)
				related, err := requestRelated(tx, j)
				if err != nil {
					return err
				}
				row["related"] = related
				if a.GetString("status") == "Pending" {
					// Identify the job data shown with this change. A rejection records
					// it, so only later changes count as corrections.
					token, err := valueReviewTokenFor(tx, j, related)
					if err != nil {
						return err
					}
					row["current_review_token"] = token
				}
				for _, key := range []string{"uploader", "approver", "assigned_approver", "rejector"} {
					if id := a.GetString(key); id != "" {
						u, err := tx.FindRecordById("users", id)
						if err != nil {
							return err
						}
						row[key+"_name"] = workflowPerson(tx, u)["name"]
					}
				}
				items = append(items, row)
			}
			return nil
		})
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"items": items})
	}
}
func attachmentContextHandler(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var result map[string]any
		err := app.RunInTransaction(func(tx core.App) error {
			if err := workflowClaim(tx, e.Auth, "job"); err != nil {
				return err
			}
			job, err := tx.FindRecordById("jobs", e.Request.URL.Query().Get("job"))
			if err != nil {
				return err
			}
			allocations, err := tx.FindRecordsByFilter("job_time_allocations", "job={:id}", "division", 0, 0, dbx.Params{"id": job.Id})
			if err != nil {
				return err
			}
			token, err := valueReviewToken(tx, job)
			if err != nil {
				return err
			}
			result = map[string]any{"job": recordPublic(job), "allocations": allocations, "review_token": token}
			return nil
		})
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, result)
	}
}

func storedValueAllocations(app core.App, jobID string) ([]*core.Record, []JobAllocationUpdate, error) {
	rows, err := app.FindRecordsByFilter("job_time_allocations", "job={:id}", "division", 0, 0, dbx.Params{"id": jobID})
	if err != nil {
		return nil, nil, err
	}
	inputs := make([]JobAllocationUpdate, 0, len(rows))
	for _, row := range rows {
		inputs = append(inputs, JobAllocationUpdate{Division: row.GetString("division"), Hours: row.GetFloat("hours")})
	}
	return rows, inputs, nil
}

func valueReviewToken(app core.App, job *core.Record) (string, error) {
	related, err := requestRelated(app, job)
	if err != nil {
		return "", err
	}
	return valueReviewTokenFor(app, job, related)
}

// valueReviewTokenFor identifies a job, its related records as already read,
// its approval history, and its allocations. Pass the related records a
// response shows, so the token describes exactly that data.
func valueReviewTokenFor(app core.App, job *core.Record, related map[string]any) (string, error) {
	// Bind review to the current approval history as well as the job data.
	// Attachment version alone cannot detect changes to these related records.
	rows, err := app.FindRecordsByFilter("job_attachments", "job={:id} && status='Approved'", "-approval_sequence", 1, 0, dbx.Params{"id": job.Id})
	if err != nil {
		return "", err
	}
	sequence := 0
	if len(rows) > 0 {
		sequence = rows[0].GetInt("approval_sequence")
	}
	allocations, err := app.FindRecordsByFilter("job_time_allocations", "job={:id}", "division", 0, 0, dbx.Params{"id": job.Id})
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(map[string]any{"job": job.PublicExport(), "sequence": sequence, "related": related, "allocations": allocations})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
func validateValueJob(app core.App, job, actor *core.Record, value float64) error {
	candidate, err := app.FindRecordById("jobs", job.Id)
	if err != nil {
		return err
	}
	candidate.Set("project_value", value)
	return hooks.ProcessJobWorkflow(app, candidate, actor, hooks.ProjectWorkflowMutationApproveValue)
}

func attachmentSaveHandler(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := workflowClaim(app, e.Auth, "job"); err != nil {
			return err
		}
		creating := e.Request.PathValue("id") == ""
		file, hash, err := readAuthorizationUpload(e, creating)
		if err != nil {
			return err
		}
		value, err := strconv.ParseFloat(e.Request.FormValue("project_value"), 64)
		if err != nil || !validWholeAmount(value, true) {
			return workflowError("Enter a positive whole-dollar project value.")
		}
		var jobUpdates map[string]any
		var allocations []JobAllocationUpdate
		if raw := e.Request.FormValue("job_updates"); raw != "" {
			if err := json.Unmarshal([]byte(raw), &jobUpdates); err != nil {
				return workflowError("Invalid job correction data.")
			}
		}
		if raw := e.Request.FormValue("allocations"); raw != "" {
			if err := json.Unmarshal([]byte(raw), &allocations); err != nil {
				return workflowError("Invalid allocation data.")
			}
		}
		assigned := e.Request.FormValue("assigned_approver")
		version, _ := strconv.Atoi(e.Request.FormValue("version"))
		var result *core.Record
		err = app.RunInTransaction(func(tx core.App) error {
			if err := workflowClaim(tx, e.Auth, "job"); err != nil {
				return err
			}
			if _, err := utilities.ProjectAuthorizationLimit(tx, assigned, value); err != nil {
				return err
			}
			var a *core.Record
			var err error
			if creating {
				c, err := tx.FindCollectionByNameOrId("job_attachments")
				if err != nil {
					return err
				}
				a = core.NewRecord(c)
				a.Set("job", e.Request.FormValue("job"))
			} else {
				a, err = tx.FindRecordById("job_attachments", e.Request.PathValue("id"))
				if err != nil {
					return err
				}
				if a.GetString("job") == "" || a.GetString("status") == "Approved" {
					return workflowError("Approved evidence cannot be edited.")
				}
				// Compare within the write transaction so an old form cannot overwrite
				// a later edit. The ID alone still identifies the same changed row.
				if version != a.GetInt("version") {
					return apis.NewApiError(http.StatusConflict, "This value change changed. Reload it.", nil)
				}
			}
			job, err := tx.FindRecordById("jobs", a.GetString("job"))
			if err != nil {
				return err
			}
			if strings.HasPrefix(job.GetString("number"), "P") {
				return workflowError("Proposals do not use project-value approval.")
			}
			if jobUpdates != nil || allocations != nil {
				if e.Request.FormValue("job_updated") != job.GetString("updated") {
					return apis.NewApiError(http.StatusConflict, "The job changed. Reload it before saving corrections.", nil)
				}
				token, err := valueReviewToken(tx, job)
				if err != nil {
					return err
				}
				if e.Request.FormValue("job_review_token") != token {
					return apis.NewApiError(http.StatusConflict, "The job or its related data changed. Reload it before saving corrections.", nil)
				}
			}
			jobChanged, err := applyValueJobCorrections(job, jobUpdates)
			if err != nil {
				return err
			}
			allocationChanged := false
			existingAllocations, finalAllocations, err := storedValueAllocations(tx, job.Id)
			if err != nil {
				return err
			}
			if allocations != nil {
				finalAllocations = allocations
				existing := []existingJobAllocation{}
				for _, row := range existingAllocations {
					existing = append(existing, existingJobAllocation{row.GetString("division"), row.GetFloat("hours")})
				}
				allocationChanged = allocationsDiffer(existing, allocations)
			}
			if err = validateRequestAllocationInputs(tx, finalAllocations, true); err != nil {
				return err
			}
			if jobChanged || allocationChanged {
				effective := job.Get("project_value")
				job.Set("project_value", value)
				if err = hooks.ProcessJobWorkflow(tx, job, e.Auth, hooks.ProjectWorkflowMutationApproveValue); err != nil {
					return err
				}
				job.Set("project_value", effective)
				ctx := hooks.WithProjectWorkflowMutation(e.Request.Context(), hooks.ProjectWorkflowMutationValue)
				if err = saveHistoricalEffectiveValue(tx, ctx, job); err != nil {
					return err
				}
				if allocationChanged {
					for _, row := range existingAllocations {
						if err = tx.Delete(row); err != nil {
							return err
						}
					}
					collection, err := tx.FindCollectionByNameOrId("job_time_allocations")
					if err != nil {
						return err
					}
					for _, allocation := range allocations {
						row := core.NewRecord(collection)
						row.Set("job", job.Id)
						row.Set("division", allocation.Division)
						row.Set("hours", allocation.Hours)
						if err = tx.Save(row); err != nil {
							return err
						}
					}
				}
			} else if err = validateValueJob(tx, job, e.Auth, value); err != nil {
				return err
			}
			count, err := tx.CountRecords("job_attachments", dbx.And(dbx.HashExp{"job": job.Id}, dbx.Not(dbx.HashExp{"id": a.Id}), dbx.Not(dbx.HashExp{"status": "Approved"})))
			if err != nil {
				return err
			}
			if count > 0 {
				return workflowError("Resolve the current pending value change first.")
			}
			currentToken, err := valueReviewToken(tx, job)
			if err != nil {
				return err
			}
			// A save is unchanged only while the job and related records match the
			// submitted data, or for a rejected change, the data as rejected. A
			// correction after rejection needs fresh approval.
			if !creating && !jobChanged && !allocationChanged && a.GetString("review_token") == currentToken && a.GetFloat("project_value") == value && a.GetString("assigned_approver") == assigned && (file == nil || hash == a.GetString("document_hash")) {
				if file != nil {
					if err = ensureAuthorizationFile(tx, a, file); err != nil {
						return err
					}
				}
				result = a
				return nil
			}
			clearAttachmentApproval(a)
			a.Set("project_value", value)
			a.Set("assigned_approver", assigned)
			// A real pending-data change invalidates earlier edit/review/delete forms.
			// The unchanged-save path above deliberately keeps the current version.
			a.Set("version", a.GetInt("version")+1)
			if file != nil {
				if err = checkAuthorizationHash(tx, hash, a.Id); err != nil {
					return err
				}
				a.Set("document", file)
				a.Set("document_hash", hash)
				a.Set("uploader", e.Auth.Id)
				a.Set("uploaded", time.Now().UTC())
			}
			a.Set("review_token", currentToken)
			if err = tx.SaveWithContext(hooks.WithProjectWorkflowMutation(e.Request.Context(), hooks.ProjectWorkflowMutationValue), a); err != nil {
				return err
			}
			if err = queueValueNotice(tx, a, "submitted", e.Auth.Id); err != nil {
				return err
			}
			result = a
			return nil
		})
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, result)
	}
}
func attachmentActionHandler(app core.App, approve bool) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var input workflowAction
		if err := e.BindBody(&input); err != nil {
			return workflowError("Invalid action data.")
		}
		var result *core.Record
		err := app.RunInTransaction(func(tx core.App) error {
			if err := workflowClaim(tx, e.Auth, "project_authorization"); err != nil {
				return err
			}
			a, err := tx.FindRecordById("job_attachments", e.Request.PathValue("id"))
			if err != nil {
				return err
			}
			if a.GetString("job") == "" || a.GetString("status") != "Pending" {
				return workflowError("Only a pending value change can be reviewed.")
			}
			if a.GetString("assigned_approver") != e.Auth.Id {
				return apis.NewForbiddenError("Only the selected approver can review this value change.", nil)
			}
			// Review applies to the exact attachment revision and PDF the user saw.
			if a.GetInt("version") != input.Version || a.GetString("document_hash") != input.DocumentHash {
				return apis.NewApiError(http.StatusConflict, "The reviewed value change changed. Reload it.", nil)
			}
			ctx := hooks.WithProjectWorkflowMutation(e.Request.Context(), hooks.ProjectWorkflowMutationApproveValue)
			if !approve {
				if strings.TrimSpace(input.RejectionReason) == "" {
					return workflowError("Enter a rejection reason.")
				}
				job, err := tx.FindRecordById("jobs", a.GetString("job"))
				if err != nil {
					return err
				}
				// Record the data the approver saw as rejected. Only a later
				// correction can return the change for approval.
				token, err := valueReviewToken(tx, job)
				if err != nil {
					return err
				}
				if input.ReviewToken != token {
					return apis.NewApiError(http.StatusConflict, "The job or its related data changed. Reload it before rejecting.", nil)
				}
				a.Set("review_token", token)
				a.Set("status", "Rejected")
				a.Set("rejector", e.Auth.Id)
				a.Set("rejected", time.Now().UTC())
				a.Set("rejection_reason", strings.TrimSpace(input.RejectionReason))
			} else {
				if err := ensureAuthorizationFile(tx, a, nil); err != nil {
					return err
				}
				limit, err := utilities.ProjectAuthorizationLimit(tx, e.Auth.Id, a.GetFloat("project_value"))
				if err != nil {
					return err
				}
				job, err := tx.FindRecordById("jobs", a.GetString("job"))
				if err != nil {
					return err
				}
				token, err := valueReviewToken(tx, job)
				if err != nil {
					return err
				}
				if token != a.GetString("review_token") {
					return apis.NewApiError(http.StatusConflict, "The job changed. Ask a job editor to save and review this value change again.", nil)
				}
				previous := job.GetFloat("project_value")
				_, allocations, err := storedValueAllocations(tx, job.Id)
				if err != nil {
					return err
				}
				if err = validateRequestAllocationInputs(tx, allocations, true); err != nil {
					return err
				}
				job.Set("project_value", a.Get("project_value"))
				if err = hooks.ProcessJobWorkflow(tx, job, e.Auth, hooks.ProjectWorkflowMutationApproveValue); err != nil {
					return err
				}
				rows, err := tx.FindRecordsByFilter("job_attachments", "job={:id} && status='Approved'", "-approval_sequence", 1, 0, dbx.Params{"id": job.Id})
				if err != nil {
					return err
				}
				// Allocate the next retained approval position in this transaction.
				// Timestamp/ID sorting cannot reliably establish which value came last.
				// A removed final position may be reused; this is not an event ID.
				seq := 1
				if len(rows) > 0 {
					seq = rows[0].GetInt("approval_sequence") + 1
				}
				a.Set("previous_project_value", previous)
				a.Set("approval_sequence", seq)
				a.Set("approver", e.Auth.Id)
				a.Set("approved", time.Now().UTC())
				a.Set("approval_max_value", limit)
				a.Set("status", "Approved")
				if err = tx.SaveWithContext(ctx, job); err != nil {
					return err
				}
			}
			// A decision changes the revision even when the PDF and value are the
			// same. This invalidates a delete confirmation opened while pending.
			a.Set("version", a.GetInt("version")+1)
			if err = tx.SaveWithContext(ctx, a); err != nil {
				return err
			}
			action := "rejected"
			if approve {
				action = "approved"
			}
			if err = queueValueNotice(tx, a, action, e.Auth.Id); err != nil {
				return err
			}
			result = a
			return nil
		})
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, result)
	}
}
func attachmentDeleteHandler(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var input workflowAction
		if err := e.BindBody(&input); err != nil {
			return workflowError("Invalid action data.")
		}
		err := app.RunInTransaction(func(tx core.App) error {
			if err := workflowClaim(tx, e.Auth, "admin"); err != nil {
				return err
			}
			a, err := tx.FindRecordById("job_attachments", e.Request.PathValue("id"))
			if err != nil {
				return err
			}
			if a.GetString("job") == "" {
				return workflowError("Use abandoned-request deletion for this attachment.")
			}
			if a.GetBool("initial_approval") {
				return workflowError("The initial project approval cannot be deleted.")
			}
			// Check the reviewed revision before choosing pending cleanup or approved
			// value restoration. A stale pending-delete must never reverse an approval.
			if a.GetInt("version") != input.Version || a.GetString("document_hash") != input.DocumentHash {
				return apis.NewApiError(http.StatusConflict, "This value change changed. Reload it and confirm deletion again.", nil)
			}
			ctx := hooks.WithProjectWorkflowMutation(e.Request.Context(), hooks.ProjectWorkflowMutationDelete)
			if a.GetString("status") == "Approved" {
				pending, err := tx.CountRecords("job_attachments", dbx.And(dbx.HashExp{"job": a.GetString("job")}, dbx.Not(dbx.HashExp{"status": "Approved"})))
				if err != nil {
					return err
				}
				if pending > 0 {
					return workflowError("Delete the pending or rejected value change first.")
				}
				rows, err := tx.FindRecordsByFilter("job_attachments", "job={:id} && status='Approved'", "-approval_sequence", 1, 0, dbx.Params{"id": a.GetString("job")})
				if err != nil {
					return err
				}
				// Only the latest retained sequence can restore its previous value;
				// deleting an earlier approval would break the value history.
				if len(rows) != 1 || rows[0].Id != a.Id {
					return workflowError("Only the latest approved value change can be deleted.")
				}
				job, err := tx.FindRecordById("jobs", a.GetString("job"))
				if err != nil {
					return err
				}
				if job.GetFloat("project_value") != a.GetFloat("project_value") {
					return apis.NewApiError(http.StatusConflict, "The current project value does not match this approval.", nil)
				}
				job.Set("project_value", a.Get("previous_project_value"))
				if err = hooks.ProcessJobWorkflow(tx, job, e.Auth, hooks.ProjectWorkflowMutationRestoreValue); err != nil {
					return err
				}
				restoreContext := hooks.WithProjectWorkflowMutation(e.Request.Context(), hooks.ProjectWorkflowMutationRestoreValue)
				historical := job.GetFloat("project_value")
				if historical < 0 {
					// Validate all other fields, then restore only the exact recorded value.
					job.Set("project_value", 0)
					err = tx.Validate(job)
					job.Set("project_value", historical)
					if err != nil {
						return err
					}
					err = tx.SaveNoValidateWithContext(restoreContext, job)
				} else {
					err = tx.SaveWithContext(restoreContext, job)
				}
				if err != nil {
					return err
				}
			}
			return tx.DeleteWithContext(ctx, a)
		})
		if err != nil {
			return err
		}
		return e.NoContent(http.StatusNoContent)
	}
}

// Check object existence without downloading the PDF. Only an editable upload
// whose computed hash matches the record may supply identicalUpload for recovery.
func ensureAuthorizationFile(app core.App, attachment *core.Record, identicalUpload *filesystem.File) error {
	fs, err := app.NewFilesystem()
	if err != nil {
		return err
	}
	defer fs.Close()
	path := attachment.BaseFilesPath() + "/" + attachment.GetString("document")
	exists, err := fs.Exists(path)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if identicalUpload != nil {
		return fs.UploadFile(identicalUpload, path)
	}
	return workflowError("The authorization PDF is unavailable. Restore or replace it before approval.")
}

func queueValueNotice(app core.App, attachment *core.Record, action, actor string) error {
	job, err := app.FindRecordById("jobs", attachment.GetString("job"))
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, uid := range []string{attachment.GetString("assigned_approver"), attachment.GetString("uploader")} {
		if uid == "" || seen[uid] {
			continue
		}
		seen[uid] = true
		_, err := notifications.DispatchNotification(app, notifications.DispatchArgs{TemplateCode: "job_value_workflow", RecipientUID: uid, System: true, ActorUID: actor, Mode: notifications.DeliveryDeferred, Data: map[string]any{"JobNumber": job.GetString("number"), "Description": job.GetString("description"), "Action": action, "ActionURL": notifications.BuildActionURL(app, "/jobs/"+job.Id+"/details")}})
		if err != nil {
			return err
		}
	}
	return nil
}

// Corrections are saved with the proposed value review; the effective value stays unchanged.
func applyValueJobCorrections(job *core.Record, updates map[string]any) (bool, error) {
	allowed := map[string]bool{}
	for _, field := range []string{"status", "description", "contact", "invoicing_information", "manager", "alternate_manager", "branch", "rate_sheet", "location", "project_award_date", "project_completion_date", "job_owner", "fn_agreement", "time_and_materials", "client_po", "client_reference_number", "authorizing_document"} {
		allowed[field] = true
	}
	changed := false
	for key, value := range updates {
		if !allowed[key] {
			return false, workflowError("Field is not editable in a value proposal: " + key)
		}
		if key == "fn_agreement" || key == "time_and_materials" {
			if _, valid := value.(bool); !valid {
				return false, workflowError("Use a Boolean value for " + key)
			}
		}
		before := job.Get(key)
		job.Set(key, value)
		if !reflect.DeepEqual(before, job.Get(key)) {
			changed = true
		}
	}
	return changed, nil
}
func saveHistoricalEffectiveValue(app core.App, ctx context.Context, job *core.Record) error {
	historical := job.GetFloat("project_value")
	if historical >= 0 {
		return app.SaveWithContext(ctx, job)
	}
	job.Set("project_value", 0)
	err := app.Validate(job)
	job.Set("project_value", historical)
	if err != nil {
		return err
	}
	return app.SaveNoValidateWithContext(ctx, job)
}
