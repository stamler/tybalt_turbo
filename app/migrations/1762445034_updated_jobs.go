package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("yovqzrnnomp0lkx")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"updateRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job' &&\n\n// the contact belongs to the client (only when changing either)\n(\n  (@request.body.contact:isset = false && @request.body.client:isset = false) ||\n  (@request.body.contact:isset = true && @request.body.client:isset = true  && @request.body.contact.client = @request.body.client) ||\n  (@request.body.contact:isset = true && @request.body.client:isset = false && @request.body.contact.client = client) ||\n  (@request.body.contact:isset = false && @request.body.client:isset = true  && contact.client = @request.body.client)\n) &&\n\n// the job number cannot be changed once created\n(@request.body.number:isset = false || @request.body.number = number)"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("yovqzrnnomp0lkx")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"updateRule": "@request.auth.id != \"\" &&\n@request.auth.user_claims_via_uid.cid.name ?= 'job' &&\n\n// the contact belongs to the client\n@request.body.contact.client = @request.body.client &&\n\n// the job number cannot be changed once created\n(@request.body.number:isset = false || @request.body.number = number)"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
