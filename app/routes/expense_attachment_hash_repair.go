package routes

import (
	"errors"
	"net/http"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"tybalt/constants"
)

const (
	expenseHashTargetExpenseDocuments = constants.ExpenseDocumentsCollectionName
)

type expenseAttachmentHashTarget struct {
	ExpenseID        string `json:"expense_id"`
	TargetCollection string `json:"target_collection"`
	TargetID         string `json:"target_id"`
	Filename         string `json:"filename"`
	StoragePath      string `json:"storage_path"`
	StoredHash       string `json:"stored_hash"`
	Updated          string `json:"updated"`
}

type expenseAttachmentHashAuditResponse struct {
	expenseAttachmentHashTarget
	CalculatedHash string `json:"calculated_hash"`
	Matches        bool   `json:"matches"`
}

type expenseAttachmentHashReplaceRequest struct {
	Updated string `json:"updated"`
}

type expenseAttachmentMissingMarkRequest struct {
	Updated string `json:"updated"`
	Reason  string `json:"reason"`
}

type expenseAttachmentHashReplaceResponse struct {
	expenseAttachmentHashAuditResponse
	PreviousHash string `json:"previous_hash"`
	NewHash      string `json:"new_hash"`
	Replaced     bool   `json:"replaced"`
	Noop         bool   `json:"noop"`
}

type expenseAttachmentMissingMarkResponse struct {
	ExpenseID               string `json:"expense_id"`
	Updated                 string `json:"updated"`
	AttachmentMissingReason string `json:"attachment_missing_reason"`
	PreviousDocumentID      string `json:"previous_attachment_document"`
	Marked                  bool   `json:"marked"`
	Noop                    bool   `json:"noop"`
}

type expenseAttachmentHashHTTPError = storedFileHashHTTPError

var expenseAttachmentHashMessages = storedFileHashMessages{
	EmptyStoragePath: "expense attachment not found",
	OpenFilesystem:   "failed to open filesystem",
	FileNotFound:     "expense attachment file not found",
	HashFailed:       "failed to hash expense attachment",
	TargetChanged:    "expense attachment target changed",
	UpdatedChanged:   "attachment hash owner changed; rerun audit before replacing",
	UniqueConflict:   "calculated hash already belongs to another attachment",
}

func createAuditExpenseAttachmentHashHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminForStoredFileHashRepair(app, e); err != nil {
			return err
		}

		response, err := auditExpenseAttachmentHash(app, e.Request.PathValue("id"))
		if err != nil {
			return expenseAttachmentHashRouteError(e, err)
		}
		return e.JSON(http.StatusOK, response)
	}
}

func createReplaceExpenseAttachmentHashHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminForStoredFileHashRepair(app, e); err != nil {
			return err
		}

		var req expenseAttachmentHashReplaceRequest
		if err := e.BindBody(&req); err != nil {
			return e.Error(http.StatusBadRequest, "invalid request body", err)
		}
		if strings.TrimSpace(req.Updated) == "" {
			return e.Error(http.StatusBadRequest, "updated is required", nil)
		}

		response, err := replaceExpenseAttachmentHash(app, e.Request.PathValue("id"), strings.TrimSpace(req.Updated))
		if err != nil {
			return expenseAttachmentHashRouteError(e, err)
		}
		return e.JSON(http.StatusOK, response)
	}
}

func createMarkExpenseAttachmentMissingHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminForStoredFileHashRepair(app, e); err != nil {
			return err
		}

		var req expenseAttachmentMissingMarkRequest
		if err := e.BindBody(&req); err != nil {
			return e.Error(http.StatusBadRequest, "invalid request body", err)
		}
		if strings.TrimSpace(req.Updated) == "" {
			return e.Error(http.StatusBadRequest, "updated is required", nil)
		}
		if strings.TrimSpace(req.Reason) == "" {
			return e.Error(http.StatusBadRequest, "attachment missing reason is required", nil)
		}

		response, err := markExpenseAttachmentMissing(app, e.Request.PathValue("id"), strings.TrimSpace(req.Updated), strings.TrimSpace(req.Reason))
		if err != nil {
			return expenseAttachmentHashRouteError(e, err)
		}
		return e.JSON(http.StatusOK, response)
	}
}

func auditExpenseAttachmentHash(app core.App, expenseID string) (expenseAttachmentHashAuditResponse, error) {
	audit, err := auditStoredFileHash(app, func(app core.App) (storedFileHashTarget, error) {
		return resolveExpenseAttachmentHashTarget(app, expenseID)
	}, expenseAttachmentHashMessages)
	if err != nil {
		return expenseAttachmentHashAuditResponse{}, err
	}
	return expenseAttachmentHashAuditResponseFromStored(expenseID, audit), nil
}

func replaceExpenseAttachmentHash(app core.App, expenseID string, expectedUpdated string) (expenseAttachmentHashReplaceResponse, error) {
	replacement, err := replaceStoredFileHash(app, expectedUpdated, func(app core.App) (storedFileHashTarget, error) {
		return resolveExpenseAttachmentHashTarget(app, expenseID)
	}, expenseAttachmentHashMessages)
	if err != nil {
		return expenseAttachmentHashReplaceResponse{}, err
	}
	return expenseAttachmentHashReplaceResponseFromStored(expenseID, replacement), nil
}

func markExpenseAttachmentMissing(app core.App, expenseID string, expectedUpdated string, reason string) (expenseAttachmentMissingMarkResponse, error) {
	expenseID = strings.TrimSpace(expenseID)
	if expenseID == "" {
		return expenseAttachmentMissingMarkResponse{}, &expenseAttachmentHashHTTPError{status: http.StatusBadRequest, message: "expense id is required"}
	}

	response := expenseAttachmentMissingMarkResponse{
		ExpenseID:               expenseID,
		AttachmentMissingReason: reason,
	}

	var updated string
	err := app.RunInTransaction(func(txApp core.App) error {
		expense, err := txApp.FindRecordById("expenses", expenseID)
		if err != nil {
			return &expenseAttachmentHashHTTPError{status: http.StatusNotFound, message: "expense not found", err: err}
		}

		currentUpdated, err := storedFileHashUpdatedString(txApp, "expenses", expenseID)
		if err != nil {
			return &expenseAttachmentHashHTTPError{status: http.StatusInternalServerError, message: "failed to load expense timestamp", err: err}
		}
		if currentUpdated != expectedUpdated {
			return &expenseAttachmentHashHTTPError{status: http.StatusConflict, message: "expense changed; refresh before marking attachment missing"}
		}

		response.PreviousDocumentID = strings.TrimSpace(expense.GetString("attachment_document"))
		currentReason := strings.TrimSpace(expense.GetString("attachment_missing_reason"))
		if response.PreviousDocumentID == "" && currentReason != "" {
			if currentReason == reason {
				updated = currentUpdated
				response.Noop = true
				return nil
			}
			return &expenseAttachmentHashHTTPError{status: http.StatusConflict, message: "attachment is already marked missing; reason cannot be changed"}
		}

		newUpdated := types.NowDateTime()
		result, err := txApp.DB().NewQuery(`
			UPDATE expenses
			SET attachment_document = '',
			    attachment_missing_reason = {:reason},
			    updated = {:updated}
			WHERE id = {:id} AND updated = {:expected_updated}
		`).Bind(dbx.Params{
			"reason":           reason,
			"updated":          newUpdated,
			"id":               expenseID,
			"expected_updated": expectedUpdated,
		}).Execute()
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return &expenseAttachmentHashHTTPError{status: http.StatusConflict, message: "expense changed; refresh before marking attachment missing"}
		}

		updated = newUpdated.String()
		response.Marked = true
		return nil
	})
	if err != nil {
		return expenseAttachmentMissingMarkResponse{}, err
	}

	response.Updated = updated
	return response, nil
}

func resolveExpenseAttachmentHashTarget(app core.App, expenseID string) (storedFileHashTarget, error) {
	expenseID = strings.TrimSpace(expenseID)
	if expenseID == "" {
		return storedFileHashTarget{}, &expenseAttachmentHashHTTPError{status: http.StatusBadRequest, message: "expense id is required"}
	}

	expense, err := app.FindRecordById("expenses", expenseID)
	if err != nil {
		return storedFileHashTarget{}, &expenseAttachmentHashHTTPError{status: http.StatusNotFound, message: "expense not found", err: err}
	}

	documentID := strings.TrimSpace(expense.GetString("attachment_document"))
	if documentID != "" {
		document, err := app.FindRecordById(constants.ExpenseDocumentsCollectionName, documentID)
		if err != nil {
			return storedFileHashTarget{}, &expenseAttachmentHashHTTPError{status: http.StatusNotFound, message: "expense attachment document not found", err: err}
		}
		return expenseAttachmentHashTargetFromRecord(app, expenseHashTargetExpenseDocuments, document)
	}

	return storedFileHashTarget{}, &expenseAttachmentHashHTTPError{status: http.StatusNotFound, message: "expense has no document-backed attachment"}
}

func expenseAttachmentHashTargetFromRecord(app core.App, collectionName string, record *core.Record) (storedFileHashTarget, error) {
	filename := strings.TrimSpace(record.GetString("attachment"))
	storagePath := ""
	if filename != "" {
		storagePath = record.BaseFilesPath() + "/" + filename
	}
	updated, err := storedFileHashUpdatedString(app, collectionName, record.Id)
	if err != nil {
		return storedFileHashTarget{}, &expenseAttachmentHashHTTPError{status: http.StatusInternalServerError, message: "failed to load attachment hash owner timestamp", err: err}
	}
	return storedFileHashTarget{
		TargetCollection: collectionName,
		TargetID:         record.Id,
		Filename:         filename,
		StoragePath:      storagePath,
		StoredHash:       strings.TrimSpace(record.GetString("attachment_hash")),
		Updated:          updated,
		HashField:        "attachment_hash",
	}, nil
}

func expenseAttachmentHashRouteError(e *core.RequestEvent, err error) error {
	var httpErr *storedFileHashHTTPError
	if errors.As(err, &httpErr) {
		return e.Error(httpErr.status, httpErr.message, httpErr.err)
	}
	return e.Error(http.StatusInternalServerError, "expense attachment hash repair failed", err)
}

func expenseAttachmentHashAuditResponseFromStored(expenseID string, audit storedFileHashAudit) expenseAttachmentHashAuditResponse {
	return expenseAttachmentHashAuditResponse{
		expenseAttachmentHashTarget: expenseAttachmentHashTargetFromStored(expenseID, audit.Target),
		CalculatedHash:              audit.CalculatedHash,
		Matches:                     audit.Matches,
	}
}

func expenseAttachmentHashReplaceResponseFromStored(expenseID string, replacement storedFileHashReplace) expenseAttachmentHashReplaceResponse {
	return expenseAttachmentHashReplaceResponse{
		expenseAttachmentHashAuditResponse: expenseAttachmentHashAuditResponseFromStored(expenseID, replacement.Audit),
		PreviousHash:                       replacement.PreviousHash,
		NewHash:                            replacement.NewHash,
		Replaced:                           replacement.Replaced,
		Noop:                               replacement.Noop,
	}
}

func expenseAttachmentHashTargetFromStored(expenseID string, target storedFileHashTarget) expenseAttachmentHashTarget {
	return expenseAttachmentHashTarget{
		ExpenseID:        expenseID,
		TargetCollection: target.TargetCollection,
		TargetID:         target.TargetID,
		Filename:         target.Filename,
		StoragePath:      target.StoragePath,
		StoredHash:       target.StoredHash,
		Updated:          target.Updated,
	}
}
