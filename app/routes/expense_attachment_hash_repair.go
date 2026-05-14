package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"

	"tybalt/constants"
	"tybalt/utilities"
)

const (
	expenseHashTargetExpenses         = "expenses"
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

type expenseAttachmentHashReplaceResponse struct {
	expenseAttachmentHashAuditResponse
	PreviousHash string `json:"previous_hash"`
	NewHash      string `json:"new_hash"`
	Replaced     bool   `json:"replaced"`
	Noop         bool   `json:"noop"`
}

type expenseAttachmentHashHTTPError struct {
	status  int
	message string
	err     error
}

func (e *expenseAttachmentHashHTTPError) Error() string {
	if e.err == nil {
		return e.message
	}
	return fmt.Sprintf("%s: %v", e.message, e.err)
}

func createAuditExpenseAttachmentHashHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminForExpenseAttachmentHashRepair(app, e); err != nil {
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
		if err := requireAdminForExpenseAttachmentHashRepair(app, e); err != nil {
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

func requireAdminForExpenseAttachmentHashRepair(app core.App, e *core.RequestEvent) error {
	if e.Auth == nil {
		return e.Error(http.StatusUnauthorized, "unauthorized", nil)
	}
	hasAdmin, err := utilities.HasClaim(app, e.Auth, "admin")
	if err != nil {
		return e.Error(http.StatusInternalServerError, "error checking admin claim", err)
	}
	if !hasAdmin {
		return e.Error(http.StatusForbidden, "admin claim required", nil)
	}
	return nil
}

func auditExpenseAttachmentHash(app core.App, expenseID string) (expenseAttachmentHashAuditResponse, error) {
	target, err := resolveExpenseAttachmentHashTarget(app, expenseID)
	if err != nil {
		return expenseAttachmentHashAuditResponse{}, err
	}
	calculatedHash, err := calculateStoredFileSHA256(app, target.StoragePath)
	if err != nil {
		return expenseAttachmentHashAuditResponse{}, err
	}
	return expenseAttachmentHashAuditResponse{
		expenseAttachmentHashTarget: target,
		CalculatedHash:              calculatedHash,
		Matches:                     strings.EqualFold(strings.TrimSpace(target.StoredHash), calculatedHash),
	}, nil
}

func replaceExpenseAttachmentHash(app core.App, expenseID string, expectedUpdated string) (expenseAttachmentHashReplaceResponse, error) {
	audit, err := auditExpenseAttachmentHash(app, expenseID)
	if err != nil {
		return expenseAttachmentHashReplaceResponse{}, err
	}

	response := expenseAttachmentHashReplaceResponse{
		expenseAttachmentHashAuditResponse: audit,
		PreviousHash:                       audit.StoredHash,
		NewHash:                            audit.CalculatedHash,
	}

	var updated string
	err = app.RunInTransaction(func(txApp core.App) error {
		freshTarget, err := resolveExpenseAttachmentHashTarget(txApp, expenseID)
		if err != nil {
			return err
		}
		if freshTarget.TargetCollection != audit.TargetCollection || freshTarget.TargetID != audit.TargetID || freshTarget.StoragePath != audit.StoragePath {
			return &expenseAttachmentHashHTTPError{status: http.StatusConflict, message: "expense attachment target changed"}
		}
		if freshTarget.Updated != expectedUpdated {
			return &expenseAttachmentHashHTTPError{status: http.StatusConflict, message: "attachment hash owner changed; rerun audit before replacing"}
		}
		if strings.EqualFold(strings.TrimSpace(freshTarget.StoredHash), audit.CalculatedHash) {
			updated = freshTarget.Updated
			response.StoredHash = freshTarget.StoredHash
			response.PreviousHash = freshTarget.StoredHash
			response.Noop = true
			return nil
		}

		newUpdated := types.NowDateTime()
		table := expenseAttachmentHashTable(freshTarget.TargetCollection)
		result, err := txApp.DB().NewQuery(fmt.Sprintf(`
			UPDATE %s
			SET attachment_hash = {:hash}, updated = {:updated}
			WHERE id = {:id} AND updated = {:expected_updated}
		`, table)).Bind(dbx.Params{
			"hash":             audit.CalculatedHash,
			"updated":          newUpdated,
			"id":               freshTarget.TargetID,
			"expected_updated": expectedUpdated,
		}).Execute()
		if err != nil {
			if isUniqueConstraintError(err) {
				return &expenseAttachmentHashHTTPError{status: http.StatusConflict, message: "calculated hash already belongs to another attachment", err: err}
			}
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return &expenseAttachmentHashHTTPError{status: http.StatusConflict, message: "attachment hash owner changed; rerun audit before replacing"}
		}

		updated = newUpdated.String()
		response.Replaced = true
		return nil
	})
	if err != nil {
		return expenseAttachmentHashReplaceResponse{}, err
	}

	response.Updated = updated
	response.StoredHash = audit.CalculatedHash
	response.Matches = true
	return response, nil
}

func resolveExpenseAttachmentHashTarget(app core.App, expenseID string) (expenseAttachmentHashTarget, error) {
	expenseID = strings.TrimSpace(expenseID)
	if expenseID == "" {
		return expenseAttachmentHashTarget{}, &expenseAttachmentHashHTTPError{status: http.StatusBadRequest, message: "expense id is required"}
	}

	expense, err := app.FindRecordById("expenses", expenseID)
	if err != nil {
		return expenseAttachmentHashTarget{}, &expenseAttachmentHashHTTPError{status: http.StatusNotFound, message: "expense not found", err: err}
	}

	documentID := strings.TrimSpace(expense.GetString("attachment_document"))
	if documentID != "" {
		document, err := app.FindRecordById(constants.ExpenseDocumentsCollectionName, documentID)
		if err != nil {
			return expenseAttachmentHashTarget{}, &expenseAttachmentHashHTTPError{status: http.StatusNotFound, message: "expense attachment document not found", err: err}
		}
		return expenseAttachmentHashTargetFromRecord(app, expenseID, expenseHashTargetExpenseDocuments, document)
	}

	return expenseAttachmentHashTargetFromRecord(app, expenseID, expenseHashTargetExpenses, expense)
}

func expenseAttachmentHashTargetFromRecord(app core.App, expenseID string, collectionName string, record *core.Record) (expenseAttachmentHashTarget, error) {
	filename := strings.TrimSpace(record.GetString("attachment"))
	storagePath := ""
	if filename != "" {
		storagePath = record.BaseFilesPath() + "/" + filename
	}
	updated, err := expenseAttachmentHashUpdatedString(app, collectionName, record.Id)
	if err != nil {
		return expenseAttachmentHashTarget{}, &expenseAttachmentHashHTTPError{status: http.StatusInternalServerError, message: "failed to load attachment hash owner timestamp", err: err}
	}
	return expenseAttachmentHashTarget{
		ExpenseID:        expenseID,
		TargetCollection: collectionName,
		TargetID:         record.Id,
		Filename:         filename,
		StoragePath:      storagePath,
		StoredHash:       strings.TrimSpace(record.GetString("attachment_hash")),
		Updated:          updated,
	}, nil
}

func expenseAttachmentHashUpdatedString(app core.App, collectionName string, recordID string) (string, error) {
	var updated string
	if err := app.DB().NewQuery("SELECT updated FROM " + expenseAttachmentHashTable(collectionName) + " WHERE id = {:id}").
		Bind(dbx.Params{"id": recordID}).Row(&updated); err != nil {
		return "", err
	}
	return updated, nil
}

func calculateStoredFileSHA256(app core.App, storagePath string) (string, error) {
	if strings.TrimSpace(storagePath) == "" {
		return "", &expenseAttachmentHashHTTPError{status: http.StatusNotFound, message: "expense attachment not found"}
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return "", &expenseAttachmentHashHTTPError{status: http.StatusInternalServerError, message: "failed to open filesystem", err: err}
	}
	defer fsys.Close()

	reader, err := fsys.GetReader(storagePath)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, filesystem.ErrNotFound) {
			status = http.StatusNotFound
		}
		return "", &expenseAttachmentHashHTTPError{status: status, message: "expense attachment file not found", err: err}
	}
	defer reader.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", &expenseAttachmentHashHTTPError{status: http.StatusInternalServerError, message: "failed to hash expense attachment", err: err}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func expenseAttachmentHashTable(collectionName string) string {
	switch collectionName {
	case expenseHashTargetExpenseDocuments:
		return constants.ExpenseDocumentsCollectionName
	default:
		return "expenses"
	}
}

func expenseAttachmentHashRouteError(e *core.RequestEvent, err error) error {
	var httpErr *expenseAttachmentHashHTTPError
	if errors.As(err, &httpErr) {
		return e.Error(httpErr.status, httpErr.message, httpErr.err)
	}
	return e.Error(http.StatusInternalServerError, "expense attachment hash repair failed", err)
}
