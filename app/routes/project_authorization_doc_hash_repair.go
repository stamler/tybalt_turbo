package routes

import (
	"errors"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

type projectAuthorizationDocHashTarget struct {
	JobID            string `json:"job_id"`
	TargetCollection string `json:"target_collection"`
	TargetID         string `json:"target_id"`
	Filename         string `json:"filename"`
	StoragePath      string `json:"storage_path"`
	StoredHash       string `json:"stored_hash"`
	Updated          string `json:"updated"`
}

type projectAuthorizationDocHashAuditResponse struct {
	projectAuthorizationDocHashTarget
	CalculatedHash string `json:"calculated_hash"`
	Matches        bool   `json:"matches"`
}

type projectAuthorizationDocHashReplaceRequest struct {
	Updated string `json:"updated"`
}

type projectAuthorizationDocHashReplaceResponse struct {
	projectAuthorizationDocHashAuditResponse
	PreviousHash string `json:"previous_hash"`
	NewHash      string `json:"new_hash"`
	Replaced     bool   `json:"replaced"`
	Noop         bool   `json:"noop"`
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
	return func(e *core.RequestEvent) error {
		if err := requireAdminForStoredFileHashRepair(app, e); err != nil {
			return err
		}

		response, err := auditProjectAuthorizationDocHash(app, e.Request.PathValue("id"))
		if err != nil {
			return projectAuthorizationDocHashRouteError(e, err)
		}
		return e.JSON(http.StatusOK, response)
	}
}

func createReplaceProjectAuthorizationDocHashHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminForStoredFileHashRepair(app, e); err != nil {
			return err
		}

		var req projectAuthorizationDocHashReplaceRequest
		if err := e.BindBody(&req); err != nil {
			return e.Error(http.StatusBadRequest, "invalid request body", err)
		}
		if strings.TrimSpace(req.Updated) == "" {
			return e.Error(http.StatusBadRequest, "updated is required", nil)
		}

		response, err := replaceProjectAuthorizationDocHash(app, e.Request.PathValue("id"), strings.TrimSpace(req.Updated))
		if err != nil {
			return projectAuthorizationDocHashRouteError(e, err)
		}
		return e.JSON(http.StatusOK, response)
	}
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
		projectAuthorizationDocHashTarget: projectAuthorizationDocHashTargetFromStored(jobID, audit.Target),
		CalculatedHash:                    audit.CalculatedHash,
		Matches:                           audit.Matches,
	}
}

func projectAuthorizationDocHashReplaceResponseFromStored(jobID string, replacement storedFileHashReplace) projectAuthorizationDocHashReplaceResponse {
	return projectAuthorizationDocHashReplaceResponse{
		projectAuthorizationDocHashAuditResponse: projectAuthorizationDocHashAuditResponseFromStored(jobID, replacement.Audit),
		PreviousHash:                             replacement.PreviousHash,
		NewHash:                                  replacement.NewHash,
		Replaced:                                 replacement.Replaced,
		Noop:                                     replacement.Noop,
	}
}

func projectAuthorizationDocHashTargetFromStored(jobID string, target storedFileHashTarget) projectAuthorizationDocHashTarget {
	return projectAuthorizationDocHashTarget{
		JobID:            jobID,
		TargetCollection: target.TargetCollection,
		TargetID:         target.TargetID,
		Filename:         target.Filename,
		StoragePath:      target.StoragePath,
		StoredHash:       target.StoredHash,
		Updated:          target.Updated,
	}
}
