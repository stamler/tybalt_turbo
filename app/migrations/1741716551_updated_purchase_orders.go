package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"listRule": "// Active purchase_orders can be viewed by any authenticated user\n(status = \"Active\" && @request.auth.id != \"\") ||\n\n// Cancelled and Closed purchase_orders can be viewed by uid, approver, second_approver, and 'report' claim holder\n(\n  (status = \"Cancelled\" || status = \"Closed\") &&\n  (\n    @request.auth.id = uid || \n    @request.auth.id = approver || \n    @request.auth.id = second_approver || \n    @request.auth.user_claims_via_uid.cid.name ?= 'report'\n  )\n) ||\n\n// TODO: We may also later allow Closed purchase_orders to be viewed by uid, approver, and committer of corresponding expenses, if any, in a rule here\n(\n  true = true\n) ||\n\n// Unapproved purchase_orders can be viewed by uid, approver, priority_second_approver and, if updated is more than 24 hours ago, any holder of second_approver_claim\n(\n  status = \"Unapproved\" &&\n  (\n    @request.auth.id = uid || \n    @request.auth.id = approver || \n    @request.auth.id = priority_second_approver || \n    (\n      // updated more than 24 hours ago and @request.auth.id holds second_approver_claim\n      updated < @yesterday && @request.auth.user_claims_via_uid.cid ?= second_approver_claim\n    )\n  )\n)",
			"viewRule": "// Active purchase_orders can be viewed by any authenticated user\n(status = \"Active\" && @request.auth.id != \"\") ||\n\n// Cancelled and Closed purchase_orders can be viewed by uid, approver, second_approver, and 'report' claim holder\n(\n  (status = \"Cancelled\" || status = \"Closed\") &&\n  (\n    @request.auth.id = uid || \n    @request.auth.id = approver || \n    @request.auth.id = second_approver || \n    @request.auth.user_claims_via_uid.cid.name ?= 'report'\n  )\n) ||\n\n// TODO: We may also later allow Closed purchase_orders to be viewed by uid, approver, and committer of corresponding expenses, if any, in a rule here\n(\n  true = true\n) ||\n\n// Unapproved purchase_orders can be viewed by uid, approver, priority_second_approver and, if updated is more than 24 hours ago, any holder of second_approver_claim\n(\n  status = \"Unapproved\" &&\n  (\n    @request.auth.id = uid || \n    @request.auth.id = approver || \n    @request.auth.id = priority_second_approver || \n    (\n      // updated more than 24 hours ago and @request.auth.id holds second_approver_claim\n      updated < @yesterday && @request.auth.user_claims_via_uid.cid ?= second_approver_claim\n    )\n  )\n)"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("m19q72syy0e3lvm")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"listRule": "@request.auth.id = uid ||\n@request.auth.id = approver ||\n@request.auth.id = second_approver",
			"viewRule": "@request.auth.id = uid ||\n@request.auth.id = approver ||\n@request.auth.id = second_approver"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
