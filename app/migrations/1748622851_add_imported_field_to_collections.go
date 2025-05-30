package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Collections to update with _imported field
		collectionsInfo := []struct {
			id   string
			name string
		}{
			{"1v6i9rrpniuatcx", "clients"},
			{"3v7wxidd2f9yhf9", "client_contacts"},
			{"yovqzrnnomp0lkx", "jobs"},
			{"nrwhbwowokwu6cr", "categories"},
			{"y0xvnesailac971", "vendors"},
			{"m19q72syy0e3lvm", "purchase_orders"},
			{"o1vpz1mm7qsfoyy", "expenses"},
			{"glmf9xpnwgpwudm", "profiles"},
			{"zc850lb2wclrr87", "admin_profiles"},
			{"pmxhrqhngh60icm", "user_claims"},
			{"fpri53nrr2xgoov", "time_sheets"},
			{"ranctx5xgih6n3a", "time_entries"},
			{"5z24r2v5jgh8qft", "time_amendments"},
			{"pbc_2078099607", "mileage_reset_dates"},
		}

		// Add _imported field to each collection
		for i, collInfo := range collectionsInfo {
			collection, err := app.FindCollectionByNameOrId(collInfo.id)
			if err != nil {
				return err
			}

			// Create unique field ID for each collection
			fieldId := fmt.Sprintf("bool_imported_%d", i+1)

			// Add _imported field at the end
			if err := collection.Fields.AddMarshaledJSONAt(-1, []byte(fmt.Sprintf(`{
				"hidden": false,
				"id": "%s",
				"name": "_imported",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "bool"
			}`, fieldId))); err != nil {
				return err
			}

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Rollback: Remove _imported field from all collections
		collectionsInfo := []struct {
			id   string
			name string
		}{
			{"1v6i9rrpniuatcx", "clients"},
			{"3v7wxidd2f9yhf9", "client_contacts"},
			{"yovqzrnnomp0lkx", "jobs"},
			{"nrwhbwowokwu6cr", "categories"},
			{"y0xvnesailac971", "vendors"},
			{"m19q72syy0e3lvm", "purchase_orders"},
			{"o1vpz1mm7qsfoyy", "expenses"},
			{"glmf9xpnwgpwudm", "profiles"},
			{"zc850lb2wclrr87", "admin_profiles"},
			{"pmxhrqhngh60icm", "user_claims"},
			{"fpri53nrr2xgoov", "time_sheets"},
			{"ranctx5xgih6n3a", "time_entries"},
			{"5z24r2v5jgh8qft", "time_amendments"},
			{"pbc_2078099607", "mileage_reset_dates"},
		}

		for i, collInfo := range collectionsInfo {
			collection, err := app.FindCollectionByNameOrId(collInfo.id)
			if err != nil {
				return err
			}

			// Remove _imported field using the same unique ID
			fieldId := fmt.Sprintf("bool_imported_%d", i+1)
			collection.Fields.RemoveById(fieldId)

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		return nil
	})
}
