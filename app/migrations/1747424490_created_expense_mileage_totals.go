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
					"id": "json1402668550",
					"maxSize": 1,
					"name": "uid",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json2862495610",
					"maxSize": 1,
					"name": "date",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json226078728",
					"maxSize": 1,
					"name": "reset_mileage_date",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json479369857",
					"maxSize": 1,
					"name": "distance",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json2857502310",
					"maxSize": 1,
					"name": "cumulative",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json790377260",
					"maxSize": 1,
					"name": "effective_date",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json792564374",
					"maxSize": 1,
					"name": "mileage_total",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				}
			],
			"id": "pbc_2931110398",
			"indexes": [],
			"listRule": null,
			"name": "expense_mileage_totals",
			"system": false,
			"type": "view",
			"updateRule": null,
			"viewQuery": "-- 1) Explode + window up front\nWITH rates_expanded AS (\n  SELECT\n    m.effective_date,\n    CAST(t.key   AS INTEGER) AS tier_lower,\n    LEAD(CAST(t.key AS INTEGER)) OVER (\n      PARTITION BY m.effective_date\n      ORDER BY CAST(t.key AS INTEGER)\n    ) AS tier_upper,\n    CAST(t.value AS REAL)    AS tier_rate\n\n  FROM expense_rates m\n  CROSS JOIN json_each(m.mileage) AS t\n),\n\n-- 2) Base: compute cumulative & interval boundaries\nbase AS (\n  SELECT\n    e.id,\n    e.uid,\n    e.date,\n    -- This ends up being faster than joining to mileage_reset_dates\n    (\n      SELECT MAX(r.date)\n      FROM mileage_reset_dates r\n      WHERE r.date <= e.date\n    ) AS reset_mileage_date,\n    e.distance,\n    -- interval end = cumulative distance\n    SUM(e.distance) OVER (\n      PARTITION BY e.uid, (\n        SELECT MAX(r.date)\n        FROM mileage_reset_dates r\n        WHERE r.date <= e.date\n      )\n      ORDER BY e.date\n    ) AS end_distance,\n    (\n      SELECT MAX(m.effective_date)\n      FROM expense_rates m\n      WHERE m.effective_date <= e.date\n    ) AS effective_date\n  FROM expenses e\n  WHERE e.payment_type = 'Mileage'\n),\n\n/* \n3) Join each expense to its tiers, filtering to only those that overlap We are\npairing each expense’s [start_distance, end_distance) interval with only those\ntier intervals [tier_lower, tier_upper) that intersect it. By exploding each\ntier into its own row, an expense whose kilometres cross a tier boundary will\njoin to two (or more, in theory) tier‐rows. Thus we can expect that the number\nof rows in overlaps can be more than the number of rows in expenses.\n*/\noverlaps AS (\n  SELECT\n    b.id,\n    b.end_distance - b.distance AS start_distance,\n    b.end_distance,\n    r.tier_lower,\n    COALESCE(r.tier_upper, 1e9) AS tier_upper,\n    r.tier_rate\n  FROM base b\n  JOIN rates_expanded r\n    ON r.effective_date = b.effective_date\n  WHERE b.end_distance > r.tier_lower\n    AND (r.tier_upper IS NULL OR (b.end_distance - b.distance) < r.tier_upper)\n),\n\n/*\n4) Compute overlap length per tier\nfor each expense × tier row, compute how many kilometres of the expense fall into that tier by:\n  1. Determining the overlap interval between the expense’s [start_distance, end_distance)\n     and the tier’s [tier_lower, tier_upper) interval:\n       • overlap_start = max(start_distance, tier_lower)\n       • overlap_end   = tier_upper IS NULL\n                           ? end_distance\n                           : min(end_distance, tier_upper)\n  2. Calculating overlap_km = max(0, overlap_end − overlap_start)\n       • Ensures negative values (no overlap) are clipped to zero\n  3. Carrying along tier_rate so we can later multiply overlap_km × tier_rate\n*/\ntier_calcs AS (\n  SELECT\n    id,\n    max(0,\n      min(end_distance, tier_upper)\n      - max(start_distance, tier_lower)\n    ) AS overlap_km,\n    tier_rate\n  FROM overlaps\n)\n\n-- 5) Sum up reimbursement per expense\nSELECT\n  b.id,\n  b.uid,\n  b.date,\n  b.reset_mileage_date,\n  b.distance,\n  b.end_distance AS cumulative,\n  b.effective_date,\n  ROUND(COALESCE(\n    -- sum up this expense’s (overlap × rate) directly\n    (SELECT SUM(overlap_km * tier_rate)\n     FROM tier_calcs tc\n     WHERE tc.id = b.id),\n    0\n  ), 2) AS mileage_total\nFROM base b\nORDER BY b.date;\n",
			"viewRule": null
		}`

		collection := &core.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_2931110398")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
