package migrations

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	kpiClaimID          = "kpihoursclaim01"
	kpiClaimName        = "kpi"
	kpiClaimDescription = "Can view management KPI reports"
)

func init() {
	m.Register(func(app core.App) error {
		return ensureKPIClaim(app)
	}, func(app core.App) error {
		record, err := app.FindRecordById("claims", kpiClaimID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		return app.Delete(record)
	})
}

func ensureKPIClaim(app core.App) error {
	existing, err := app.FindFirstRecordByFilter("claims", "name={:name}", dbx.Params{"name": kpiClaimName})
	if err == nil && existing != nil {
		existing.Set("description", kpiClaimDescription)
		return app.Save(existing)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	collection, err := app.FindCollectionByNameOrId("claims")
	if err != nil {
		return err
	}

	record := core.NewRecord(collection)
	record.Set("id", kpiClaimID)
	record.Set("name", kpiClaimName)
	record.Set("description", kpiClaimDescription)
	return app.Save(record)
}
