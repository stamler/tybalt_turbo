package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1959231337")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"viewQuery": "WITH user_props AS (\n  SELECT \n    uc.uid AS id,\n    COALESCE(p.max_amount, 0) max_amount,\n    COALESCE(p.divisions, '[]') divisions\n  FROM user_claims uc\n  LEFT JOIN po_approver_props p \n    ON uc.id = p.user_claim\n)\nSELECT \n  up.id,\n  up.max_amount,\n  COALESCE(\n    (SELECT MAX(threshold)\n     FROM po_approval_thresholds\n     WHERE threshold < up.max_amount),\n    0\n  ) AS lower_threshold,\n  COALESCE(\n    (SELECT MIN(threshold)\n     FROM po_approval_thresholds\n     WHERE threshold >= up.max_amount),\n    1000000\n  ) AS upper_threshold,\n  up.divisions,\n  (SELECT json_group_array(c.name)\n   FROM user_claims uc\n   JOIN claims c \n     ON uc.cid = c.id\n   WHERE uc.uid = up.id) AS claims\nFROM user_props up"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_1959231337")
		if err != nil {
			return err
		}

		// update collection data
		if err := json.Unmarshal([]byte(`{
			"viewQuery": "WITH user_props AS (\n  SELECT \n    uc.uid AS id,\n    p.max_amount,\n    p.divisions\n  FROM user_claims uc\n  JOIN po_approver_props p \n    ON uc.id = p.user_claim\n)\nSELECT \n  up.id,\n  up.max_amount,\n  COALESCE(\n    (SELECT MAX(threshold)\n     FROM po_approval_thresholds\n     WHERE threshold < up.max_amount),\n    0\n  ) AS lower_threshold,\n  COALESCE(\n    (SELECT MIN(threshold)\n     FROM po_approval_thresholds\n     WHERE threshold >= up.max_amount),\n    1000000\n  ) AS upper_threshold,\n  up.divisions,\n  (SELECT json_group_array(c.name)\n   FROM user_claims uc\n   JOIN claims c \n     ON uc.cid = c.id\n   WHERE uc.uid = up.id) AS claims\nFROM user_props up"
		}`), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	})
}
