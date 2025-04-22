package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("3v7wxidd2f9yhf9")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_KxKk01Y` + "`" + ` ON ` + "`" + `client_contacts` + "`" + ` (\n  ` + "`" + `surname` + "`" + `,\n  ` + "`" + `given_name` + "`" + `\n)",
				"CREATE UNIQUE INDEX ` + "`" + `idx_0tGweKG7kJ` + "`" + ` ON ` + "`" + `client_contacts` + "`" + ` (` + "`" + `email` + "`" + `) WHERE email != ''"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("3v7wxidd2f9yhf9")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"indexes": [
				"CREATE INDEX ` + "`" + `idx_KxKk01Y` + "`" + ` ON ` + "`" + `client_contacts` + "`" + ` (\n  ` + "`" + `surname` + "`" + `,\n  ` + "`" + `given_name` + "`" + `\n)"
			]
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
