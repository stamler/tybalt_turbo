package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
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
					"hidden": false,
					"id": "json432058571",
					"maxSize": 1,
					"name": "max_amount",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json2651176244",
					"maxSize": 1,
					"name": "lower_threshold",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json3779115990",
					"maxSize": 1,
					"name": "upper_threshold",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json29625393",
					"maxSize": 1,
					"name": "divisions",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json3198358462",
					"maxSize": 1,
					"name": "claims",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				}
			],
			"id": "pbc_1959231337",
			"indexes": [],
			"listRule": null,
			"name": "user_po_permission_data",
			"system": false,
			"type": "view",
			"updateRule": null,
			"viewQuery": "WITH user_props AS (\n  SELECT \n    uc.uid AS id,\n    p.max_amount,\n    p.divisions\n  FROM user_claims uc\n  JOIN po_approver_props p \n    ON uc.id = p.user_claim\n)\nSELECT \n  up.id,\n  up.max_amount,\n  COALESCE(\n    (SELECT MAX(threshold)\n     FROM po_approval_thresholds\n     WHERE threshold < up.max_amount),\n    0\n  ) AS lower_threshold,\n  COALESCE(\n    (SELECT MIN(threshold)\n     FROM po_approval_thresholds\n     WHERE threshold >= up.max_amount),\n    1000000\n  ) AS upper_threshold,\n  up.divisions,\n  (SELECT json_group_array(c.name)\n   FROM user_claims uc\n   JOIN claims c \n     ON uc.cid = c.id\n   WHERE uc.uid = up.id) AS claims\nFROM user_props up",
			"viewRule": null
		}`

		collection := &core.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1959231337")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
