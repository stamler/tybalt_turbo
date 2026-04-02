package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

type importedUserAuthRepairRow struct {
	ID string `db:"id"`
}

// This migration repairs imported auth records in the users collection whose
// password and tokenKey were inserted directly by import_data instead of being
// created through PocketBase's auth-record APIs.
//
// Why this exists:
//   - The legacy import wrote blank passwords and tokenKey values shaped like
//     hex-encoded UUID strings.
//   - Those rows can exist at rest in SQLite, but PocketBase rejects them on
//     save because auth records require a non-empty password hash and a
//     tokenKey that matches the collection field constraints.
//   - That breaks later flows that need to resave the user, including the
//     Microsoft OAuth relink/identity-sync path.
//
// Predicate and idempotency:
//   - Only rows with LENGTH(password) = 0 and LENGTH(tokenKey) = 72 are
//     repaired.
//   - In the production dump this exact shape uniquely identified the imported
//     broken auth rows.
//   - After repair the password becomes a PocketBase-managed hash and tokenKey
//     becomes a PocketBase-generated value, so rerunning the migration skips
//     already-fixed users.
//
// Operational implications:
//   - Every repaired user will get a brand-new tokenKey.
//   - PocketBase derives record-token signing secrets from tokenKey plus
//     collection-specific secrets, so rotating tokenKey invalidates all
//     existing tokens for that user, including auth tokens, verification
//     tokens, password-reset tokens, email-change tokens, and file tokens.
//   - The repaired password is a new random internal value, not a user-known
//     password. This matches PocketBase's OAuth2/OTP auth-record behavior.
//   - Because the records are resaved through PocketBase, their updated
//     timestamps will advance to the migration time.
//
// Why the generation uses SetRandomPassword():
//   - This intentionally matches PocketBase source behavior for auth records.
//   - core.Record.SetRandomPassword() generates a random password, stores it as
//     a PocketBase password hash, clears the plain value so plain-password
//     validators are skipped, and then calls RefreshTokenKey().
//   - RefreshTokenKey() sets tokenKey:autogenerate, which tells PocketBase to
//     regenerate tokenKey using the field's configured autogenerate pattern on
//     save instead of us inventing our own format in this migration.
//
// Rollback note:
//   - The down migration is intentionally a no-op.
//   - Restoring the old values would reintroduce known-invalid auth data, and
//     the original blank password / imported tokenKey values are not desirable
//     to preserve.
func init() {
	m.Register(func(app core.App) error {
		return app.RunInTransaction(func(txApp core.App) error {
			var rows []importedUserAuthRepairRow
			if err := txApp.DB().NewQuery(`
				SELECT id
				FROM users
				WHERE LENGTH(COALESCE(password, '')) = 0
				  AND LENGTH(COALESCE(tokenKey, '')) = 72
			`).All(&rows); err != nil {
				return err
			}

			for _, row := range rows {
				user, err := txApp.FindRecordById("users", row.ID)
				if err != nil {
					return fmt.Errorf("find imported auth user %s: %w", row.ID, err)
				}

				user.SetRandomPassword()

				if err := txApp.Save(user); err != nil {
					return fmt.Errorf("repair imported auth user %s: %w", row.ID, err)
				}
			}

			return nil
		})
	}, func(app core.App) error {
		return nil
	})
}
