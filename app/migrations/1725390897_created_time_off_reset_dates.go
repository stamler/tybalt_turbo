package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		jsonData := `{
			"id": "c9b90wqyjpqa7tk",
			"created": "2024-09-03 19:14:57.257Z",
			"updated": "2024-09-03 19:14:57.257Z",
			"name": "time_off_reset_dates",
			"type": "base",
			"system": false,
			"schema": [
				{
					"system": false,
					"id": "kxtzlmig",
					"name": "date",
					"type": "text",
					"required": true,
					"presentable": true,
					"unique": false,
					"options": {
						"min": null,
						"max": null,
						"pattern": "^\\d{4}-\\d{2}-\\d{2}$"
					}
				}
			],
			"indexes": [],
			"listRule": null,
			"viewRule": null,
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"options": {}
		}`

		collection := &models.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return daos.New(db).SaveCollection(collection)
	}, func(db dbx.Builder) error {
		dao := daos.New(db);

		collection, err := dao.FindCollectionByNameOrId("c9b90wqyjpqa7tk")
		if err != nil {
			return err
		}

		return dao.DeleteCollection(collection)
	})
}
