package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_3457251721")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"updateRule": "@request.auth.id != \"\" &&\n(\n  @request.auth.user_claims_via_uid.cid.name ?= 'job' ||\n  @request.auth.jobs_via_manager.id ?= job ||\n  @request.auth.jobs_via_alternate_manager.id ?= job\n)"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_3457251721")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"updateRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job'"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
