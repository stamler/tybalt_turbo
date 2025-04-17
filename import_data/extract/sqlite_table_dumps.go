package extract

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath" // For creating paths

	_ "github.com/marcboeker/go-duckdb" // Import DuckDB driver (underscore means import for side effects)
)

func sqliteTableDumps(sqliteDBPath string, sqliteTableName string) {
	// --- Configuration ---
	outputParquetPath := "./parquet/" + sqliteTableName + ".parquet" // Path where the Parquet file will be saved
	sqliteDBAlias := "sqlite_db"                                     // Alias for the attached SQLite DB in DuckDB queries
	// ---------------------

	// Ensure output path is absolute or relative to the execution directory
	absParquetPath, err := filepath.Abs(outputParquetPath)
	if err != nil {
		log.Fatalf("Failed to get absolute path for output Parquet file: %v", err)
	}

	// 1. Connect to DuckDB (in-memory is sufficient for this task)
	// The DSN is empty because we don't need to persist the DuckDB instance itself.
	db, err := sql.Open("duckdb", "")
	if err != nil {
		log.Fatalf("Failed to connect to DuckDB: %v", err)
	}
	defer db.Close() // Ensure connection is closed when main finishes

	// 2. Install and Load the SQLite Extension
	// This needs to be done once per connection/session that needs SQLite access.
	_, err = db.Exec("INSTALL sqlite; LOAD sqlite;")
	if err != nil {
		// Note: If the extension is already installed globally or locally for DuckDB,
		// you might only need `LOAD sqlite;`. INSTALL ensures it's available.
		log.Fatalf("Failed to install/load SQLite extension: %v", err)
	}

	// 3. Attach the SQLite Database File
	// Use fmt.Sprintf to safely include the path and alias in the SQL query.
	attachQuery := fmt.Sprintf(
		"ATTACH '%s' AS %s (TYPE SQLITE, READ_ONLY);",
		sqliteDBPath, sqliteDBAlias,
	)
	_, err = db.Exec(attachQuery)
	if err != nil {
		log.Fatalf("Failed to attach SQLite database '%s': %v", sqliteDBPath, err)
	}

	// 4. Copy the table contents to a Parquet file
	// Select all columns (*) from the attached SQLite table and copy to the target Parquet file.
	copyQuery := fmt.Sprintf(
		"COPY (SELECT * FROM %s.%s) TO '%s' (FORMAT PARQUET);",
		sqliteDBAlias, sqliteTableName, absParquetPath, // Use absolute path here
	)
	_, err = db.Exec(copyQuery)
	if err != nil {
		log.Fatalf("Failed to copy table '%s' to Parquet file '%s': %v", sqliteTableName, absParquetPath, err)
	}
}
