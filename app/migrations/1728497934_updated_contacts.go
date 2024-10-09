package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("3v7wxidd2f9yhf9")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE INDEX ` + "`" + `idx_KxKk01Y` + "`" + ` ON ` + "`" + `contacts` + "`" + ` (\n  ` + "`" + `surname` + "`" + `,\n  ` + "`" + `given_name` + "`" + `\n)",
			"CREATE UNIQUE INDEX ` + "`" + `idx_0KoVkzQ` + "`" + ` ON ` + "`" + `contacts` + "`" + ` (` + "`" + `email` + "`" + `)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("3v7wxidd2f9yhf9")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE INDEX ` + "`" + `idx_KxKk01Y` + "`" + ` ON ` + "`" + `contacts` + "`" + ` (\n  ` + "`" + `surname` + "`" + `,\n  ` + "`" + `given_name` + "`" + `\n)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	})
}
