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

		collection, err := dao.FindCollectionByNameOrId("y0xvnesailac971")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_GCZxhiM` + "`" + ` ON ` + "`" + `vendors` + "`" + ` (` + "`" + `name` + "`" + `)",
			"CREATE UNIQUE INDEX ` + "`" + `idx_c8OTvkU` + "`" + ` ON ` + "`" + `vendors` + "`" + ` (` + "`" + `alias` + "`" + `) WHERE ` + "`" + `alias` + "`" + ` != ''"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("y0xvnesailac971")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_GCZxhiM` + "`" + ` ON ` + "`" + `vendors` + "`" + ` (` + "`" + `name` + "`" + `)",
			"CREATE UNIQUE INDEX ` + "`" + `idx_c8OTvkU` + "`" + ` ON ` + "`" + `vendors` + "`" + ` (` + "`" + `alias` + "`" + `)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		return dao.SaveCollection(collection)
	})
}
