package load

import (
	"database/sql"
	"fmt"
)

type ClientBillingImportReader interface{ QueryRow(string, ...any) *sql.Row }

// CheckClientBillingImport stops bulk replacement of billing profile references.
// The importer cannot restore locally maintained profiles or their job links.
// Check within the same transaction as each affected import or deletion.
func CheckClientBillingImport(db ClientBillingImportReader) error {
	var exists int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='client_invoicing_information'").Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return nil
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM client_invoicing_information").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("job/client/contact import would replace records used by invoicing profiles")
	}
	return nil
}
