package routes

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"tybalt/errs"
	"tybalt/hooks"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

var expenseWriteAllowedFields = map[string]struct{}{
	"uid":              {},
	"date":             {},
	"division":         {},
	"description":      {},
	"total":            {},
	"payment_type":     {},
	"attachment":       {},
	"job":              {},
	"category":         {},
	"kind":             {},
	"allowance_types":  {},
	"distance":         {},
	"cc_last_4_digits": {},
	"currency":         {},
	"settled_total":    {},
	"purchase_order":   {},
	"vendor":           {},
	"source_expense":   {},
}

func createCreateExpenseHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil || e.Auth.Id == "" {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		collection, err := app.FindCollectionByNameOrId("expenses")
		if err != nil {
			return e.InternalServerError("failed to load expenses collection", err)
		}

		record := core.NewRecord(collection)
		return saveExpenseFromRequest(app, e, record)
	}
}

func createUpdateExpenseHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil || e.Auth.Id == "" {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		id := e.Request.PathValue("id")
		if id == "" {
			return e.NotFoundError("missing expense id", nil)
		}

		record, err := app.FindRecordById("expenses", id)
		if err != nil {
			return e.NotFoundError("expense not found", err)
		}

		return saveExpenseFromRequest(app, e, record)
	}
}

func saveExpenseFromRequest(app core.App, e *core.RequestEvent, record *core.Record) error {
	requestInfo, err := e.RequestInfo()
	if err != nil {
		return e.BadRequestError("failed to read request", err)
	}

	data, err := expenseRecordDataFromRequest(e, record)
	if err != nil {
		return e.BadRequestError("failed to read expense data", err)
	}
	if err := rejectDisallowedExpenseWriteFields(data); err != nil {
		return expenseWriteError(e, err)
	}
	requestInfo.Body = data

	form := forms.NewRecordUpsert(app, record)
	form.SetContext(e.Request.Context())
	form.Load(data)
	loadVirtualExpenseAttachmentInput(record, data)

	event := &core.RecordRequestEvent{
		RequestEvent: e,
		Record:       record,
	}

	originalApp := e.App
	err = e.App.RunInTransaction(func(txApp core.App) error {
		e.App = txApp
		defer func() { e.App = originalApp }()

		form.SetApp(txApp)
		form.SetRecord(event.Record)

		if err := hooks.ProcessExpense(txApp, event); err != nil {
			return err
		}
		if err := form.Submit(); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		hooks.CleanupExpenseDocumentUploads(app, event)
		return expenseWriteError(e, err)
	}

	return e.JSON(http.StatusOK, event.Record)
}

func loadVirtualExpenseAttachmentInput(record *core.Record, data map[string]any) {
	raw, ok := data["attachment"]
	if !ok {
		return
	}

	files := expenseAttachmentFilesFromRequestValue(raw)
	if len(files) == 0 {
		return
	}

	// `attachment` is the durable expense-write API field, not the durable
	// storage field. While the old expenses.attachment schema field exists,
	// RecordUpsert.Load attaches uploaded files through PocketBase's normal file
	// field machinery. After that schema field is removed, RecordUpsert will
	// correctly ignore the unknown field, so we also stash the same files under
	// the raw unsaved-file key that ProcessExpense reads.
	//
	// This keeps the client contract stable without asking the browser to create
	// expense_documents directly. The server still owns the transaction that
	// validates the expense, creates or reuses the document row, links
	// attachment_document, and rolls back the uploaded document if the expense
	// save fails.
	record.SetRaw("attachment:unsaved", files)
}

func expenseAttachmentFilesFromRequestValue(raw any) []*filesystem.File {
	switch value := raw.(type) {
	case *filesystem.File:
		return []*filesystem.File{value}
	case []*filesystem.File:
		return value
	case []any:
		files := make([]*filesystem.File, 0, len(value))
		for _, item := range value {
			if file, ok := item.(*filesystem.File); ok {
				files = append(files, file)
			}
		}
		return files
	default:
		return nil
	}
}

func expenseRecordDataFromRequest(e *core.RequestEvent, record *core.Record) (map[string]any, error) {
	info, err := e.RequestInfo()
	if err != nil {
		return nil, err
	}

	data := record.ReplaceModifiers(info.Body)

	if strings.HasPrefix(e.Request.Header.Get("content-type"), "multipart/form-data") {
		files, err := e.FindUploadedFiles("attachment")
		if err != nil && !errors.Is(err, http.ErrMissingFile) {
			return nil, err
		}
		if len(files) > 0 {
			uploaded := make([]any, 0, len(files))
			for _, file := range files {
				uploaded = append(uploaded, file)
			}
			data["attachment"] = uploaded
		}
	}

	return data, nil
}

func rejectDisallowedExpenseWriteFields(data map[string]any) error {
	for field := range data {
		if _, allowed := expenseWriteAllowedFields[field]; allowed {
			continue
		}
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "expense field is not editable through this endpoint",
			Data: map[string]errs.CodeError{
				field: {
					Code:    "not_editable",
					Message: "field is not editable through this endpoint",
				},
			},
		}
	}
	return nil
}

func expenseWriteError(e *core.RequestEvent, err error) error {
	var hookErr *errs.HookError
	if errors.As(err, &hookErr) {
		return e.JSON(hookErr.Status, hookErr)
	}

	if validationErrors, ok := err.(interface{ Filter() error }); ok {
		if filtered := validationErrors.Filter(); filtered != nil {
			return e.BadRequestError("failed to save expense", filtered)
		}
	}

	return e.BadRequestError("failed to save expense", fmt.Errorf("%w", err))
}
