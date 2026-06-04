package routes

import (
	"errors"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

type projectAuthorizationDocHashAuditResponse struct {
	JobID string `json:"job_id"`
	storedFileHashAuditResponse
}

type projectAuthorizationDocHashReplaceResponse struct {
	JobID string `json:"job_id"`
	storedFileHashReplaceResponse
}

var projectAuthorizationDocHashMessages = storedFileHashMessages{
	EmptyStoragePath: "project authorization document not found",
	OpenFilesystem:   "failed to open filesystem",
	FileNotFound:     "project authorization document file not found",
	HashFailed:       "failed to hash project authorization document",
	TargetChanged:    "project authorization document target changed",
	UpdatedChanged:   "project authorization document changed; rerun audit before replacing",
	UniqueConflict:   "calculated hash already belongs to another project authorization document",
}

func createAuditProjectAuthorizationDocHashHandler(app core.App) func(e *core.RequestEvent) error {
	return createStoredFileHashAuditHandler(app, auditProjectAuthorizationDocHash, projectAuthorizationDocHashRouteError)
}

func createReplaceProjectAuthorizationDocHashHandler(app core.App) func(e *core.RequestEvent) error {
	return createStoredFileHashReplaceHandler(app, replaceProjectAuthorizationDocHash, projectAuthorizationDocHashRouteError)
}

func auditProjectAuthorizationDocHash(app core.App, jobID string) (projectAuthorizationDocHashAuditResponse, error) {
	audit, err := auditStoredFileHash(app, func(app core.App) (storedFileHashTarget, error) {
		return resolveProjectAuthorizationDocHashTarget(app, jobID)
	}, projectAuthorizationDocHashMessages)
	if err != nil {
		return projectAuthorizationDocHashAuditResponse{}, err
	}
	return projectAuthorizationDocHashAuditResponseFromStored(jobID, audit), nil
}

func replaceProjectAuthorizationDocHash(app core.App, jobID string, expectedUpdated string) (projectAuthorizationDocHashReplaceResponse, error) {
	replacement, err := replaceStoredFileHash(app, expectedUpdated, func(app core.App) (storedFileHashTarget, error) {
		return resolveProjectAuthorizationDocHashTarget(app, jobID)
	}, projectAuthorizationDocHashMessages)
	if err != nil {
		return projectAuthorizationDocHashReplaceResponse{}, err
	}
	return projectAuthorizationDocHashReplaceResponseFromStored(jobID, replacement), nil
}

func resolveProjectAuthorizationDocHashTarget(app core.App, jobID string) (storedFileHashTarget, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return storedFileHashTarget{}, &storedFileHashHTTPError{status: http.StatusBadRequest, message: "job id is required"}
	}

	job, err := app.FindRecordById("jobs", jobID)
	if err != nil {
		return storedFileHashTarget{}, &storedFileHashHTTPError{status: http.StatusNotFound, message: "job not found", err: err}
	}

	filename := strings.TrimSpace(job.GetString("project_authorization_doc"))
	if filename == "" {
		return storedFileHashTarget{}, &storedFileHashHTTPError{status: http.StatusNotFound, message: "job has no project authorization document"}
	}
	updated, err := storedFileHashUpdatedString(app, "jobs", job.Id)
	if err != nil {
		return storedFileHashTarget{}, &storedFileHashHTTPError{status: http.StatusInternalServerError, message: "failed to load project authorization document timestamp", err: err}
	}

	return storedFileHashTarget{
		TargetCollection: "jobs",
		TargetID:         job.Id,
		Filename:         filename,
		StoragePath:      job.BaseFilesPath() + "/" + filename,
		StoredHash:       strings.TrimSpace(job.GetString("project_authorization_doc_hash")),
		Updated:          updated,
		HashField:        "project_authorization_doc_hash",
	}, nil
}

func projectAuthorizationDocHashRouteError(e *core.RequestEvent, err error) error {
	var httpErr *storedFileHashHTTPError
	if errors.As(err, &httpErr) {
		return e.Error(httpErr.status, httpErr.message, httpErr.err)
	}
	return projectAuthorizationRouteError(e, err)
}

func projectAuthorizationDocHashAuditResponseFromStored(jobID string, audit storedFileHashAudit) projectAuthorizationDocHashAuditResponse {
	return projectAuthorizationDocHashAuditResponse{
		JobID:                       jobID,
		storedFileHashAuditResponse: storedFileHashAuditResponseFromStored(audit),
	}
}

func projectAuthorizationDocHashReplaceResponseFromStored(jobID string, replacement storedFileHashReplace) projectAuthorizationDocHashReplaceResponse {
	return projectAuthorizationDocHashReplaceResponse{
		JobID:                         jobID,
		storedFileHashReplaceResponse: storedFileHashReplaceResponseFromStored(replacement),
	}
}
