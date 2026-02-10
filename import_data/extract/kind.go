package extract

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

// Fallback when DB lookup fails (e.g. DB not yet initialized).
const fallbackStandardExpenditureKindID = "l3vtlbqg529m52j"

// standardExpenditureKindID is set by ToParquet before running export steps so
// that SQL uses the target DB's "standard" kind ID. Also used by tool.go for
// normalizeExpenditureKindID when importing.
var standardExpenditureKindID = fallbackStandardExpenditureKindID

// GetStandardExpenditureKindID returns the expenditure_kinds.id for name
// 'standard' from the given SQLite DB, or the fallback constant if the lookup
// fails.
func GetStandardExpenditureKindID(dbPath string) string {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Printf("GetStandardExpenditureKindID: open DB %q: %v; using fallback", dbPath, err)
		return fallbackStandardExpenditureKindID
	}
	defer db.Close()

	var id string
	err = db.QueryRow("SELECT id FROM expenditure_kinds WHERE name = 'standard' LIMIT 1").Scan(&id)
	if err != nil {
		log.Printf("GetStandardExpenditureKindID: query %q: %v; using fallback", dbPath, err)
		return fallbackStandardExpenditureKindID
	}
	return id
}

// StandardExpenditureKindID returns the current standard kind ID (set by
// ToParquet during export, or fallback). Used by augment_expenses,
// expenses_to_pos, and export_to_parquet.
func StandardExpenditureKindID() string {
	return standardExpenditureKindID
}
