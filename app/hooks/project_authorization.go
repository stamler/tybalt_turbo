package hooks

import (
	"context"
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

	projectAuthorizationDocField      = "project_authorization_doc"
	projectAuthorizationDocHashField  = "project_authorization_doc_hash"
	projectAuthorizationUploaderField = "pa_uploader"
	projectAuthorizationUploadedField = "pa_uploaded"
	projectAuthorizationReviewerField = "pa_reviewer"
	projectAuthorizationReviewedField = "pa_reviewed"
	projectAuthorizationRejectorField = "pa_rejector"
	projectAuthorizationRejectedField = "pa_rejected"
	projectAuthorizationReasonField   = "pa_rejection_reason"
)

type ProjectAuthorizationMutation string

const (
	ProjectAuthorizationMutationUpload  ProjectAuthorizationMutation = "upload"
	ProjectAuthorizationMutationDelete  ProjectAuthorizationMutation = "delete"
	ProjectAuthorizationMutationApprove ProjectAuthorizationMutation = "approve"
	ProjectAuthorizationMutationReject  ProjectAuthorizationMutation = "reject"
	ProjectAuthorizationMutationRevoke  ProjectAuthorizationMutation = "revoke"
)

type projectAuthorizationMutationContextKey struct{}

var (
	projectAuthorizationFields = []string{
		projectAuthorizationDocField,
		projectAuthorizationDocHashField,
		projectAuthorizationUploaderField,
		projectAuthorizationUploadedField,
		projectAuthorizationReviewerField,
		projectAuthorizationReviewedField,
		projectAuthorizationRejectorField,
		projectAuthorizationRejectedField,
		projectAuthorizationReasonField,
	}
	projectAuthorizationApprovalFields = []string{
		projectAuthorizationDocField,
		projectAuthorizationDocHashField,
		projectAuthorizationReviewerField,
		projectAuthorizationReviewedField,
	}
	projectAuthorizationReviewFields = []string{
		projectAuthorizationReviewerField,
		projectAuthorizationReviewedField,
	}
	projectAuthorizationRejectionFields = []string{
		projectAuthorizationRejectorField,
		projectAuthorizationRejectedField,
		projectAuthorizationReasonField,
	}
)

type ProjectAuthorizationBlockingJob struct {
	ID          string `db:"id" json:"id"`
	Number      string `db:"number" json:"number"`
	Description string `db:"description" json:"description"`
	Manager     string `db:"manager" json:"manager"`
	ManagerName string `db:"manager_name" json:"manager_name"`
}

func WithProjectAuthorizationMutation(ctx context.Context, mutation ProjectAuthorizationMutation) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, projectAuthorizationMutationContextKey{}, mutation)
}

func ProjectAuthorizationMutationFromContext(ctx context.Context) ProjectAuthorizationMutation {
	if ctx == nil {
		return ""
	}
	mutation, _ := ctx.Value(projectAuthorizationMutationContextKey{}).(ProjectAuthorizationMutation)
	return mutation
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
		clearProjectAuthorizationStoredState(record)
		return nil
	}

	original := record.Original()
	approved := projectAuthorizationApprovedFieldsPopulated(original)
	if approved && projectAuthorizationDocumentChanged(record) {
		return projectAuthorizationHookError(
			http.StatusBadRequest,
			"approved project authorization document is immutable",
			projectAuthorizationDocField,
			"project_authorization_approved_immutable",
			"revoke PA approval before replacing or removing the uploaded document",
		)
	}

	if !uploadRoute {
		restoreProjectAuthorizationStoredState(record, original)
		return nil
	}

	attachmentHash, err := CalculateFileFieldHash(e, projectAuthorizationDocField)
	if err != nil {
		return err
	}
	if attachmentHash == "" {
		return projectAuthorizationHookError(
			http.StatusBadRequest,
			"project authorization document upload is required",
			projectAuthorizationDocField,
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
			projectAuthorizationDocField,
			"duplicate_file",
			"this PA document has already been uploaded to another job",
		)
	}

	record.Set(projectAuthorizationDocHashField, attachmentHash)
	clearProjectAuthorizationReviewFields(record)
	clearProjectAuthorizationRejectionFields(record)
	return nil
}

func EnforceProjectAuthorizationSaveInvariant(record *core.Record, mutation ProjectAuthorizationMutation) error {
	if record == nil {
		return nil
	}
	if record.IsNew() {
		if field := firstPopulatedProjectAuthorizationField(record); field != "" {
			return projectAuthorizationNotEditableError(field)
		}
		return nil
	}

	changes := projectAuthorizationFieldChangesFor(record)
	if !changes.any() {
		return nil
	}

	switch mutation {
	case ProjectAuthorizationMutationUpload:
		return validateProjectAuthorizationUploadSave(record)
	case ProjectAuthorizationMutationDelete:
		return validateProjectAuthorizationDeleteSave(record)
	case ProjectAuthorizationMutationApprove:
		return validateProjectAuthorizationApproveSave(record, changes)
	case ProjectAuthorizationMutationReject:
		return validateProjectAuthorizationRejectSave(record, changes)
	case ProjectAuthorizationMutationRevoke:
		return validateProjectAuthorizationRevokeSave(record, changes)
	default:
		return projectAuthorizationNotEditableError(changes.firstField())
	}
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
	for _, field := range projectAuthorizationApprovalFields {
		if strings.TrimSpace(record.GetString(field)) == "" {
			return false
		}
	}
	return true
}

type projectAuthorizationFieldChanges struct {
	doc       bool
	hash      bool
	uploader  bool
	uploaded  bool
	reviewer  bool
	reviewed  bool
	rejector  bool
	rejected  bool
	rejection bool
}

func projectAuthorizationFieldChangesFor(record *core.Record) projectAuthorizationFieldChanges {
	return projectAuthorizationFieldChanges{
		doc:       projectAuthorizationDocumentChanged(record),
		hash:      projectAuthorizationStringChanged(record, projectAuthorizationDocHashField),
		uploader:  projectAuthorizationStringChanged(record, projectAuthorizationUploaderField),
		uploaded:  projectAuthorizationStringChanged(record, projectAuthorizationUploadedField),
		reviewer:  projectAuthorizationStringChanged(record, projectAuthorizationReviewerField),
		reviewed:  projectAuthorizationStringChanged(record, projectAuthorizationReviewedField),
		rejector:  projectAuthorizationStringChanged(record, projectAuthorizationRejectorField),
		rejected:  projectAuthorizationStringChanged(record, projectAuthorizationRejectedField),
		rejection: projectAuthorizationStringChanged(record, projectAuthorizationReasonField),
	}
}

func (changes projectAuthorizationFieldChanges) any() bool {
	return changes.doc || changes.hash || changes.uploader || changes.uploaded || changes.reviewer || changes.reviewed || changes.rejector || changes.rejected || changes.rejection
}

func (changes projectAuthorizationFieldChanges) firstField() string {
	switch {
	case changes.doc:
		return projectAuthorizationDocField
	case changes.hash:
		return projectAuthorizationDocHashField
	case changes.uploader:
		return projectAuthorizationUploaderField
	case changes.uploaded:
		return projectAuthorizationUploadedField
	case changes.reviewer:
		return projectAuthorizationReviewerField
	case changes.reviewed:
		return projectAuthorizationReviewedField
	case changes.rejector:
		return projectAuthorizationRejectorField
	case changes.rejected:
		return projectAuthorizationRejectedField
	case changes.rejection:
		return projectAuthorizationReasonField
	default:
		return projectAuthorizationDocField
	}
}

func validateProjectAuthorizationUploadSave(record *core.Record) error {
	original := record.Original()
	if projectAuthorizationReviewMetadataPresent(original) {
		return projectAuthorizationApprovedImmutableError()
	}
	if len(record.GetUnsavedFiles(projectAuthorizationDocField)) == 0 {
		return projectAuthorizationHookError(
			http.StatusBadRequest,
			"project authorization document upload is required",
			projectAuthorizationDocField,
			"required",
			"upload a signed PA PDF",
		)
	}
	if strings.TrimSpace(record.GetString(projectAuthorizationDocHashField)) == "" {
		return projectAuthorizationHookError(
			http.StatusBadRequest,
			"project authorization document hash is required",
			projectAuthorizationDocHashField,
			"required",
			"project authorization document hash is required",
		)
	}
	if field := firstPopulatedProjectAuthorizationReviewField(record); field != "" {
		return projectAuthorizationNotEditableError(field)
	}
	if field := firstPopulatedProjectAuthorizationRejectionField(record); field != "" {
		return projectAuthorizationNotEditableError(field)
	}
	if strings.TrimSpace(record.GetString(projectAuthorizationUploaderField)) == "" {
		return projectAuthorizationHookError(
			http.StatusBadRequest,
			"project authorization uploader is required",
			projectAuthorizationUploaderField,
			"required",
			"project authorization uploader is required",
		)
	}
	if strings.TrimSpace(record.GetString(projectAuthorizationUploadedField)) == "" {
		return projectAuthorizationHookError(
			http.StatusBadRequest,
			"project authorization upload timestamp is required",
			projectAuthorizationUploadedField,
			"required",
			"project authorization upload timestamp is required",
		)
	}
	return nil
}

func validateProjectAuthorizationDeleteSave(record *core.Record) error {
	if projectAuthorizationReviewMetadataPresent(record.Original()) {
		return projectAuthorizationApprovedImmutableError()
	}
	if field := firstPopulatedProjectAuthorizationField(record); field != "" {
		return projectAuthorizationNotEditableError(field)
	}
	return nil
}

func validateProjectAuthorizationApproveSave(record *core.Record, changes projectAuthorizationFieldChanges) error {
	original := record.Original()
	if changes.doc {
		return projectAuthorizationNotEditableError(projectAuthorizationDocField)
	}
	if changes.hash {
		return projectAuthorizationNotEditableError(projectAuthorizationDocHashField)
	}
	if changes.uploader {
		return projectAuthorizationNotEditableError(projectAuthorizationUploaderField)
	}
	if changes.uploaded {
		return projectAuthorizationNotEditableError(projectAuthorizationUploadedField)
	}
	if changes.rejector {
		return projectAuthorizationNotEditableError(projectAuthorizationRejectorField)
	}
	if changes.rejected {
		return projectAuthorizationNotEditableError(projectAuthorizationRejectedField)
	}
	if changes.rejection {
		return projectAuthorizationNotEditableError(projectAuthorizationReasonField)
	}
	if projectAuthorizationReviewMetadataPresent(original) {
		return projectAuthorizationHookError(
			http.StatusConflict,
			"project authorization document has already been approved",
			projectAuthorizationReviewedField,
			"project_authorization_already_approved",
			"this project authorization document has already been approved",
		)
	}
	if projectAuthorizationRejectionMetadataPresent(original) {
		return projectAuthorizationHookError(
			http.StatusConflict,
			"project authorization document has been rejected",
			projectAuthorizationRejectedField,
			"project_authorization_rejected",
			"replace the rejected project authorization document before approving",
		)
	}
	if strings.TrimSpace(original.GetString(projectAuthorizationDocField)) == "" {
		return projectAuthorizationHookError(
			http.StatusBadRequest,
			"project authorization document is required before approval",
			projectAuthorizationDocField,
			"required",
			"project authorization document is required before approval",
		)
	}
	if strings.TrimSpace(original.GetString(projectAuthorizationDocHashField)) == "" {
		return projectAuthorizationHookError(
			http.StatusConflict,
			"project authorization document hash is missing",
			projectAuthorizationDocHashField,
			"project_authorization_doc_changed",
			"review the current project authorization document before approving",
		)
	}
	if strings.TrimSpace(record.GetString(projectAuthorizationReviewerField)) == "" {
		return projectAuthorizationHookError(
			http.StatusBadRequest,
			"project authorization reviewer is required",
			projectAuthorizationReviewerField,
			"required",
			"project authorization reviewer is required",
		)
	}
	if strings.TrimSpace(record.GetString(projectAuthorizationReviewedField)) == "" {
		return projectAuthorizationHookError(
			http.StatusBadRequest,
			"project authorization reviewed timestamp is required",
			projectAuthorizationReviewedField,
			"required",
			"project authorization reviewed timestamp is required",
		)
	}
	return nil
}

func validateProjectAuthorizationRejectSave(record *core.Record, changes projectAuthorizationFieldChanges) error {
	original := record.Original()
	if changes.doc {
		return projectAuthorizationNotEditableError(projectAuthorizationDocField)
	}
	if changes.hash {
		return projectAuthorizationNotEditableError(projectAuthorizationDocHashField)
	}
	if changes.uploader {
		return projectAuthorizationNotEditableError(projectAuthorizationUploaderField)
	}
	if changes.uploaded {
		return projectAuthorizationNotEditableError(projectAuthorizationUploadedField)
	}
	if changes.reviewer {
		return projectAuthorizationNotEditableError(projectAuthorizationReviewerField)
	}
	if changes.reviewed {
		return projectAuthorizationNotEditableError(projectAuthorizationReviewedField)
	}
	if projectAuthorizationReviewMetadataPresent(original) {
		return projectAuthorizationHookError(
			http.StatusConflict,
			"project authorization document has already been approved",
			projectAuthorizationReviewedField,
			"project_authorization_already_approved",
			"approved project authorization documents cannot be rejected",
		)
	}
	if projectAuthorizationRejectionMetadataPresent(original) {
		return projectAuthorizationHookError(
			http.StatusConflict,
			"project authorization document has already been rejected",
			projectAuthorizationRejectedField,
			"project_authorization_already_rejected",
			"replace the rejected project authorization document before rejecting again",
		)
	}
	if strings.TrimSpace(original.GetString(projectAuthorizationDocField)) == "" {
		return projectAuthorizationHookError(
			http.StatusBadRequest,
			"project authorization document is required before rejection",
			projectAuthorizationDocField,
			"required",
			"project authorization document is required before rejection",
		)
	}
	if strings.TrimSpace(original.GetString(projectAuthorizationDocHashField)) == "" {
		return projectAuthorizationHookError(
			http.StatusConflict,
			"project authorization document hash is missing",
			projectAuthorizationDocHashField,
			"project_authorization_doc_changed",
			"review the current project authorization document before rejecting",
		)
	}
	if strings.TrimSpace(record.GetString(projectAuthorizationRejectorField)) == "" {
		return projectAuthorizationHookError(
			http.StatusBadRequest,
			"project authorization rejector is required",
			projectAuthorizationRejectorField,
			"required",
			"project authorization rejector is required",
		)
	}
	if strings.TrimSpace(record.GetString(projectAuthorizationRejectedField)) == "" {
		return projectAuthorizationHookError(
			http.StatusBadRequest,
			"project authorization rejection timestamp is required",
			projectAuthorizationRejectedField,
			"required",
			"project authorization rejection timestamp is required",
		)
	}
	if strings.TrimSpace(record.GetString(projectAuthorizationReasonField)) == "" {
		return projectAuthorizationHookError(
			http.StatusBadRequest,
			"project authorization rejection reason is required",
			projectAuthorizationReasonField,
			"required",
			"project authorization rejection reason is required",
		)
	}
	return nil
}

func validateProjectAuthorizationRevokeSave(record *core.Record, changes projectAuthorizationFieldChanges) error {
	if changes.doc {
		return projectAuthorizationNotEditableError(projectAuthorizationDocField)
	}
	if changes.hash {
		return projectAuthorizationNotEditableError(projectAuthorizationDocHashField)
	}
	if changes.uploader {
		return projectAuthorizationNotEditableError(projectAuthorizationUploaderField)
	}
	if changes.uploaded {
		return projectAuthorizationNotEditableError(projectAuthorizationUploadedField)
	}
	if changes.rejector {
		return projectAuthorizationNotEditableError(projectAuthorizationRejectorField)
	}
	if changes.rejected {
		return projectAuthorizationNotEditableError(projectAuthorizationRejectedField)
	}
	if changes.rejection {
		return projectAuthorizationNotEditableError(projectAuthorizationReasonField)
	}
	if field := firstPopulatedProjectAuthorizationReviewField(record); field != "" {
		return projectAuthorizationNotEditableError(field)
	}
	return nil
}

func projectAuthorizationReviewMetadataPresent(record *core.Record) bool {
	return firstPopulatedProjectAuthorizationReviewField(record) != ""
}

func projectAuthorizationRejectionMetadataPresent(record *core.Record) bool {
	return firstPopulatedProjectAuthorizationRejectionField(record) != ""
}

func firstPopulatedProjectAuthorizationField(record *core.Record) string {
	return firstPopulatedProjectAuthorizationFieldIn(record, projectAuthorizationFields)
}

func firstPopulatedProjectAuthorizationReviewField(record *core.Record) string {
	return firstPopulatedProjectAuthorizationFieldIn(record, projectAuthorizationReviewFields)
}

func firstPopulatedProjectAuthorizationRejectionField(record *core.Record) string {
	return firstPopulatedProjectAuthorizationFieldIn(record, projectAuthorizationRejectionFields)
}

func firstPopulatedProjectAuthorizationFieldIn(record *core.Record, fields []string) string {
	for _, field := range fields {
		if projectAuthorizationFieldPopulated(record, field) {
			return field
		}
	}
	return ""
}

func projectAuthorizationFieldPopulated(record *core.Record, field string) bool {
	return record != nil && (strings.TrimSpace(record.GetString(field)) != "" || len(record.GetUnsavedFiles(field)) > 0)
}

func projectAuthorizationStringChanged(record *core.Record, field string) bool {
	return strings.TrimSpace(record.GetString(field)) != strings.TrimSpace(record.Original().GetString(field))
}

func projectAuthorizationDocumentChanged(record *core.Record) bool {
	return projectAuthorizationStringChanged(record, projectAuthorizationDocField) ||
		len(record.GetUnsavedFiles(projectAuthorizationDocField)) > 0
}

func clearProjectAuthorizationStoredState(record *core.Record) {
	record.Set(projectAuthorizationDocHashField, "")
	clearProjectAuthorizationUploadFields(record)
	clearProjectAuthorizationReviewFields(record)
	clearProjectAuthorizationRejectionFields(record)
}

func clearProjectAuthorizationUploadFields(record *core.Record) {
	record.Set(projectAuthorizationUploaderField, "")
	record.Set(projectAuthorizationUploadedField, "")
}

func clearProjectAuthorizationReviewFields(record *core.Record) {
	record.Set(projectAuthorizationReviewerField, "")
	record.Set(projectAuthorizationReviewedField, "")
}

func clearProjectAuthorizationRejectionFields(record *core.Record) {
	record.Set(projectAuthorizationRejectorField, "")
	record.Set(projectAuthorizationRejectedField, "")
	record.Set(projectAuthorizationReasonField, "")
}

func restoreProjectAuthorizationStoredState(record *core.Record, original *core.Record) {
	for _, field := range projectAuthorizationFields[1:] {
		record.Set(field, original.GetString(field))
	}
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
	for _, field := range projectAuthorizationFields {
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

func projectAuthorizationApprovedImmutableError() *errs.HookError {
	return projectAuthorizationHookError(
		http.StatusBadRequest,
		"approved project authorization document is immutable",
		projectAuthorizationDocField,
		"project_authorization_approved_immutable",
		"revoke PA approval before replacing or removing the uploaded document",
	)
}

func projectAuthorizationNotEditableError(field string) *errs.HookError {
	return projectAuthorizationHookError(
		http.StatusBadRequest,
		"project authorization fields are server-owned",
		field,
		"not_editable",
		"project authorization fields must be changed through the dedicated PA endpoints",
	)
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
