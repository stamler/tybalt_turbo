package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_3847400939")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	}, func(app core.App) error {
		jsonData := `{
			"createRule": null,
			"deleteRule": null,
			"fields": [
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text3208210256",
					"max": 0,
					"min": 0,
					"name": "id",
					"pattern": "^[a-z0-9]+$",
					"presentable": false,
					"primaryKey": true,
					"required": true,
					"system": true,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_ZMrb",
					"max": 48,
					"min": 2,
					"name": "surname",
					"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_lo78",
					"max": 48,
					"min": 2,
					"name": "given_name",
					"pattern": "^[a-zA-Z]+(?:-[a-zA-Z]+)*$",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				}
			],
			"id": "pbc_3847400939",
			"indexes": [],
			"listRule": "@request.auth.id != \"\"",
			"name": "po_second_approvers",
			"system": false,
			"type": "view",
			"updateRule": null,
			"viewQuery": "SELECT p.uid AS id, p.surname AS surname, p.given_name AS given_name \nFROM profiles p\nINNER JOIN user_claims u ON p.uid = u.uid\nINNER JOIN claims c ON u.cid = c.id\nWHERE c.name = 'po_approver_tier2'",
			"viewRule": "@request.auth.id != \"\""
		}`

		collection := &core.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
