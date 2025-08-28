package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pmxhrqhngh60icm")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"createRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'admin'",
			"deleteRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'admin'",
			"listRule": "@request.auth.id = uid ||\n(\n  @request.auth.id != \"\" &&\n  @request.auth.user_claims_via_uid.cid.name ?= 'admin'\n)",
			"updateRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'admin'",
			"viewRule": "@request.auth.id = uid ||\n(\n  @request.auth.id != \"\" &&\n  @request.auth.user_claims_via_uid.cid.name ?= 'admin'\n)"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pmxhrqhngh60icm")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"createRule": null,
			"deleteRule": null,
			"listRule": "@request.auth.id = uid",
			"updateRule": null,
			"viewRule": "@request.auth.id = uid"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
