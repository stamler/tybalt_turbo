package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("y0xvnesailac971")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"createRule": "@request.auth.id != \"\" &&\n(\n  @request.auth.user_claims_via_uid.cid.name ?= 'time' ||\n  @request.auth.user_claims_via_uid.cid.name ?= 'payables_admin'\n)"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("y0xvnesailac971")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"createRule": "@request.auth.user_claims_via_uid.cid.name ?= 'payables_admin'"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
