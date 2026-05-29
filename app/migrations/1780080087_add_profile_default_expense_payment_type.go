package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("glmf9xpnwgpwudm")
		if err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(12, []byte(`{
			"hidden": false,
			"id": "select1780080087",
			"maxSelect": 1,
			"name": "default_expense_payment_type",
			"presentable": false,
			"required": false,
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
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("glmf9xpnwgpwudm")
		if err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("select1780080087")

		return app.Save(collection)
	})
}
