package extract

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// capitalExpenditureKindID and projectExpenditureKindID are set by
// GetCapitalAndProjectKindIDs before running export/import steps.
var (
	capitalExpenditureKindID string
	projectExpenditureKindID string

	// Legacy alias: kept so StandardExpenditureKindID() still returns the
	// capital ID for backward-compat in callers that haven't been updated.
	standardExpenditureKindID string
)

// GetCapitalAndProjectKindIDs resolves both the 'capital' and 'project'
// expenditure kind IDs from the given SQLite DB. Falls back to 'standard'
// for capital when the DB hasn't been migrated yet.
func GetCapitalAndProjectKindIDs(dbPath string) (capitalID string, projectID string, err error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return "", "", fmt.Errorf("open DB %q: %w", dbPath, err)
	}
	defer db.Close()

	// Try capital first, fall back to standard.
	err = db.QueryRow("SELECT id FROM expenditure_kinds WHERE name = 'capital' LIMIT 1").Scan(&capitalID)
	if err != nil {
		// Fall back to legacy standard.
		err = db.QueryRow("SELECT id FROM expenditure_kinds WHERE name = 'standard' LIMIT 1").Scan(&capitalID)
		if err != nil {
			return "", "", fmt.Errorf("query expenditure_kinds for capital/standard in %q: %w", dbPath, err)
		}
	}

	// Project may not exist in pre-migration DBs.
	err = db.QueryRow("SELECT id FROM expenditure_kinds WHERE name = 'project' LIMIT 1").Scan(&projectID)
	if err != nil {
		// No project kind yet — use capital as fallback (pre-migration).
		projectID = capitalID
	}

	capitalExpenditureKindID = capitalID
	projectExpenditureKindID = projectID
	standardExpenditureKindID = capitalID
	return capitalID, projectID, nil
}

// GetStandardExpenditureKindID resolves the expenditure_kinds.id for the
// 'standard' (or 'capital') kind name from the given SQLite DB.
// Kept for backward-compat; delegates to GetCapitalAndProjectKindIDs.
func GetStandardExpenditureKindID(dbPath string) (string, error) {
	capitalID, _, err := GetCapitalAndProjectKindIDs(dbPath)
	return capitalID, err
}

// CapitalExpenditureKindID returns the current capital kind ID.
func CapitalExpenditureKindID() string {
	return capitalExpenditureKindID
}

// ProjectExpenditureKindID returns the current project kind ID.
func ProjectExpenditureKindID() string {
	return projectExpenditureKindID
}

// StandardExpenditureKindID returns the capital kind ID (legacy alias).
func StandardExpenditureKindID() string {
	return standardExpenditureKindID
}
