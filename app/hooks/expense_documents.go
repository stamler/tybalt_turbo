package hooks

import (
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
		// `attachment` is a virtual expense-write input. It deliberately keeps
		// the old field name at the HTTP/form boundary because callers are
		// attaching a receipt to an expense, not managing document records by
		// hand. Storage, identity, and duplicate protection now live on
		// expense_documents.
		//
		// While the legacy columns still exist, PocketBase can otherwise persist
		// the uploaded file back onto expenses. Clear both old fields before the
		// expense save so the only durable link is attachment_document. When the
		// columns are removed, these Set calls become harmless custom-key cleanup
		// and can go away in the schema-removal pass.
		expenseRecord.Set("attachment", "")
		expenseRecord.Set("attachment_hash", "")
	}
}

func clearExpenseDocumentForAttachmentlessType(expenseRecord *core.Record) {
	if expensePaymentTypeSkipsAttachment(expenseRecord.GetString("payment_type")) {
		expenseRecord.Set("attachment_document", "")
		expenseRecord.Set("attachment", "")
		expenseRecord.Set("attachment_hash", "")
	}
}

func expensePaymentTypeSkipsAttachment(paymentType string) bool {
	switch paymentType {
	case "Allowance", "Mileage", "PersonalReimbursement":
		return true
	default:
		return false
	}
}

func documentIDFromUploadedExpenseAttachment(app core.App, e *core.RecordRequestEvent, attachmentHash string, hasBookKeeperClaim bool, poBacked bool) (string, error) {
	existingDocument, err := findExpenseDocumentByHash(app, attachmentHash)
	if err != nil {
		return "", err
	}
	if existingDocument != nil {
		documentInUse, err := expenseDocumentReferencedByAnotherExpense(app, existingDocument.Id, e.Record.Id)
		if err != nil {
			return "", err
		}
		if !documentInUse {
			return existingDocument.Id, nil
		}
		if !hasBookKeeperClaim {
			return "", duplicateAttachmentError()
		}
		if !poBacked {
			return "", duplicateAttachmentPermissionError("purchase_order", "required", "purchase order is required to reuse an attachment")
		}
		return existingDocument.Id, nil
	}

	// Read from the virtual expense "attachment" input. For the custom expense
	// API this may be a raw "attachment:unsaved" value rather than a real
	// expenses collection file field, which keeps the write path intact after
	// the legacy file column is removed.
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

	return "", duplicateAttachmentPermissionError("source_expense", "missing_attachment", "source expense has no document-backed attachment")
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

func findExpenseDocumentByHash(app core.App, attachmentHash string) (*core.Record, error) {
	if strings.TrimSpace(attachmentHash) == "" {
		return nil, nil
	}
	var id string
	err := app.DB().NewQuery(`
		SELECT id
		FROM expense_documents
		WHERE attachment_hash = {:hash}
		LIMIT 1
	`).Bind(dbx.Params{
		"hash": attachmentHash,
	}).Row(&id)
	if err != nil {
		return nil, nil
	}
	return app.FindRecordById(constants.ExpenseDocumentsCollectionName, id)
}

func expenseDocumentReferencedByAnotherExpense(app core.App, documentID string, currentExpenseID string) (bool, error) {
	if strings.TrimSpace(documentID) == "" {
		return false, nil
	}

	var count int
	err := app.DB().NewQuery(`
		SELECT COUNT(*)
		FROM expenses
		WHERE attachment_document = {:document_id}
		  AND id != {:current_expense_id}
	`).Bind(dbx.Params{
		"document_id":        documentID,
		"current_expense_id": currentExpenseID,
	}).Row(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
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
