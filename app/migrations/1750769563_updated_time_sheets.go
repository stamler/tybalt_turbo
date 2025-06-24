package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("fpri53nrr2xgoov")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"listRule": "@request.auth.id = uid ||\n(submitted = true && @request.auth.id = approver) ||\n(submitted = true && @request.auth.id ?= time_sheet_reviewers_via_time_sheet.reviewer) ||\n(committed != '' && @request.auth.user_claims_via_uid.cid.name ?= 'report')",
			"viewRule": "@request.auth.id = uid ||\n(submitted = true && @request.auth.id = approver) ||\n(submitted = true && @request.auth.id ?= time_sheet_reviewers_via_time_sheet.reviewer) ||\n(committed != '' && @request.auth.user_claims_via_uid.cid.name ?= 'report')"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("fpri53nrr2xgoov")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"listRule": "@request.auth.id = uid ||\n(submitted = true && @request.auth.id = approver) ||\n(committed != '' && @request.auth.user_claims_via_uid.cid.name ?= 'report')",
			"viewRule": "@request.auth.id = uid ||\n(submitted = true && @request.auth.id = approver) ||\n(committed != '' && @request.auth.user_claims_via_uid.cid.name ?= 'report')"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
