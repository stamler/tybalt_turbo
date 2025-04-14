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
					"id": "number2431691161",
					"max": null,
					"min": null,
					"name": "num_pos_qualified",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				}
			],
			"id": "pbc_540250600",
			"indexes": [],
			"listRule": null,
			"name": "pending_items_for_qualified_po_second_approvers",
			"system": false,
			"type": "view",
			"updateRule": null,
			"viewQuery": "WITH QualifiedUsers AS (\n    SELECT\n        u.id AS user_id,\n        pap.max_amount,\n        pap.divisions\n    FROM users u\n    JOIN user_claims uc ON u.id = uc.uid\n    JOIN claims c ON uc.cid = c.id\n    JOIN po_approver_props pap ON uc.id = pap.user_claim\n    WHERE c.name = 'po_approver'\n),\nPOsNeedingSecondApproval AS (\n    SELECT\n        po.id AS po_id,\n        po.approval_total,\n        po.division,\n        poa.upper_threshold\n    FROM purchase_orders po\n    JOIN purchase_orders_augmented poa ON po.id = poa.id\n    WHERE\n        po.approved != ''\n        AND po.status = 'Unapproved'\n        AND po.second_approval = ''\n        AND po.updated < strftime('%Y-%m-%d %H:%M:%fZ', 'now', '-24 hours')\n)\nSELECT\n    qu.user_id AS id,\n    COUNT(pnsa.po_id) AS num_pos_qualified\nFROM QualifiedUsers qu\nLEFT JOIN POsNeedingSecondApproval pnsa\n    ON qu.max_amount >= pnsa.approval_total\n    AND (\n        json_valid(qu.divisions) AND json_array_length(qu.divisions) = 0\n        OR (\n           json_valid(qu.divisions) AND EXISTS (SELECT 1 FROM json_each(qu.divisions) WHERE value = pnsa.division)\n        )\n    )\n    AND qu.max_amount <= pnsa.upper_threshold\nGROUP BY qu.user_id;",
			"viewRule": null
		}`

		collection := &core.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_540250600")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
