package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("ranctx5xgih6n3a")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"listRule": "@request.auth.id = uid ||\n(tsid.submitted = true && @request.auth.id = tsid.approver) ||\n(tsid.submitted = true && @request.auth.id ?= tsid.time_sheet_reviewers_via_time_sheet.reviewer) ||\n(tsid.committed != '' && @request.auth.user_claims_via_uid.cid.name ?= 'report')",
			"viewRule": "@request.auth.id = uid ||\n(tsid.submitted = true && @request.auth.id = tsid.approver) ||\n(tsid.submitted = true && @request.auth.id ?= tsid.time_sheet_reviewers_via_time_sheet.reviewer) ||\n(tsid.committed != '' && @request.auth.user_claims_via_uid.cid.name ?= 'report')"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("ranctx5xgih6n3a")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"listRule": "@request.auth.id = uid ||\n(tsid.submitted = true && @request.auth.id = tsid.approver) ||\n(tsid.committed != '' && @request.auth.user_claims_via_uid.cid.name ?= 'report')",
			"viewRule": "@request.auth.id = uid ||\n(tsid.submitted = true && @request.auth.id = tsid.approver) ||\n(tsid.committed != '' && @request.auth.user_claims_via_uid.cid.name ?= 'report')"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
