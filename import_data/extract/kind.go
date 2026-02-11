package extract

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// standardExpenditureKindID is set by ToParquet before running export steps so
// that SQL uses the target DB's "standard" kind ID. Also used by tool.go for
// normalizeExpenditureKindID when importing.
var standardExpenditureKindID string

// GetStandardExpenditureKindID resolves the expenditure_kinds.id for the
// 'standard' kind name from the given SQLite DB.
func GetStandardExpenditureKindID(dbPath string) (string, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return "", fmt.Errorf("open DB %q: %w", dbPath, err)
	}
	defer db.Close()

	var id string
	err = db.QueryRow("SELECT id FROM expenditure_kinds WHERE name = 'standard' LIMIT 1").Scan(&id)
	if err != nil {
		return "", fmt.Errorf("query expenditure_kinds name='standard' in %q: %w", dbPath, err)
	}
	return id, nil
}

// StandardExpenditureKindID returns the current standard kind ID. Must be
// called after the ID has been resolved via GetStandardExpenditureKindID.
func StandardExpenditureKindID() string {
	return standardExpenditureKindID
}
