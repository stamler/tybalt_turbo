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

		collection, err := dao.FindCollectionByNameOrId("yovqzrnnomp0lkx")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_V1RKd7H` + "`" + ` ON ` + "`" + `jobs` + "`" + ` (` + "`" + `number` + "`" + `)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		// update
		edit_number := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "zloyds7s",
			"name": "number",
			"type": "text",
			"required": true,
			"presentable": true,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": "(P)?[0-9]{2}-[0-9]{3,4}(-[0-9]{1,2})?(-[0-9])?"
			}
		}`), edit_number); err != nil {
			return err
		}
		collection.Schema.AddField(edit_number)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("yovqzrnnomp0lkx")
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(`[
			"CREATE UNIQUE INDEX ` + "`" + `idx_V1RKd7H` + "`" + ` ON ` + "`" + `jobs` + "`" + ` (` + "`" + `job_number` + "`" + `)"
		]`), &collection.Indexes); err != nil {
			return err
		}

		// update
		edit_number := &schema.SchemaField{}
		if err := json.Unmarshal([]byte(`{
			"system": false,
			"id": "zloyds7s",
			"name": "job_number",
			"type": "text",
			"required": true,
			"presentable": true,
			"unique": false,
			"options": {
				"min": null,
				"max": null,
				"pattern": "(P)?[0-9]{2}-[0-9]{3,4}(-[0-9]{1,2})?(-[0-9])?"
			}
		}`), edit_number); err != nil {
			return err
		}
		collection.Schema.AddField(edit_number)

		return dao.SaveCollection(collection)
	})
}
