package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

const (
	// This migration removes the old `committed = ''` requirement from the
	// commit-claim branch of the time_sheets list/view rules. After this
	// migration, commit holders can read approved time_sheets whether they are
	// still awaiting commit or already committed. It does not change action
	// permissions; commit/reject route handlers still control what commit users
	// are allowed to do with unapproved or already committed records.
	timeSheetsCommitVisibilityNewRule = "@request.auth.id = uid ||\n(submitted = true && @request.auth.id = approver) ||\n(submitted = true && @request.auth.id ?= time_sheet_reviewers_via_time_sheet.reviewer) ||\n(submitted = true && approved != '' && @request.auth.user_claims_via_uid.cid.name ?= 'commit') ||\n(committed != '' && @request.auth.user_claims_via_uid.cid.name ?= 'report')"
	timeSheetsCommitVisibilityOldRule = "@request.auth.id = uid ||\n(submitted = true && @request.auth.id = approver) ||\n(submitted = true && @request.auth.id ?= time_sheet_reviewers_via_time_sheet.reviewer) ||\n(submitted = true && approved != '' && committed = '' && @request.auth.user_claims_via_uid.cid.name ?= 'commit') ||\n(committed != '' && @request.auth.user_claims_via_uid.cid.name ?= 'report')"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("time_sheets")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer(timeSheetsCommitVisibilityNewRule)
		collection.ViewRule = types.Pointer(timeSheetsCommitVisibilityNewRule)

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("time_sheets")
		if err != nil {
			return err
		}

		collection.ListRule = types.Pointer(timeSheetsCommitVisibilityOldRule)
		collection.ViewRule = types.Pointer(timeSheetsCommitVisibilityOldRule)

		return app.Save(collection)
	})
}
