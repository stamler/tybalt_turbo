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
			"listRule": "@request.auth.id != \"\" && (\n  uid = @request.auth.id || \n  manager = @request.auth.id || \n  @request.auth.user_claims_via_uid.cid.name ?= 'tame' ||\n  @request.auth.user_claims_via_uid.cid.name ?= 'job'\n)"
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
			"listRule": "@request.auth.id != \"\" &&\n(uid = @request.auth.id || manager = @request.auth.id || @request.auth.user_claims_via_uid.cid.name ?= 'tame')"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
