package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	fixExpenseAttachmentRecordID       = "yo9pv26rwic85tc"
	fixExpenseAttachmentOriginalName   = "179_mcidjha551.qsimInvoice#1885.pdf"
	fixExpenseAttachmentNormalizedName = "179_mcidjha551.qsimInvoice_1885.pdf"
	fixExpenseAttachmentQuery          = `
		UPDATE expenses
		SET
			attachment = {:newAttachment},
			updated = strftime('%Y-%m-%d %H:%M:%fZ', 'now')
		WHERE id = {:id}
		  AND attachment = {:oldAttachment}
	`
)

func init() {
	m.Register(func(app core.App) error {
		return updateExpenseAttachmentFilename(
			app,
			fixExpenseAttachmentOriginalName,
			fixExpenseAttachmentNormalizedName,
		)
	}, func(app core.App) error {
		return updateExpenseAttachmentFilename(
			app,
			fixExpenseAttachmentNormalizedName,
			fixExpenseAttachmentOriginalName,
		)
	})
}

func updateExpenseAttachmentFilename(app core.App, oldAttachment string, newAttachment string) error {
	_, err := app.DB().NewQuery(fixExpenseAttachmentQuery).Bind(dbx.Params{
		"id":            fixExpenseAttachmentRecordID,
		"oldAttachment": oldAttachment,
		"newAttachment": newAttachment,
	}).Execute()
	return err
}
