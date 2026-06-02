package hooks

import (
	"net/http"
	"strings"
	"tybalt/errs"
	"tybalt/utilities"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	ProjectAuthorizationNotApprovedCode    = "project_authorization_not_approved"
	ProjectAuthorizationNotApprovedMessage = "This project is not approved for time, purchase orders, or expenses yet. Please speak with the project's manager."
)

type ProjectAuthorizationBlockingJob struct {
	ID          string `db:"id" json:"id"`
	Number      string `db:"number" json:"number"`
	Description string `db:"description" json:"description"`
	Manager     string `db:"manager" json:"manager"`
	ManagerName string `db:"manager_name" json:"manager_name"`
}

func ProcessJobProjectAuthorizationFields(app core.App, e *core.RecordRequestEvent) error {
	record := e.Record
	if record == nil {
		return nil
	}

	uploadRoute := isProjectAuthorizationUploadRoute(e)
	if !uploadRoute {
		if err := rejectGenericProjectAuthorizationFieldMutation(e); err != nil {
			return err
		}
	}

	if record.IsNew() {
		record.Set("project_authorization_doc_hash", "")
		record.Set("pa_reviewer", "")
		record.Set("pa_reviewed", "")
		return nil
	}

	original := record.Original()
	approved := projectAuthorizationApprovedFieldsPopulated(original)
	docChanged := strings.TrimSpace(record.GetString("project_authorization_doc")) != strings.TrimSpace(original.GetString("project_authorization_doc")) ||
		len(record.GetUnsavedFiles("project_authorization_doc")) > 0

	if approved && docChanged {
		return projectAuthorizationHookError(
			http.StatusBadRequest,
			"approved project authorization document is immutable",
			"project_authorization_doc",
			"project_authorization_approved_immutable",
			"revoke PA approval before replacing or removing the uploaded document",
		)
	}

	if !uploadRoute {
		record.Set("project_authorization_doc_hash", original.GetString("project_authorization_doc_hash"))
		record.Set("pa_reviewer", original.GetString("pa_reviewer"))
		record.Set("pa_reviewed", original.GetString("pa_reviewed"))
		return nil
	}

	attachmentHash, err := CalculateFileFieldHash(e, "project_authorization_doc")
	if err != nil {
		return err
	}
	if attachmentHash == "" {
		return projectAuthorizationHookError(
			http.StatusBadRequest,
			"project authorization document upload is required",
			"project_authorization_doc",
			"required",
			"upload a signed PA PDF",
		)
	}

	existingJob, _ := app.FindFirstRecordByFilter("jobs", "project_authorization_doc_hash = {:hash} && id != {:id}", dbx.Params{
		"hash": attachmentHash,
		"id":   record.Id,
	})
	if existingJob != nil {
		return projectAuthorizationHookError(
			http.StatusBadRequest,
			"duplicate project authorization document detected",
			"project_authorization_doc",
			"duplicate_file",
			"this PA document has already been uploaded to another job",
		)
	}

	record.Set("project_authorization_doc_hash", attachmentHash)
	record.Set("pa_reviewer", "")
	record.Set("pa_reviewed", "")
	return nil
}

func CanUploadProjectAuthorizationDocument(app core.App, job *core.Record, auth *core.Record) (bool, error) {
	if job == nil || auth == nil || auth.Id == "" {
		return false, nil
	}
	hasJobClaim, err := utilities.HasClaim(app, auth, "job")
	if err != nil {
		return false, err
	}
	if hasJobClaim {
		return true, nil
	}
	if auth.Id == strings.TrimSpace(job.GetString("manager")) || auth.Id == strings.TrimSpace(job.GetString("alternate_manager")) {
		return true, nil
	}
	branchID := strings.TrimSpace(job.GetString("branch"))
	if branchID == "" {
		return false, nil
	}
	branch, err := app.FindRecordById("branches", branchID)
	if err != nil || branch == nil {
		return false, nil
	}
	return auth.Id == strings.TrimSpace(branch.GetString("manager")), nil
}

func EnsureProjectAuthorizationApprovedForJob(app core.App, jobID string, fieldName string) error {
	if !utilities.IsProjectAuthorizationEnforced(app) || strings.TrimSpace(jobID) == "" {
		return nil
	}
	job, err := app.FindRecordById("jobs", jobID)
	if err != nil || job == nil {
		return nil
	}
	if typeFromNumber(job.GetString("number")) == jobTypeProposal {
		return nil
	}
	if projectAuthorizationApprovedFieldsPopulated(job) {
		return nil
	}
	if strings.TrimSpace(fieldName) == "" {
		fieldName = "job"
	}
	return validation.Errors{
		fieldName: validation.NewError(ProjectAuthorizationNotApprovedCode, ProjectAuthorizationNotApprovedMessage),
	}.Filter()
}

func UnapprovedProjectAuthorizationJobsForTimeEntries(app core.App, userID string, weekEnding string) ([]ProjectAuthorizationBlockingJob, error) {
	if !utilities.IsProjectAuthorizationEnforced(app) {
		return nil, nil
	}

	rows := []ProjectAuthorizationBlockingJob{}
	if err := app.DB().NewQuery(`
		SELECT DISTINCT
		  j.id,
		  j.number,
		  j.description,
		  j.manager,
		  TRIM(COALESCE(p.given_name, '') || ' ' || COALESCE(p.surname, '')) AS manager_name
		FROM time_entries te
		JOIN jobs j ON j.id = te.job
		LEFT JOIN profiles p ON p.uid = j.manager
		WHERE te.uid = {:uid}
		  AND te.week_ending = {:weekEnding}
		  AND te.job != ''
		  AND j.number NOT LIKE 'P%'
		  AND (
		    j.project_authorization_doc = '' OR
		    j.project_authorization_doc_hash = '' OR
		    j.pa_reviewed = '' OR
		    j.pa_reviewer = ''
		  )
		ORDER BY j.number
	`).Bind(dbx.Params{
		"uid":        userID,
		"weekEnding": weekEnding,
	}).All(&rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func projectAuthorizationApprovedFieldsPopulated(record *core.Record) bool {
	if record == nil {
		return false
	}
	return strings.TrimSpace(record.GetString("project_authorization_doc")) != "" &&
		strings.TrimSpace(record.GetString("project_authorization_doc_hash")) != "" &&
		strings.TrimSpace(record.GetString("pa_reviewed")) != "" &&
		strings.TrimSpace(record.GetString("pa_reviewer")) != ""
}

func isProjectAuthorizationUploadRoute(e *core.RecordRequestEvent) bool {
	if e == nil || e.Request == nil {
		return false
	}
	path := e.Request.URL.Path
	return strings.HasPrefix(path, "/api/jobs/") && strings.HasSuffix(path, "/project_authorization_doc")
}

func rejectGenericProjectAuthorizationFieldMutation(e *core.RecordRequestEvent) error {
	info, err := e.RequestInfo()
	if err != nil {
		return err
	}
	protectedFields := []string{
		"project_authorization_doc",
		"project_authorization_doc_hash",
		"pa_reviewer",
		"pa_reviewed",
	}
	for _, field := range protectedFields {
		if _, ok := info.Body[field]; ok || len(e.Record.GetUnsavedFiles(field)) > 0 {
			return projectAuthorizationHookError(
				http.StatusBadRequest,
				"project authorization fields are server-owned",
				field,
				"not_editable",
				"project authorization fields must be changed through the dedicated PA endpoints",
			)
		}
	}
	return nil
}

func projectAuthorizationHookError(status int, message string, field string, code string, fieldMessage string) *errs.HookError {
	return &errs.HookError{
		Status:  status,
		Message: message,
		Data: map[string]errs.CodeError{
			field: {Code: code, Message: fieldMessage},
		},
	}
}
