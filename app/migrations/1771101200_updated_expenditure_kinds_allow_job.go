package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const expenditureKindsCollectionID = "pbc_675944091"
const expenditureKindsAllowJobFieldID = "bool1771101200"

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(expenditureKindsCollectionID)
		if err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(5, []byte(`{
			"hidden": false,
			"id": "bool1771101200",
			"name": "allow_job",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "bool"
		}`)); err != nil {
			return err
		}

		if err := app.Save(collection); err != nil {
			return err
		}

		if _, err := app.DB().NewQuery(`
			UPDATE expenditure_kinds
			SET allow_job = CASE name
				WHEN 'standard' THEN 1
				WHEN 'computer' THEN 1
				ELSE 0
			END
		`).Execute(); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(expenditureKindsCollectionID)
		if err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById(expenditureKindsAllowJobFieldID)

		return app.Save(collection)
	})
}
