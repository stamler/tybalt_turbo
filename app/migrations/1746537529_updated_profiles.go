package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("glmf9xpnwgpwudm")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"viewRule": "@request.auth.id != \"\""
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("glmf9xpnwgpwudm")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"viewRule": "// The caller is logged in and either\n@request.auth.id != \"\" &&\n(\n  // 1. The caller id matches the record uid or\n  @request.auth.id = uid ||\n  // 2. The record has the tapr claim\n  uid.user_claims_via_uid.cid.name ?= 'tapr' ||\n  // 3. The record has the po_approver claim\n  uid.user_claims_via_uid.cid.name ?= 'po_approver' ||\n  // 4. The caller has the 'tame' claim\n  @request.auth.user_claims_via_uid.cid.name ?= 'tame'\n)"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
