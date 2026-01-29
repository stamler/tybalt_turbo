package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_126575313")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"createRule": "@request.auth.id != \"\" && (\n  @request.auth.user_claims_via_uid.cid.name ?= 'rate_sheet_revise' ||\n  @request.auth.user_claims_via_uid.cid.name ?= 'admin' ||\n  (@request.auth.user_claims_via_uid.cid.name ?= 'job' && @request.body.revision = 0)\n)"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_126575313")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"createRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job'"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
