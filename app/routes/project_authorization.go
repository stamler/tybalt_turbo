package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"tybalt/errs"
	"tybalt/hooks"
	"tybalt/notifications"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

const duplicateProjectAuthorizationDocMessage = "this PA document has already been uploaded to another job"

const (
	projectAuthorizationMissingPriorityAll     = "all"
	projectAuthorizationMissingPriorityInUse   = "in_use"
	projectAuthorizationMissingPriorityRecent  = "recent"
	projectAuthorizationMissingPriorityDormant = "dormant"
	projectAuthorizationMissingDefaultLimit    = 50
	projectAuthorizationMissingMaxLimit        = 200
)

type projectAuthorizationApproveRequest struct {
	ProjectAuthorizationDocHash string `json:"project_authorization_doc_hash"`
}

type projectAuthorizationRejectRequest struct {
	ProjectAuthorizationDocHash string `json:"project_authorization_doc_hash"`
	RejectionReason             string `json:"rejection_reason"`
}

type projectAuthorizationQueueRow struct {
	ID                          string `db:"id" json:"id"`
	Number                      string `db:"number" json:"number"`
	Description                 string `db:"description" json:"description"`
	ClientPO                    string `db:"client_po" json:"client_po"`
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

type projectAuthorizationMissingRow struct {
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
	ProjectAwardDate            string `db:"project_award_date" json:"project_award_date"`
	Updated                     string `db:"updated" json:"updated"`
	ProjectAuthorizationDoc     string `db:"project_authorization_doc" json:"project_authorization_doc"`
	ProjectAuthorizationDocHash string `db:"project_authorization_doc_hash" json:"project_authorization_doc_hash"`
	ProjectAuthorizationState   string `db:"project_authorization_state" json:"project_authorization_state"`
	TimeEntryCount              int    `db:"time_entry_count" json:"time_entry_count"`
	PurchaseOrderCount          int    `db:"purchase_order_count" json:"purchase_order_count"`
	ActivePurchaseOrderCount    int    `db:"active_purchase_order_count" json:"active_purchase_order_count"`
	ExpenseCount                int    `db:"expense_count" json:"expense_count"`
	LatestActivityDate          string `db:"latest_activity_date" json:"latest_activity_date"`
	Priority                    string `db:"priority" json:"priority"`
	CanUpload                   bool   `db:"can_upload" json:"can_upload"`
}

type projectAuthorizationRejectedRow struct {
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
	PARejectorID                string `db:"pa_rejector_id" json:"pa_rejector_id"`
	PARejectorName              string `db:"pa_rejector_name" json:"pa_rejector_name"`
	PARejected                  string `db:"pa_rejected" json:"pa_rejected"`
	PARejectionReason           string `db:"pa_rejection_reason" json:"pa_rejection_reason"`
	CanUpload                   bool   `db:"can_upload" json:"can_upload"`
}

type projectAuthorizationMissingCountRow struct {
	Priority string `db:"priority"`
	Count    int    `db:"count"`
}

type projectAuthorizationMissingResponse struct {
	Items              []projectAuthorizationMissingRow `json:"items"`
	Counts             map[string]int                   `json:"counts"`
	Page               int                              `json:"page"`
	Limit              int                              `json:"limit"`
	Total              int                              `json:"total"`
	TotalPages         int                              `json:"total_pages"`
	Priority           string                           `json:"priority"`
	PendingReviewCount int                              `json:"pending_review_count"`
	RejectedCount      int                              `json:"rejected_count"`
}

const projectAuthorizationMissingBaseQuery = `
	WITH missing_jobs AS (
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
		  j.project_award_date,
		  j.created,
		  j.updated,
		  j.project_authorization_doc,
		  j.project_authorization_doc_hash,
		  CASE
		    WHEN COALESCE(j.project_authorization_doc, '') = '' THEN 'missing_pdf'
		    ELSE 'missing_hash'
		  END AS project_authorization_state,
		  CASE
		    WHEN {:canUploadAll} = 1
		      OR j.manager = {:uid}
		      OR j.alternate_manager = {:uid}
		      OR b.manager = {:uid}
		    THEN 1
		    ELSE 0
		  END AS can_upload
		FROM jobs j
		LEFT JOIN clients c ON c.id = j.client
		LEFT JOIN profiles mp ON mp.uid = j.manager
		LEFT JOIN branches b ON b.id = j.branch
		WHERE j.status = 'Active'
		  AND j.number NOT LIKE 'P%'
		  AND (
		    COALESCE(j.project_authorization_doc, '') = ''
		    OR COALESCE(j.project_authorization_doc_hash, '') = ''
		  )
		  AND (
		    {:canSeeAll} = 1
		    OR j.manager = {:uid}
		    OR j.alternate_manager = {:uid}
		    OR b.manager = {:uid}
		  )
	),
	time_activity AS (
		SELECT
		  te.job AS job_id,
		  COUNT(*) AS time_entry_count,
		  MAX(te.date) AS latest_time_entry_date
		FROM time_entries te
		JOIN missing_jobs mj ON mj.id = te.job
		WHERE te.job != ''
		GROUP BY te.job
	),
	purchase_order_activity AS (
		SELECT
		  po.job AS job_id,
		  COUNT(*) AS purchase_order_count,
		  SUM(CASE WHEN po.status = 'Active' THEN 1 ELSE 0 END) AS active_purchase_order_count,
		  MAX(po.date) AS latest_purchase_order_date
		FROM purchase_orders po
		JOIN missing_jobs mj ON mj.id = po.job
		WHERE po.job != ''
		GROUP BY po.job
	),
	expense_activity AS (
		SELECT
		  e.job AS job_id,
		  COUNT(*) AS expense_count,
		  MAX(e.date) AS latest_expense_date
		FROM expenses e
		JOIN missing_jobs mj ON mj.id = e.job
		WHERE e.job != ''
		GROUP BY e.job
	),
	enriched AS (
		SELECT
		  mj.*,
		  COALESCE(ta.time_entry_count, 0) AS time_entry_count,
		  COALESCE(poa.purchase_order_count, 0) AS purchase_order_count,
		  COALESCE(poa.active_purchase_order_count, 0) AS active_purchase_order_count,
		  COALESCE(ea.expense_count, 0) AS expense_count,
		  MAX(
		    COALESCE(ta.latest_time_entry_date, ''),
		    COALESCE(poa.latest_purchase_order_date, ''),
		    COALESCE(ea.latest_expense_date, '')
		  ) AS latest_activity_date
		FROM missing_jobs mj
		LEFT JOIN time_activity ta ON ta.job_id = mj.id
		LEFT JOIN purchase_order_activity poa ON poa.job_id = mj.id
		LEFT JOIN expense_activity ea ON ea.job_id = mj.id
	),
	classified AS (
		SELECT
		  *,
		  CASE
		    WHEN time_entry_count + purchase_order_count + expense_count > 0 THEN 'in_use'
		    WHEN COALESCE(project_award_date, '') >= date('now', '-90 days')
		      OR substr(COALESCE(updated, ''), 1, 10) >= date('now', '-90 days')
		      OR substr(COALESCE(created, ''), 1, 10) >= date('now', '-90 days')
		    THEN 'recent'
		    ELSE 'dormant'
		  END AS priority
		FROM enriched
	)
`

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
			return projectAuthorizationRouteError(e, projectAuthorizationNotProjectJobAPIError("uploaded to"))
		}
		allowed, err := hooks.CanUploadProjectAuthorizationDocument(app, job, e.Auth)
		if err != nil {
			return e.InternalServerError("failed to check upload permissions", err)
		}
		if !allowed {
			return e.ForbiddenError("you do not have permission to upload this project authorization document", nil)
		}
		if !projectAuthorizationUploadCertified(e.Request.FormValue("project_authorization_certified")) {
			return projectAuthorizationRouteError(e, projectAuthorizationFieldAPIError(
				http.StatusBadRequest,
				"required",
				"confirm that the PDF contains a completed TBT Engineering Project Authorization Form",
				"project_authorization_certified",
			))
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
				return projectAuthorizationNotProjectJobAPIError("uploaded to")
			}
			allowed, err := hooks.CanUploadProjectAuthorizationDocument(txApp, txJob, e.Auth)
			if err != nil {
				return err
			}
			if !allowed {
				return projectAuthorizationAPIError(http.StatusForbidden, "forbidden", "you do not have permission to upload this project authorization document")
			}
			if projectAuthorizationReviewMetadataPresent(txJob) {
				return projectAuthorizationApprovedImmutableAPIError()
			}
			existingJob, _ := txApp.FindFirstRecordByFilter("jobs", "project_authorization_doc_hash = {:hash} && id != {:id}", dbx.Params{
				"hash": attachmentHash,
				"id":   txJob.Id,
			})
			if existingJob != nil {
				return duplicateProjectAuthorizationDocAPIError()
			}

			form := forms.NewRecordUpsert(txApp, txJob)
			form.SetContext(hooks.WithProjectAuthorizationMutation(e.Request.Context(), hooks.ProjectAuthorizationMutationUpload))
			form.GrantSuperuserAccess()
			txJob.Set("pa_uploader", e.Auth.Id)
			txJob.Set("pa_uploaded", time.Now().UTC())
			txJob.Set("pa_rejector", "")
			txJob.Set("pa_rejected", "")
			txJob.Set("pa_rejection_reason", "")
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

func projectAuthorizationUploadCertified(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
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
			return projectAuthorizationRouteError(e, projectAuthorizationNotProjectJobAPIError("removed from"))
		}
		allowed, err := hooks.CanUploadProjectAuthorizationDocument(app, job, e.Auth)
		if err != nil {
			return e.InternalServerError("failed to check upload permissions", err)
		}
		if !allowed {
			return e.ForbiddenError("you do not have permission to remove this project authorization document", nil)
		}
		if projectAuthorizationReviewMetadataPresent(job) {
			return projectAuthorizationRouteError(e, projectAuthorizationApprovedImmutableAPIError())
		}

		var cleared *core.Record
		err = app.RunInTransaction(func(txApp core.App) error {
			txJob, err := txApp.FindRecordById("jobs", job.Id)
			if err != nil {
				return projectAuthorizationAPIError(http.StatusNotFound, "job_not_found", "job not found")
			}
			if projectAuthorizationReviewMetadataPresent(txJob) {
				return projectAuthorizationApprovedImmutableAPIError()
			}
			txJob.Set("project_authorization_doc", "")
			txJob.Set("project_authorization_doc_hash", "")
			txJob.Set("pa_uploader", "")
			txJob.Set("pa_uploaded", "")
			txJob.Set("pa_reviewer", "")
			txJob.Set("pa_reviewed", "")
			txJob.Set("pa_rejector", "")
			txJob.Set("pa_rejected", "")
			txJob.Set("pa_rejection_reason", "")
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

func createGetMissingProjectAuthorizationHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil || e.Auth.Id == "" {
			return projectAuthorizationRouteError(e, projectAuthorizationAPIError(http.StatusUnauthorized, "unauthorized", "unauthorized"))
		}

		priority := normalizeProjectAuthorizationMissingPriority(e.Request.URL.Query().Get("priority"))
		page, limit := projectAuthorizationMissingPagination(e.Request.URL.Query().Get("page"), e.Request.URL.Query().Get("limit"))
		params, err := projectAuthorizationMissingParams(app, e.Auth)
		if err != nil {
			return e.InternalServerError("failed to check project authorization permissions", err)
		}
		params["priority"] = priority

		counts, err := projectAuthorizationMissingPriorityCounts(app, params)
		if err != nil {
			return e.InternalServerError("failed to count missing project authorization jobs", err)
		}
		pendingReviewCount := 0
		hasAccounting, err := utilities.HasClaim(app, e.Auth, "accounting")
		if err != nil {
			return e.InternalServerError("failed to check accounting claim", err)
		}
		if hasAccounting {
			pendingReviewCount, err = countProjectAuthorizationPendingReview(app)
			if err != nil {
				return e.InternalServerError("failed to count project authorization queue", err)
			}
		}
		rejectedCount, err := countProjectAuthorizationRejectedForAuth(app, e.Auth)
		if err != nil {
			return e.InternalServerError("failed to count rejected project authorizations", err)
		}
		total := projectAuthorizationMissingTotalForPriority(counts, priority)
		totalPages := projectAuthorizationTotalPages(total, limit)
		if totalPages == 0 {
			page = 1
		} else if page > totalPages {
			page = totalPages
		}
		params["limit"] = limit
		params["offset"] = (page - 1) * limit

		rows := []projectAuthorizationMissingRow{}
		if err := app.DB().NewQuery(projectAuthorizationMissingBaseQuery + `
			SELECT
			  id,
			  number,
			  description,
			  client_id,
			  client_name,
			  manager_id,
			  manager_name,
			  branch_id,
			  branch_code,
			  branch_name,
			  status,
			  project_award_date,
			  updated,
			  project_authorization_doc,
			  project_authorization_doc_hash,
			  project_authorization_state,
			  time_entry_count,
			  purchase_order_count,
			  active_purchase_order_count,
			  expense_count,
			  latest_activity_date,
			  priority,
			  can_upload
			FROM classified
			WHERE {:priority} = 'all' OR priority = {:priority}
			ORDER BY
			  CASE priority
			    WHEN 'in_use' THEN 0
			    WHEN 'recent' THEN 1
			    ELSE 2
			  END,
			  CASE WHEN priority = 'in_use' THEN latest_activity_date ELSE '' END DESC,
			  CASE
			    WHEN priority = 'recent' AND COALESCE(project_award_date, '') != '' THEN project_award_date
			    WHEN priority = 'recent' THEN substr(COALESCE(updated, ''), 1, 10)
			    ELSE ''
			  END DESC,
			  CASE
			    WHEN priority = 'dormant' AND COALESCE(project_award_date, '') != '' THEN project_award_date
			    WHEN priority = 'dormant' THEN substr(COALESCE(updated, ''), 1, 10)
			    ELSE ''
			  END ASC,
			  number ASC
			LIMIT {:limit} OFFSET {:offset}
		`).Bind(params).All(&rows); err != nil {
			return e.InternalServerError("failed to load missing project authorization jobs", err)
		}

		return e.JSON(http.StatusOK, projectAuthorizationMissingResponse{
			Items:              rows,
			Counts:             counts,
			Page:               page,
			Limit:              limit,
			Total:              total,
			TotalPages:         totalPages,
			Priority:           priority,
			PendingReviewCount: pendingReviewCount,
			RejectedCount:      rejectedCount,
		})
	}
}

func createGetRejectedProjectAuthorizationHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		params, err := projectAuthorizationMissingParams(app, e.Auth)
		if err != nil {
			return e.InternalServerError("failed to check project authorization visibility", err)
		}
		rows := []projectAuthorizationRejectedRow{}
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
			  j.project_authorization_doc_hash,
			  j.pa_rejector AS pa_rejector_id,
			  TRIM(COALESCE(rp.given_name, '') || ' ' || COALESCE(rp.surname, '')) AS pa_rejector_name,
			  j.pa_rejected,
			  j.pa_rejection_reason,
			  CASE
			    WHEN {:canUploadAll} = 1
			      OR j.manager = {:uid}
			      OR j.alternate_manager = {:uid}
			      OR b.manager = {:uid}
			    THEN 1
			    ELSE 0
			  END AS can_upload
			FROM jobs j
			LEFT JOIN clients c ON c.id = j.client
			LEFT JOIN profiles mp ON mp.uid = j.manager
			LEFT JOIN profiles rp ON rp.uid = j.pa_rejector
			LEFT JOIN branches b ON b.id = j.branch
			WHERE j.status = 'Active'
			  AND j.number NOT LIKE 'P%'
			  AND j.project_authorization_doc != ''
			  AND j.project_authorization_doc_hash != ''
			  AND (
			    j.pa_rejected != ''
			    OR j.pa_rejector != ''
			    OR j.pa_rejection_reason != ''
			  )
			  AND (
			    {:canSeeAll} = 1
			    OR j.manager = {:uid}
			    OR j.alternate_manager = {:uid}
			    OR b.manager = {:uid}
			  )
			ORDER BY j.pa_rejected DESC, j.number
		`).Bind(params).All(&rows); err != nil {
			return e.InternalServerError("failed to load rejected project authorizations", err)
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
			  j.client_po,
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
			  AND j.pa_rejected = ''
			  AND j.pa_rejector = ''
			  AND j.pa_rejection_reason = ''
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

func normalizeProjectAuthorizationMissingPriority(raw string) string {
	switch strings.TrimSpace(raw) {
	case projectAuthorizationMissingPriorityAll,
		projectAuthorizationMissingPriorityInUse,
		projectAuthorizationMissingPriorityRecent,
		projectAuthorizationMissingPriorityDormant:
		return strings.TrimSpace(raw)
	default:
		return projectAuthorizationMissingPriorityInUse
	}
}

func projectAuthorizationMissingPagination(pageRaw string, limitRaw string) (int, int) {
	page := 1
	if parsed, err := strconv.Atoi(strings.TrimSpace(pageRaw)); err == nil && parsed > 0 {
		page = parsed
	}
	limit := projectAuthorizationMissingDefaultLimit
	if parsed, err := strconv.Atoi(strings.TrimSpace(limitRaw)); err == nil && parsed > 0 {
		limit = parsed
	}
	if limit > projectAuthorizationMissingMaxLimit {
		limit = projectAuthorizationMissingMaxLimit
	}
	return page, limit
}

func projectAuthorizationTotalPages(total int, limit int) int {
	if total == 0 || limit <= 0 {
		return 0
	}
	return (total + limit - 1) / limit
}

func projectAuthorizationMissingParams(app core.App, auth *core.Record) (dbx.Params, error) {
	hasAccounting, err := utilities.HasClaim(app, auth, "accounting")
	if err != nil {
		return nil, err
	}
	hasJobClaim, err := utilities.HasClaim(app, auth, "job")
	if err != nil {
		return nil, err
	}
	canSeeAll := 0
	if hasAccounting || hasJobClaim {
		canSeeAll = 1
	}
	canUploadAll := 0
	if hasJobClaim {
		canUploadAll = 1
	}
	uid := ""
	if auth != nil {
		uid = auth.Id
	}
	return dbx.Params{
		"uid":          uid,
		"canSeeAll":    canSeeAll,
		"canUploadAll": canUploadAll,
	}, nil
}

func projectAuthorizationMissingPriorityCounts(app core.App, params dbx.Params) (map[string]int, error) {
	counts := map[string]int{
		projectAuthorizationMissingPriorityInUse:   0,
		projectAuthorizationMissingPriorityRecent:  0,
		projectAuthorizationMissingPriorityDormant: 0,
		projectAuthorizationMissingPriorityAll:     0,
	}
	rows := []projectAuthorizationMissingCountRow{}
	if err := app.DB().NewQuery(projectAuthorizationMissingBaseQuery + `
		SELECT priority, COUNT(*) AS count
		FROM classified
		GROUP BY priority
	`).Bind(params).All(&rows); err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.Priority] = row.Count
		counts[projectAuthorizationMissingPriorityAll] += row.Count
	}
	return counts, nil
}

func projectAuthorizationMissingTotalForPriority(counts map[string]int, priority string) int {
	if priority == projectAuthorizationMissingPriorityAll {
		return counts[projectAuthorizationMissingPriorityAll]
	}
	return counts[priority]
}

func countProjectAuthorizationPendingReview(app core.App) (int, error) {
	return countNavRows(app, `
		SELECT COUNT(*)
		FROM jobs j
		WHERE j.status = 'Active'
		  AND j.number NOT LIKE 'P%'
		  AND j.project_authorization_doc != ''
		  AND j.project_authorization_doc_hash != ''
		  AND j.pa_reviewed = ''
		  AND j.pa_reviewer = ''
		  AND j.pa_rejected = ''
		  AND j.pa_rejector = ''
		  AND j.pa_rejection_reason = ''
	`, dbx.Params{})
}

func countProjectAuthorizationMissingForAuth(app core.App, auth *core.Record) (int, error) {
	params, err := projectAuthorizationMissingParams(app, auth)
	if err != nil {
		return 0, err
	}
	return countNavRows(app, `
		SELECT COUNT(*)
		FROM jobs j
		LEFT JOIN branches b ON b.id = j.branch
		WHERE j.status = 'Active'
		  AND j.number NOT LIKE 'P%'
		  AND (
		    COALESCE(j.project_authorization_doc, '') = ''
		    OR COALESCE(j.project_authorization_doc_hash, '') = ''
		  )
		  AND (
		    {:canSeeAll} = 1
		    OR j.manager = {:uid}
		    OR j.alternate_manager = {:uid}
		    OR b.manager = {:uid}
		  )
	`, params)
}

func countProjectAuthorizationRejectedForAuth(app core.App, auth *core.Record) (int, error) {
	params, err := projectAuthorizationMissingParams(app, auth)
	if err != nil {
		return 0, err
	}
	return countNavRows(app, `
		SELECT COUNT(*)
		FROM jobs j
		LEFT JOIN branches b ON b.id = j.branch
		WHERE j.status = 'Active'
		  AND j.number NOT LIKE 'P%'
		  AND j.project_authorization_doc != ''
		  AND j.project_authorization_doc_hash != ''
		  AND (
		    j.pa_rejected != ''
		    OR j.pa_rejector != ''
		    OR j.pa_rejection_reason != ''
		  )
		  AND (
		    {:canSeeAll} = 1
		    OR j.manager = {:uid}
		    OR j.alternate_manager = {:uid}
		    OR b.manager = {:uid}
		  )
	`, params)
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
			if projectAuthorizationReviewMetadataPresent(job) {
				return projectAuthorizationAPIError(http.StatusConflict, "project_authorization_already_approved", "This project authorization document has already been approved.")
			}
			if projectAuthorizationRejectionMetadataPresent(job) {
				return projectAuthorizationAPIError(http.StatusConflict, "project_authorization_rejected", "This project authorization document has been rejected. Please upload a replacement before approving.")
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

func createRejectProjectAuthorizationHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAccountingClaim(app, e.Auth); err != nil {
			return projectAuthorizationRouteError(e, err)
		}
		var req projectAuthorizationRejectRequest
		if err := e.BindBody(&req); err != nil {
			return e.BadRequestError("invalid rejection request", err)
		}
		submittedHash := strings.TrimSpace(req.ProjectAuthorizationDocHash)
		reason := strings.TrimSpace(req.RejectionReason)
		if len(reason) < 4 {
			return projectAuthorizationRouteError(e, projectAuthorizationFieldAPIError(
				http.StatusBadRequest,
				"rejection_reason_too_short",
				"rejection reason must be at least 4 characters long",
				"rejection_reason",
			))
		}

		var rejected *core.Record
		err := app.RunInTransaction(func(txApp core.App) error {
			job, err := txApp.FindRecordById("jobs", e.Request.PathValue("id"))
			if err != nil {
				return projectAuthorizationAPIError(http.StatusNotFound, "job_not_found", "job not found")
			}
			if job.GetString("project_authorization_doc") == "" {
				return projectAuthorizationFieldAPIError(http.StatusBadRequest, "required", "project authorization document is required before rejection", "project_authorization_doc")
			}
			currentHash := strings.TrimSpace(job.GetString("project_authorization_doc_hash"))
			if currentHash == "" || currentHash != submittedHash {
				return projectAuthorizationAPIError(http.StatusConflict, "project_authorization_doc_changed", "The project authorization document changed after you opened it. Please review the current document before rejecting.")
			}
			if projectAuthorizationReviewMetadataPresent(job) {
				return projectAuthorizationAPIError(http.StatusConflict, "project_authorization_already_approved", "This project authorization document has already been approved.")
			}
			if projectAuthorizationRejectionMetadataPresent(job) {
				return projectAuthorizationAPIError(http.StatusConflict, "project_authorization_already_rejected", "This project authorization document has already been rejected.")
			}
			job.Set("pa_rejected", time.Now().UTC())
			job.Set("pa_rejector", e.Auth.Id)
			job.Set("pa_rejection_reason", reason)
			if err := txApp.SaveWithContext(hooks.WithProjectAuthorizationMutation(e.Request.Context(), hooks.ProjectAuthorizationMutationReject), job); err != nil {
				return err
			}
			rejected = job
			return nil
		})
		if err != nil {
			return projectAuthorizationRouteError(e, err)
		}
		if rejected != nil {
			if notifErr := notifications.QueueProjectAuthorizationRejectedNotifications(app, rejected, e.Auth.Id, reason); notifErr != nil {
				app.Logger().Error(
					"error queueing project authorization rejection notifications",
					"job_id", rejected.Id,
					"error", notifErr,
				)
			}
		}
		return e.JSON(http.StatusOK, rejected)
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
	return requireProjectAuthorizationClaim(app, auth, "accounting", "you do not have permission to approve project authorization documents")
}

func requireAdminClaim(app core.App, auth *core.Record) error {
	return requireProjectAuthorizationClaim(app, auth, "admin", "you do not have permission to revoke project authorization approvals")
}

func requireProjectAuthorizationClaim(app core.App, auth *core.Record, claim string, forbiddenMessage string) error {
	if auth == nil || auth.Id == "" {
		return projectAuthorizationAPIError(http.StatusUnauthorized, "unauthorized", "unauthorized")
	}
	hasClaim, err := utilities.HasClaim(app, auth, claim)
	if err != nil {
		return err
	}
	if !hasClaim {
		return projectAuthorizationAPIError(http.StatusForbidden, "forbidden", forbiddenMessage)
	}
	return nil
}

func projectAuthorizationReviewMetadataPresent(record *core.Record) bool {
	return record != nil && (record.GetString("pa_reviewed") != "" || record.GetString("pa_reviewer") != "")
}

func projectAuthorizationRejectionMetadataPresent(record *core.Record) bool {
	return record != nil && (record.GetString("pa_rejected") != "" || record.GetString("pa_rejector") != "" || record.GetString("pa_rejection_reason") != "")
}

func projectAuthorizationApprovedImmutableAPIError() *projectAuthorizationHTTPError {
	return projectAuthorizationFieldAPIError(
		http.StatusBadRequest,
		"project_authorization_approved_immutable",
		"revoke PA approval before replacing or removing the uploaded document",
		"project_authorization_doc",
	)
}

func projectAuthorizationNotProjectJobAPIError(action string) *projectAuthorizationHTTPError {
	return projectAuthorizationAPIError(
		http.StatusBadRequest,
		"not_project_job",
		"project authorization documents can only be "+action+" project jobs",
	)
}

func duplicateProjectAuthorizationDocAPIError() *projectAuthorizationHTTPError {
	return projectAuthorizationFieldAPIError(
		http.StatusBadRequest,
		"duplicate_file",
		duplicateProjectAuthorizationDocMessage,
		"project_authorization_doc",
	)
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
		return projectAuthorizationRouteError(e, duplicateProjectAuthorizationDocAPIError())
	}
	return e.Error(http.StatusInternalServerError, "project authorization request failed", err)
}
