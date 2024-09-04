package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models/schema"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("zc850lb2wclrr87")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_UpEVC7E` + "`" + ` ON ` + "`" + `admin_profiles` + "`" + ` (` + "`" + `uid` + "`" + `)",
			"CREATE UNIQUE INDEX ` + "`" + `idx_XnQ4v11` + "`" + ` ON ` + "`" + `admin_profiles` + "`" + ` (` + "`" + `payroll_id` + "`" + `)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		// add
		new_payroll_id := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "d6fgkrwy",
			"name": "payroll_id",
			"type": "text",
			"required": true,
			"presentable": false,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": "^(?:[1-9]\\d*|CMS[0-9]{1,2})$"
			}
		}`), new_payroll_id); err != nil {
			return err
		}
		collection.Schema.AddField(new_payroll_id)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("zc850lb2wclrr87")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_UpEVC7E` + "`" + ` ON ` + "`" + `admin_profiles` + "`" + ` (` + "`" + `uid` + "`" + `)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		// remove
		collection.Schema.RemoveField("d6fgkrwy")

		return dao.SaveCollection(collection)
	})
}
