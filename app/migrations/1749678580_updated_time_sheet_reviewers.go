package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("g3surmbkacieshv")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"createRule": "// The caller is authenticated\n@request.auth.id != \"\" &&\n// The caller is the approver of the specified time_sheet\n@request.auth.time_sheets_via_approver.id ?= time_sheet",
			"deleteRule": "// The caller is authenticated\n@request.auth.id != \"\" &&\n// The caller is the approver of the specified time_sheet\n@request.auth.time_sheets_via_approver.id ?= time_sheet",
			"listRule": "// The caller is the reviewer\n@request.auth.id = reviewer ||\n// The caller has the uid of a referenced time sheet\n@request.auth.id = time_sheet.uid ||\n// The caller is the approver of a referenced time sheet\n@request.auth.id = time_sheet.approver",
			"viewRule": "// The caller is the reviewer\n@request.auth.id = reviewer ||\n// The caller has the uid of a referenced time sheet\n@request.auth.id = time_sheet.uid ||\n// The caller is the approver of a referenced time sheet\n@request.auth.id = time_sheet.approver"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("g3surmbkacieshv")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"createRule": "@request.auth.id != \"\" &&\n@request.auth.time_sheets_via_approver.id ?= time_sheet",
			"deleteRule": "@request.auth.id != \"\" &&\ntime_sheet ?= @request.auth.time_sheets_via_approver.id",
			"listRule": "@request.auth.id = @collection.users.time_sheets_via_approver.approver ||\n@request.auth.id = reviewer",
			"viewRule": "@request.auth.id = @collection.users.time_sheets_via_approver.approver ||\n@request.auth.id = reviewer"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
