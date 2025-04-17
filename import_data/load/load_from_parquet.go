package load

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/marcboeker/go-duckdb" // Import DuckDB driver
)

// TODO: shape the data into the target form then ATTACH the sqlite database
// and write the data to the corresponding tables in the sqlite database,
// honouring foreign key constraints and primary keys.

// https://duckdb.org/2024/01/26/multi-database-support-in-duckdb.html

/*
	Anticipated order of operations:
	1. Upload Clients.parquet to the sqlite database
	2. Upload Contacts.parquet to the sqlite database (these reference clients)
	3. Upload Jobs.parquet to the sqlite database (these reference clients and contacts)
	4. Upload Profiles.parquet to the sqlite database (these reference divisions and time types)
	5. Upload TimeSheets.parquet to the sqlite database (these reference profiles)
	6. Upload TimeEntries.parquet to the sqlite database (these reference timesheets, jobs, and profiles)
	7. Upload TimeAmendments.parquet to the sqlite database (these reference timesheets, jobs, divisions, time types, and profiles)
	8. Upload Expenses.parquet to the sqlite database (these reference jobs, profiles, and purchase orders) We may not do this because there aren't many purchase orders and we can archive the attachments.
*/

func FromParquet(parquetFilePath string, sqliteDBPath string, sqliteTableName string, columnNameMap map[string]string) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Install and Load the SQLite extension in DuckDB
	db.Exec(`
	INSTALL sqlite;
	LOAD sqlite;
	`)

	// Attach the SQLite database
	db.Exec(fmt.Sprintf(`
	ATTACH '%s' AS sqlite_db (TYPE sqlite);
	`, sqliteDBPath))

	// Now we copy each parquet file's data into the corresponding table in
	// sqlite_db, transforming the data and renaming columns as necessary.

	// ColumnNameMap is a map of the column names in the parquet file to the
	// column names in the sqlite table. So we need to create a string that uses
	// the 'AS' keyword to rename the columns in the parquet file to the column
	// names in the sqlite table.
	columnNameString := ""
	for parquetColumnName, sqliteColumnName := range columnNameMap {
		columnNameString += fmt.Sprintf("%s AS %s, ", parquetColumnName, sqliteColumnName)
	}

	insertStatement := fmt.Sprintf(`
	INSERT INTO sqlite_db.%s SELECT %s FROM read_parquet('%s');
	`, sqliteTableName, columnNameString, parquetFilePath)
	fmt.Println(insertStatement)
	db.Exec(insertStatement)

	// Now we need to create the foreign key constraints.
}
