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
			"listRule": null,
			"viewRule": null
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
			"listRule": "// The caller is the reviewer\n@request.auth.id = reviewer ||\n// The caller has the uid of a referenced time sheet\n@request.auth.id = time_sheet.uid ||\n// The caller is the approver of a referenced time sheet\n@request.auth.id = time_sheet.approver",
			"viewRule": "// The caller is the reviewer\n@request.auth.id = reviewer ||\n// The caller has the uid of a referenced time sheet\n@request.auth.id = time_sheet.uid ||\n// The caller is the approver of a referenced time sheet\n@request.auth.id = time_sheet.approver"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
