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
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_n5LU",
					"max": 0,
					"min": 0,
					"name": "date",
					"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "_clone_5bdv",
					"max": 0,
					"min": 0,
					"name": "allowance_rates_effective_date",
					"pattern": "^\\d{4}-\\d{2}-\\d{2}$",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "_clone_SZvK",
					"maxSelect": 1,
					"name": "payment_type",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "select",
					"values": [
						"OnAccount",
						"Expense",
						"CorporateCreditCard",
						"Allowance",
						"FuelCard",
						"Mileage",
						"PersonalReimbursement"
					]
				},
				{
					"hidden": false,
					"id": "_clone_7Poq",
					"maxSelect": 4,
					"name": "allowance_types",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "select",
					"values": [
						"Lodging",
						"Breakfast",
						"Lunch",
						"Dinner"
					]
				},
				{
					"hidden": false,
					"id": "_clone_yiGs",
					"max": null,
					"min": 0,
					"name": "breakfast_rate",
					"onlyInt": false,
					"presentable": false,
					"required": true,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "_clone_H9ON",
					"max": null,
					"min": 0,
					"name": "lunch_rate",
					"onlyInt": false,
					"presentable": false,
					"required": true,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "_clone_HCfz",
					"max": null,
					"min": 0,
					"name": "dinner_rate",
					"onlyInt": false,
					"presentable": false,
					"required": true,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "_clone_Kf2D",
					"max": null,
					"min": 0,
					"name": "lodging_rate",
					"onlyInt": false,
					"presentable": false,
					"required": true,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "_clone_4TXZ",
					"maxSize": 2000000,
					"name": "mileage",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json1192429895",
					"maxSize": 1,
					"name": "allowance_total",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json211419417",
					"maxSize": 1,
					"name": "allowance_description",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				}
			],
			"id": "pbc_582883213",
			"indexes": [],
			"listRule": null,
			"name": "expense_allowance_totals",
			"system": false,
			"type": "view",
			"updateRule": null,
			"viewQuery": "SELECT e.id, \n  e.date, \n  r.effective_date allowance_rates_effective_date, \n  e.payment_type,\n  e.allowance_types,\n  r.breakfast breakfast_rate,\n  r.lunch lunch_rate,\n  r.dinner dinner_rate,\n  r.lodging lodging_rate,\n  r.mileage,\n  ((CASE WHEN e.allowance_types LIKE '%\"Breakfast\"%'   THEN r.breakfast ELSE 0 END)\n  + (CASE WHEN e.allowance_types LIKE '%\"Lunch\"%'     THEN r.lunch     ELSE 0 END)\n  + (CASE WHEN e.allowance_types LIKE '%\"Dinner\"%'    THEN r.dinner    ELSE 0 END)\n  + (CASE WHEN e.allowance_types LIKE '%\"Lodging\"%'   THEN r.lodging   ELSE 0 END)\n  )AS allowance_total,\n  RTRIM(\n    (CASE WHEN e.allowance_types LIKE '%\"Breakfast\"%' THEN 'Breakfast ' ELSE '' END) ||\n    (CASE WHEN e.allowance_types LIKE '%\"Lunch\"%'     THEN 'Lunch '     ELSE '' END) ||\n    (CASE WHEN e.allowance_types LIKE '%\"Dinner\"%'    THEN 'Dinner '    ELSE '' END) ||\n    (CASE WHEN e.allowance_types LIKE '%\"Lodging\"%'   THEN 'Lodging '   ELSE '' END)\n  ) AS allowance_description\nFROM expenses e \nLEFT JOIN expense_rates r ON ((r.effective_date = (SELECT MAX(i.effective_date) FROM expense_rates i WHERE (i.effective_date <= e.date))))\nWHERE e.payment_type IN ('Allowance','Meals');",
			"viewRule": null
		}`

		collection := &core.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_582883213")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
