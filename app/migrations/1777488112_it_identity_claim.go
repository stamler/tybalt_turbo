package migrations

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	itIdentityClaimID          = "itclaim00000001"
	itIdentityClaimName        = "it"
	itIdentityClaimDescription = "Can manage identity repair fields and authorized providers for users"
)

func init() {
	m.Register(func(app core.App) error {
		return ensureITIdentityClaim(app)
	}, func(app core.App) error {
		record, err := app.FindRecordById("claims", itIdentityClaimID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		return app.Delete(record)
	})
}

func ensureITIdentityClaim(app core.App) error {
	existing, err := app.FindFirstRecordByFilter("claims", "name={:name}", dbx.Params{"name": itIdentityClaimName})
	if err == nil && existing != nil {
		existing.Set("description", itIdentityClaimDescription)
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
	record.Set("id", itIdentityClaimID)
	record.Set("name", itIdentityClaimName)
	record.Set("description", itIdentityClaimDescription)
	return app.Save(record)
}
