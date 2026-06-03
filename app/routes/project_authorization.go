package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
	"tybalt/errs"
	"tybalt/hooks"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

type projectAuthorizationApproveRequest struct {
	ProjectAuthorizationDocHash string `json:"project_authorization_doc_hash"`
}

type projectAuthorizationQueueRow struct {
	ID                          string `db:"id" json:"id"`
	Number                      string `db:"number" json:"number"`
	Description                 string `db:"description" json:"description"`
	ClientID                    string `db:"client_id" json:"client_id"`
	ClientName                  string `db:"client_name" json:"client_name"`
	ManagerID                   string `db:"manager_id" json:"manager_id"`
	ManagerName                 string `db:"manager_name" json:"manager_name"`
	BranchID                    string `db:"branch_id" json:"branch_id"`
	BranchCode                  string `db:"branch_code" json:"branch_code"`
	BranchName                  string `db:"branch_name" json:"branch_name"`
	Status                      string `db:"status" json:"status"`
	ProjectAuthorizationDoc     string `db:"project_authorization_doc" json:"project_authorization_doc"`
	ProjectAuthorizationDocURL  string `json:"project_authorization_doc_url"`
	ProjectAuthorizationDocHash string `db:"project_authorization_doc_hash" json:"project_authorization_doc_hash"`
}

func createUploadProjectAuthorizationDocumentHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil || e.Auth.Id == "" {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}
		job, err := app.FindRecordById("jobs", e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("job not found", err)
		}
		if strings.HasPrefix(job.GetString("number"), "P") {
			return projectAuthorizationRouteError(e, projectAuthorizationAPIError(
				http.StatusBadRequest,
				"not_project_job",
				"project authorization documents can only be uploaded to project jobs",
			))
		}
		allowed, err := hooks.CanUploadProjectAuthorizationDocument(app, job, e.Auth)
		if err != nil {
			return e.InternalServerError("failed to check upload permissions", err)
		}
		if !allowed {
			return e.ForbiddenError("you do not have permission to upload this project authorization document", nil)
		}

		files, err := e.FindUploadedFiles("project_authorization_doc")
		if err != nil && !errors.Is(err, http.ErrMissingFile) {
			return e.BadRequestError("failed to read uploaded PA document", err)
		}
		if len(files) != 1 {
			return projectAuthorizationRouteError(e, projectAuthorizationFieldAPIError(
				http.StatusBadRequest,
				"required",
				"upload exactly one signed PA PDF",
				"project_authorization_doc",
			))
		}
		if !uploadedFileLooksLikePDF(files[0]) {
			return projectAuthorizationRouteError(e, projectAuthorizationFieldAPIError(
				http.StatusBadRequest,
				"invalid_mime_type",
				"project authorization document must be a PDF",
				"project_authorization_doc",
			))
		}
		attachmentHash, err := hashUploadedFileSHA256(files[0])
		if err != nil {
			return e.InternalServerError("failed to hash project authorization document", err)
		}

		var uploaded *core.Record
		err = app.RunInTransaction(func(txApp core.App) error {
			txJob, err := txApp.FindRecordById("jobs", job.Id)
			if err != nil {
				return projectAuthorizationAPIError(http.StatusNotFound, "job_not_found", "job not found")
			}
			if strings.HasPrefix(txJob.GetString("number"), "P") {
				return projectAuthorizationAPIError(
					http.StatusBadRequest,
					"not_project_job",
					"project authorization documents can only be uploaded to project jobs",
				)
			}
			allowed, err := hooks.CanUploadProjectAuthorizationDocument(txApp, txJob, e.Auth)
			if err != nil {
				return err
			}
			if !allowed {
				return projectAuthorizationAPIError(http.StatusForbidden, "forbidden", "you do not have permission to upload this project authorization document")
			}
			if txJob.GetString("pa_reviewed") != "" || txJob.GetString("pa_reviewer") != "" {
				return projectAuthorizationFieldAPIError(
					http.StatusBadRequest,
					"project_authorization_approved_immutable",
					"revoke PA approval before replacing or removing the uploaded document",
					"project_authorization_doc",
				)
			}
			existingJob, _ := txApp.FindFirstRecordByFilter("jobs", "project_authorization_doc_hash = {:hash} && id != {:id}", dbx.Params{
				"hash": attachmentHash,
				"id":   txJob.Id,
			})
			if existingJob != nil {
				return projectAuthorizationFieldAPIError(
					http.StatusBadRequest,
					"duplicate_file",
					"this PA document has already been uploaded to another job",
					"project_authorization_doc",
				)
			}

			form := forms.NewRecordUpsert(txApp, txJob)
			form.SetContext(hooks.WithProjectAuthorizationMutation(e.Request.Context(), hooks.ProjectAuthorizationMutationUpload))
			form.GrantSuperuserAccess()
			form.Load(map[string]any{
				"project_authorization_doc":      files,
				"project_authorization_doc_hash": attachmentHash,
				"pa_reviewer":                    "",
				"pa_reviewed":                    "",
			})
			if err := form.Submit(); err != nil {
				return err
			}
			uploaded = txJob
			return nil
		})
		if err != nil {
			return projectAuthorizationRouteError(e, err)
		}
		return e.JSON(http.StatusOK, uploaded)
	}
}

func createDeleteProjectAuthorizationDocumentHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil || e.Auth.Id == "" {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}
		job, err := app.FindRecordById("jobs", e.Request.PathValue("id"))
		if err != nil {
			return e.NotFoundError("job not found", err)
		}
		if strings.HasPrefix(job.GetString("number"), "P") {
			return projectAuthorizationRouteError(e, projectAuthorizationAPIError(
				http.StatusBadRequest,
				"not_project_job",
				"project authorization documents can only be removed from project jobs",
			))
		}
		allowed, err := hooks.CanUploadProjectAuthorizationDocument(app, job, e.Auth)
		if err != nil {
			return e.InternalServerError("failed to check upload permissions", err)
		}
		if !allowed {
			return e.ForbiddenError("you do not have permission to remove this project authorization document", nil)
		}
		if job.GetString("pa_reviewed") != "" || job.GetString("pa_reviewer") != "" {
			return projectAuthorizationRouteError(e, projectAuthorizationFieldAPIError(
				http.StatusBadRequest,
				"project_authorization_approved_immutable",
				"revoke PA approval before replacing or removing the uploaded document",
				"project_authorization_doc",
			))
		}

		var cleared *core.Record
		err = app.RunInTransaction(func(txApp core.App) error {
			txJob, err := txApp.FindRecordById("jobs", job.Id)
			if err != nil {
				return projectAuthorizationAPIError(http.StatusNotFound, "job_not_found", "job not found")
			}
			if txJob.GetString("pa_reviewed") != "" || txJob.GetString("pa_reviewer") != "" {
				return projectAuthorizationFieldAPIError(
					http.StatusBadRequest,
					"project_authorization_approved_immutable",
					"revoke PA approval before replacing or removing the uploaded document",
					"project_authorization_doc",
				)
			}
			txJob.Set("project_authorization_doc", "")
			txJob.Set("project_authorization_doc_hash", "")
			txJob.Set("pa_reviewer", "")
			txJob.Set("pa_reviewed", "")
			if err := txApp.SaveWithContext(hooks.WithProjectAuthorizationMutation(e.Request.Context(), hooks.ProjectAuthorizationMutationDelete), txJob); err != nil {
				return err
			}
			cleared = txJob
			return nil
		})
		if err != nil {
			return projectAuthorizationRouteError(e, err)
		}
		return e.JSON(http.StatusOK, cleared)
	}
}

func createGetProjectAuthorizationQueueHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAccountingClaim(app, e.Auth); err != nil {
			return projectAuthorizationRouteError(e, err)
		}
		rows := []projectAuthorizationQueueRow{}
		if err := app.DB().NewQuery(`
			SELECT
			  j.id,
			  j.number,
			  j.description,
			  j.client AS client_id,
			  c.name AS client_name,
			  j.manager AS manager_id,
			  TRIM(COALESCE(mp.given_name, '') || ' ' || COALESCE(mp.surname, '')) AS manager_name,
			  j.branch AS branch_id,
			  b.code AS branch_code,
			  b.name AS branch_name,
			  j.status,
			  j.project_authorization_doc,
			  j.project_authorization_doc_hash
			FROM jobs j
			LEFT JOIN clients c ON c.id = j.client
			LEFT JOIN profiles mp ON mp.uid = j.manager
			LEFT JOIN branches b ON b.id = j.branch
			WHERE j.status = 'Active'
			  AND j.number NOT LIKE 'P%'
			  AND j.project_authorization_doc != ''
			  AND j.project_authorization_doc_hash != ''
			  AND j.pa_reviewed = ''
			  AND j.pa_reviewer = ''
			ORDER BY j.number
		`).All(&rows); err != nil {
			return e.InternalServerError("failed to load PA approval queue", err)
		}
		jobsCollection, err := app.FindCollectionByNameOrId("jobs")
		if err != nil {
			return e.InternalServerError("failed to load jobs collection", err)
		}
		for i := range rows {
			if rows[i].ProjectAuthorizationDoc != "" {
				rows[i].ProjectAuthorizationDocURL = "/api/files/" + jobsCollection.Id + "/" + rows[i].ID + "/" + rows[i].ProjectAuthorizationDoc
			}
		}
		return e.JSON(http.StatusOK, map[string]any{"items": rows})
	}
}

func createApproveProjectAuthorizationHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAccountingClaim(app, e.Auth); err != nil {
			return projectAuthorizationRouteError(e, err)
		}
		var req projectAuthorizationApproveRequest
		if err := e.BindBody(&req); err != nil {
			return e.BadRequestError("invalid approval request", err)
		}
		submittedHash := strings.TrimSpace(req.ProjectAuthorizationDocHash)
		var approved *core.Record
		err := app.RunInTransaction(func(txApp core.App) error {
			job, err := txApp.FindRecordById("jobs", e.Request.PathValue("id"))
			if err != nil {
				return projectAuthorizationAPIError(http.StatusNotFound, "job_not_found", "job not found")
			}
			if job.GetString("project_authorization_doc") == "" {
				return projectAuthorizationFieldAPIError(http.StatusBadRequest, "required", "project authorization document is required before approval", "project_authorization_doc")
			}
			currentHash := strings.TrimSpace(job.GetString("project_authorization_doc_hash"))
			if currentHash == "" || currentHash != submittedHash {
				return projectAuthorizationAPIError(http.StatusConflict, "project_authorization_doc_changed", "The project authorization document changed after you opened it. Please review the current document before approving.")
			}
			if job.GetString("pa_reviewed") != "" || job.GetString("pa_reviewer") != "" {
				return projectAuthorizationAPIError(http.StatusConflict, "project_authorization_already_approved", "This project authorization document has already been approved.")
			}
			job.Set("pa_reviewed", time.Now().UTC())
			job.Set("pa_reviewer", e.Auth.Id)
			if err := txApp.SaveWithContext(hooks.WithProjectAuthorizationMutation(e.Request.Context(), hooks.ProjectAuthorizationMutationApprove), job); err != nil {
				return err
			}
			approved = job
			return nil
		})
		if err != nil {
			return projectAuthorizationRouteError(e, err)
		}
		return e.JSON(http.StatusOK, approved)
	}
}

func createRevokeProjectAuthorizationHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminClaim(app, e.Auth); err != nil {
			return projectAuthorizationRouteError(e, err)
		}
		var revoked *core.Record
		err := app.RunInTransaction(func(txApp core.App) error {
			job, err := txApp.FindRecordById("jobs", e.Request.PathValue("id"))
			if err != nil {
				return projectAuthorizationAPIError(http.StatusNotFound, "job_not_found", "job not found")
			}
			job.Set("pa_reviewed", "")
			job.Set("pa_reviewer", "")
			if err := txApp.SaveWithContext(hooks.WithProjectAuthorizationMutation(e.Request.Context(), hooks.ProjectAuthorizationMutationRevoke), job); err != nil {
				return err
			}
			revoked = job
			return nil
		})
		if err != nil {
			return projectAuthorizationRouteError(e, err)
		}
		return e.JSON(http.StatusOK, revoked)
	}
}

func uploadedFileLooksLikePDF(file *filesystem.File) bool {
	if file == nil || file.Reader == nil {
		return false
	}
	reader, err := file.Reader.Open()
	if err != nil {
		return false
	}
	defer reader.Close()
	buf := make([]byte, 512)
	n, _ := reader.Read(buf)
	return http.DetectContentType(buf[:n]) == "application/pdf"
}

func hashUploadedFileSHA256(file *filesystem.File) (string, error) {
	reader, err := file.Reader.Open()
	if err != nil {
		return "", err
	}
	defer reader.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func requireAccountingClaim(app core.App, auth *core.Record) error {
	if auth == nil || auth.Id == "" {
		return projectAuthorizationAPIError(http.StatusUnauthorized, "unauthorized", "unauthorized")
	}
	hasClaim, err := utilities.HasClaim(app, auth, "accounting")
	if err != nil {
		return err
	}
	if !hasClaim {
		return projectAuthorizationAPIError(http.StatusForbidden, "forbidden", "you do not have permission to approve project authorization documents")
	}
	return nil
}

func requireAdminClaim(app core.App, auth *core.Record) error {
	if auth == nil || auth.Id == "" {
		return projectAuthorizationAPIError(http.StatusUnauthorized, "unauthorized", "unauthorized")
	}
	hasClaim, err := utilities.HasClaim(app, auth, "admin")
	if err != nil {
		return err
	}
	if !hasClaim {
		return projectAuthorizationAPIError(http.StatusForbidden, "forbidden", "you do not have permission to revoke project authorization approvals")
	}
	return nil
}

type projectAuthorizationHTTPError struct {
	status  int
	code    string
	message string
	field   string
}

func (e *projectAuthorizationHTTPError) Error() string {
	return e.message
}

func projectAuthorizationAPIError(status int, code string, message string) *projectAuthorizationHTTPError {
	return &projectAuthorizationHTTPError{status: status, code: code, message: message}
}

func projectAuthorizationFieldAPIError(status int, code string, message string, field string) *projectAuthorizationHTTPError {
	return &projectAuthorizationHTTPError{status: status, code: code, message: message, field: field}
}

func projectAuthorizationRouteError(e *core.RequestEvent, err error) error {
	var httpErr *projectAuthorizationHTTPError
	if errors.As(err, &httpErr) {
		body := map[string]any{"code": httpErr.code, "message": httpErr.message}
		if httpErr.field != "" {
			body["data"] = map[string]any{httpErr.field: map[string]any{"code": httpErr.code, "message": httpErr.message}}
		}
		return e.JSON(httpErr.status, body)
	}
	var hookErr *errs.HookError
	if errors.As(err, &hookErr) {
		return e.JSON(hookErr.Status, hookErr)
	}
	if strings.Contains(err.Error(), "idx_jobs_project_authorization_doc_hash") ||
		strings.Contains(err.Error(), "UNIQUE constraint failed: jobs.project_authorization_doc_hash") {
		return e.JSON(http.StatusBadRequest, map[string]any{
			"code":    "duplicate_file",
			"message": "this PA document has already been uploaded to another job",
			"data": map[string]any{
				"project_authorization_doc": map[string]any{"code": "duplicate_file", "message": "this PA document has already been uploaded to another job"},
			},
		})
	}
	return e.Error(http.StatusInternalServerError, "project authorization request failed", err)
}
