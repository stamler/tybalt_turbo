package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	purchaseOrdersCollectionID          = "m19q72syy0e3lvm"
	purchaseOrdersAugmentedCollectionID = "pbc_1245168108"
	purchaseOrdersTypeFieldID           = "wwwtd51w"
	purchaseOrdersAugmentedTypeFieldID  = "_clone_QX9z"
)

func init() {
	m.Register(func(app core.App) error {
		if err := setPurchaseOrderTypeOptions(app, []string{"Normal", "One-Time", "Cumulative", "Recurring"}); err != nil {
			return err
		}

		if _, err := app.DB().NewQuery(`
			UPDATE purchase_orders
			SET type = 'One-Time'
			WHERE type = 'Normal'
		`).Execute(); err != nil {
			return err
		}

		return setPurchaseOrderTypeOptions(app, []string{"One-Time", "Cumulative", "Recurring"})
	}, func(app core.App) error {
		if err := setPurchaseOrderTypeOptions(app, []string{"Normal", "One-Time", "Cumulative", "Recurring"}); err != nil {
			return err
		}

		if _, err := app.DB().NewQuery(`
			UPDATE purchase_orders
			SET type = 'Normal'
			WHERE type = 'One-Time'
		`).Execute(); err != nil {
			return err
		}

		return setPurchaseOrderTypeOptions(app, []string{"Normal", "Cumulative", "Recurring"})
	})
}

func setPurchaseOrderTypeOptions(app core.App, values []string) error {
	if err := updatePurchaseOrderTypeField(
		app,
		purchaseOrdersCollectionID,
		purchaseOrdersTypeFieldID,
		values,
	); err != nil {
		return err
	}

	return updatePurchaseOrderTypeField(
		app,
		purchaseOrdersAugmentedCollectionID,
		purchaseOrdersAugmentedTypeFieldID,
		values,
	)
}

func updatePurchaseOrderTypeField(app core.App, collectionID string, fieldID string, values []string) error {
	collection, err := app.FindCollectionByNameOrId(collectionID)
	if err != nil {
		return err
	}

	fieldJSON, err := json.Marshal(map[string]any{
		"hidden":      false,
		"id":          fieldID,
		"maxSelect":   1,
		"name":        "type",
		"presentable": false,
		"required":    true,
		"system":      false,
		"type":        "select",
		"values":      values,
	})
	if err != nil {
		return err
	}

	collection.Fields.RemoveById(fieldID)
	if err := collection.Fields.AddMarshaledJSONAt(4, fieldJSON); err != nil {
		return err
	}

	return app.Save(collection)
}
