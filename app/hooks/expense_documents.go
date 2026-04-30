package hooks

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"tybalt/constants"
	"tybalt/errs"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/core/validators"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"
)

var expenseDocumentMimeTypes = []string{
	"application/pdf",
	"image/jpeg",
	"image/png",
	"image/heic",
}

const expenseDocumentUploadCleanupKey = "expense_document_upload_cleanup_paths"

func registerExpenseDocumentUploadForCleanup(e *core.RecordRequestEvent, path string) {
	if e == nil || strings.TrimSpace(path) == "" {
		return
	}

	var paths []string
	if raw := e.Get(expenseDocumentUploadCleanupKey); raw != nil {
		if existing, ok := raw.([]string); ok {
			paths = append(paths, existing...)
		}
	}
	paths = append(paths, path)
	e.Set(expenseDocumentUploadCleanupKey, paths)
}

func CleanupExpenseDocumentUploads(app core.App, e *core.RecordRequestEvent) {
	if app == nil || e == nil {
		return
	}

	raw := e.Get(expenseDocumentUploadCleanupKey)
	paths, ok := raw.([]string)
	if !ok || len(paths) == 0 {
		return
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		app.Logger().Error("failed to initialize filesystem for expense document cleanup", "error", err)
		return
	}
	defer fsys.Close()

	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		if err := fsys.Delete(path); err != nil {
			app.Logger().Warn("failed to cleanup rolled back expense document upload", "path", path, "error", err)
		}
	}
	e.Set(expenseDocumentUploadCleanupKey, []string{})
}

func resolveExpenseDocumentForSave(app core.App, e *core.RecordRequestEvent, attachmentHash string, hasBookKeeperClaim bool, poBacked bool) (string, error) {
	expenseRecord := e.Record
	sourceExpenseID := strings.TrimSpace(expenseRecord.GetString("source_expense"))
	if sourceExpenseID != "" {
		if !hasBookKeeperClaim {
			return "", duplicateAttachmentPermissionError("source_expense", "book_keeper_required", "book_keeper claim required to reuse an attachment")
		}
		if !poBacked {
			return "", duplicateAttachmentPermissionError("purchase_order", "required", "purchase order is required to reuse an attachment")
		}
		return documentIDFromSourceExpense(app, e, sourceExpenseID)
	}

	if attachmentHash != "" {
		return documentIDFromUploadedExpenseAttachment(app, e, attachmentHash, hasBookKeeperClaim, poBacked)
	}

	if expenseRecord.GetString("attachment") != "" && expenseRecord.GetString("attachment_document") == "" {
		return migrateLegacyExpenseAttachmentToDocument(app, e, expenseRecord, e.Auth.Id, false)
	}

	if expenseRecord.IsNew() {
		return "", nil
	}

	original := expenseRecord.Original()
	if original == nil {
		return "", nil
	}

	// The attachment_document relation is server-managed. If no new file or
	// source expense was provided, preserve the original relation rather than
	// allowing clients to point at arbitrary documents.
	return original.GetString("attachment_document"), nil
}

func applyResolvedExpenseDocument(expenseRecord *core.Record, documentID string) {
	expenseRecord.Set("attachment_document", documentID)
	if documentID != "" {
		expenseRecord.Set("attachment", "")
		expenseRecord.Set("attachment_hash", "")
	}
}

func clearExpenseDocumentForAttachmentlessType(expenseRecord *core.Record) {
	switch expenseRecord.GetString("payment_type") {
	case "Allowance", "Mileage", "PersonalReimbursement":
		expenseRecord.Set("attachment_document", "")
		expenseRecord.Set("attachment_hash", "")
	}
}

func documentIDFromUploadedExpenseAttachment(app core.App, e *core.RecordRequestEvent, attachmentHash string, hasBookKeeperClaim bool, poBacked bool) (string, error) {
	existingDocument, err := findExpenseDocumentByHash(app, attachmentHash)
	if err != nil {
		return "", err
	}
	if existingDocument != nil {
		if !hasBookKeeperClaim {
			return "", duplicateAttachmentError()
		}
		if !poBacked {
			return "", duplicateAttachmentPermissionError("purchase_order", "required", "purchase order is required to reuse an attachment")
		}
		return existingDocument.Id, nil
	}

	legacyDuplicate, err := legacyExpenseAttachmentHashExists(app, attachmentHash, e.Record.Id)
	if err != nil {
		return "", err
	}
	if legacyDuplicate && !hasBookKeeperClaim {
		return "", duplicateAttachmentError()
	}
	if legacyDuplicate && !poBacked {
		return "", duplicateAttachmentPermissionError("purchase_order", "required", "purchase order is required to reuse an attachment")
	}

	files := e.Record.GetUnsavedFiles("attachment")
	if len(files) != 1 {
		return "", &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "hook error processing attachment",
			Data: map[string]errs.CodeError{
				"attachment": {Code: "invalid_file_count", Message: "expected one attachment file"},
			},
		}
	}

	document, err := createExpenseDocument(app, e, files[0], attachmentHash, e.Auth.Id)
	if err != nil {
		return "", err
	}
	return document.Id, nil
}

func documentIDFromSourceExpense(app core.App, e *core.RecordRequestEvent, sourceExpenseID string) (string, error) {
	sourceExpense, err := findVisibleExpenseForAttachmentReuse(app, e.Auth, sourceExpenseID)
	if err != nil {
		return "", err
	}

	if documentID := strings.TrimSpace(sourceExpense.GetString("attachment_document")); documentID != "" {
		return documentID, nil
	}

	if sourceExpense.GetString("attachment") == "" {
		return "", duplicateAttachmentPermissionError("source_expense", "missing_attachment", "source expense has no attachment")
	}

	return migrateLegacyExpenseAttachmentToDocument(app, e, sourceExpense, e.Auth.Id, true)
}

func findVisibleExpenseForAttachmentReuse(app core.App, auth *core.Record, expenseID string) (*core.Record, error) {
	if auth == nil || auth.Id == "" {
		return nil, duplicateAttachmentPermissionError("source_expense", "authentication_required", "authentication is required")
	}

	hasCommit, err := utilities.HasClaim(app, auth, "commit")
	if err != nil {
		return nil, err
	}
	hasReport, err := utilities.HasClaim(app, auth, "report")
	if err != nil {
		return nil, err
	}
	hasAdmin, err := utilities.HasClaim(app, auth, "admin")
	if err != nil {
		return nil, err
	}

	filter := `
		id = {:id} && (
			uid = {:auth} ||
			creator = {:auth} ||
			(approver = {:auth} && submitted = true) ||
			({:has_commit} = 1 && approved != '') ||
			({:has_report} = 1 && committed != '') ||
			({:has_admin} = 1 && committed != '')
		)
	`
	record, err := app.FindFirstRecordByFilter("expenses", filter, dbx.Params{
		"id":         expenseID,
		"auth":       auth.Id,
		"has_commit": boolToIntHook(hasCommit),
		"has_report": boolToIntHook(hasReport),
		"has_admin":  boolToIntHook(hasAdmin),
	})
	if err != nil || record == nil {
		return nil, duplicateAttachmentPermissionError("source_expense", "not_found", "source expense not found or not visible")
	}

	return record, nil
}

func migrateLegacyExpenseAttachmentToDocument(app core.App, e *core.RecordRequestEvent, expenseRecord *core.Record, fallbackUploaderID string, saveExpenseLink bool) (string, error) {
	filename := strings.TrimSpace(expenseRecord.GetString("attachment"))
	if filename == "" {
		return "", nil
	}

	attachmentHash := strings.TrimSpace(expenseRecord.GetString("attachment_hash"))
	var fileBytes []byte
	var err error
	if attachmentHash == "" {
		fileBytes, err = readExpenseFileBytes(app, expenseRecord.BaseFilesPath()+"/"+filename)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(fileBytes)
		attachmentHash = hex.EncodeToString(sum[:])
	}

	existingDocument, err := findExpenseDocumentByHash(app, attachmentHash)
	if err != nil {
		return "", err
	}
	if existingDocument != nil {
		expenseRecord.Set("attachment_document", existingDocument.Id)
		if saveExpenseLink {
			if err := app.Save(expenseRecord); err != nil {
				return "", err
			}
		}
		return existingDocument.Id, nil
	}

	if fileBytes == nil {
		fileBytes, err = readExpenseFileBytes(app, expenseRecord.BaseFilesPath()+"/"+filename)
		if err != nil {
			return "", err
		}
	}
	file, err := filesystem.NewFileFromBytes(fileBytes, filename)
	if err != nil {
		return "", err
	}

	uploaderID := strings.TrimSpace(expenseRecord.GetString("creator"))
	if uploaderID == "" {
		uploaderID = strings.TrimSpace(expenseRecord.GetString("uid"))
	}
	if uploaderID == "" {
		uploaderID = fallbackUploaderID
	}

	document, err := createExpenseDocument(app, e, file, attachmentHash, uploaderID)
	if err != nil {
		return "", err
	}

	expenseRecord.Set("attachment_document", document.Id)
	if saveExpenseLink {
		if err := app.Save(expenseRecord); err != nil {
			return "", err
		}
	}
	return document.Id, nil
}

func findExpenseDocumentByHash(app core.App, attachmentHash string) (*core.Record, error) {
	if strings.TrimSpace(attachmentHash) == "" {
		return nil, nil
	}
	record, err := app.FindFirstRecordByFilter(constants.ExpenseDocumentsCollectionName, "attachment_hash = {:hash}", dbx.Params{
		"hash": attachmentHash,
	})
	if err != nil {
		return nil, nil
	}
	return record, nil
}

func legacyExpenseAttachmentHashExists(app core.App, attachmentHash string, currentExpenseID string) (bool, error) {
	if strings.TrimSpace(attachmentHash) == "" {
		return false, nil
	}
	record, err := app.FindFirstRecordByFilter("expenses", "attachment_hash = {:hash} && id != {:id}", dbx.Params{
		"hash": attachmentHash,
		"id":   currentExpenseID,
	})
	if err != nil {
		return false, nil
	}
	return record != nil, nil
}

func createExpenseDocument(app core.App, e *core.RecordRequestEvent, file *filesystem.File, attachmentHash string, uploadedBy string) (*core.Record, error) {
	if err := validateExpenseDocumentFile(file); err != nil {
		return nil, err
	}

	collection, err := app.FindCollectionByNameOrId(constants.ExpenseDocumentsCollectionName)
	if err != nil {
		return nil, err
	}
	document := core.NewRecord(collection)
	document.Id = core.GenerateDefaultRandomId()

	fsys, err := app.NewFilesystem()
	if err != nil {
		return nil, err
	}
	defer fsys.Close()

	storagePath := document.BaseFilesPath() + "/" + file.Name
	if err := fsys.UploadFile(file, storagePath); err != nil {
		return nil, err
	}

	now := types.NowDateTime()
	_, err = app.DB().Insert(constants.ExpenseDocumentsCollectionName, dbx.Params{
		"id":              document.Id,
		"attachment":      file.Name,
		"attachment_hash": attachmentHash,
		"uploaded_by":     uploadedBy,
		"created":         now,
		"updated":         now,
	}).Execute()
	if err != nil {
		_ = fsys.Delete(storagePath)
		return nil, err
	}
	registerExpenseDocumentUploadForCleanup(e, storagePath)

	document.SetRaw("attachment", file.Name)
	document.SetRaw("attachment_hash", attachmentHash)
	document.SetRaw("uploaded_by", uploadedBy)
	document.SetRaw("created", now)
	document.SetRaw("updated", now)
	if err := document.PostScan(); err != nil {
		return nil, err
	}

	return document, nil
}

func validateExpenseDocumentFile(file *filesystem.File) error {
	if file == nil {
		return duplicateAttachmentPermissionError("attachment", "required", "attachment is required")
	}

	if err := validators.UploadedFileSize(5242880)(file); err != nil {
		return attachmentFileValidationError("invalid_file_size", err)
	}
	if err := validators.UploadedFileMimeType(expenseDocumentMimeTypes)(file); err != nil {
		return attachmentFileValidationError("invalid_mime_type", err)
	}

	return nil
}

func readExpenseFileBytes(app core.App, path string) ([]byte, error) {
	fsys, err := app.NewFilesystem()
	if err != nil {
		return nil, err
	}
	defer fsys.Close()

	reader, err := fsys.GetReader(path)
	if err != nil {
		return nil, &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "failed to read legacy expense attachment",
			Data: map[string]errs.CodeError{
				"attachment": {Code: "legacy_attachment_read_failed", Message: fmt.Sprintf("failed to read legacy attachment: %v", err)},
			},
		}
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

func duplicateAttachmentError() error {
	return &errs.HookError{
		Status:  http.StatusBadRequest,
		Message: "duplicate attachment detected",
		Data: map[string]errs.CodeError{
			"attachment": {
				Code:    "duplicate_file",
				Message: "This file has already been uploaded to another expense",
			},
		},
	}
}

func duplicateAttachmentPermissionError(field string, code string, message string) error {
	return &errs.HookError{
		Status:  http.StatusBadRequest,
		Message: "attachment reuse is not allowed",
		Data: map[string]errs.CodeError{
			field: {Code: code, Message: message},
		},
	}
}

func attachmentFileValidationError(code string, err error) error {
	return &errs.HookError{
		Status:  http.StatusBadRequest,
		Message: "invalid attachment",
		Data: map[string]errs.CodeError{
			"attachment": {Code: code, Message: err.Error()},
		},
	}
}

func boolToIntHook(v bool) int {
	if v {
		return 1
	}
	return 0
}
