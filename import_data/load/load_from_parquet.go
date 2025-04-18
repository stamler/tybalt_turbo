package load

import (
	"fmt"
	"log"

	"github.com/pocketbase/dbx"
	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/reader"
	_ "modernc.org/sqlite" // Import modernc SQLite driver for side-effect registration
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

// --- Struct definitions for Parquet data ---
// TODO: Move these definitions to a more shared location if used outside this package.

// Client represents the schema for the Clients.parquet file.
type Client struct {
	Id   string `db:"id" parquet:"name=id, type=BYTE_ARRAY, encoding=PLAIN_DICTIONARY"`
	Name string `db:"name" parquet:"name=name, type=BYTE_ARRAY, encoding=PLAIN_DICTIONARY"`
}

type ClientContact struct {
	Id        string `db:"id" parquet:"name=id, type=BYTE_ARRAY, encoding=PLAIN_DICTIONARY"`
	Surname   string `db:"surname" parquet:"name=surname, type=BYTE_ARRAY, encoding=PLAIN_DICTIONARY"`
	GivenName string `db:"given_name" parquet:"name=givenName, type=BYTE_ARRAY, encoding=PLAIN_DICTIONARY"`

	Client string `db:"client" parquet:"name=client_id, type=BYTE_ARRAY, encoding=PLAIN_DICTIONARY"`
}

type Job struct {
	Id          string `db:"id" parquet:"name=pocketbase_id, type=BYTE_ARRAY, encoding=PLAIN_DICTIONARY"`
	Number      string `db:"number" parquet:"name=id, type=BYTE_ARRAY, encoding=PLAIN_DICTIONARY"`
	Description string `db:"description" parquet:"name=description, type=BYTE_ARRAY, encoding=PLAIN_DICTIONARY"`
	Client      string `db:"client" parquet:"name=client_id, type=BYTE_ARRAY, encoding=PLAIN_DICTIONARY"`
	Contact     string `db:"contact" parquet:"name=contact_id, type=BYTE_ARRAY, encoding=PLAIN_DICTIONARY"`
	Manager     string `db:"manager" parquet:"name=manager_id, type=BYTE_ARRAY, encoding=PLAIN_DICTIONARY"`
}

// TODO: Add struct definitions for other tables (Contacts, Jobs, etc.) here or elsewhere.

// FromParquet reads data from a Parquet file and inserts it into a SQLite table using a generic approach.
// T: The struct type corresponding to the Parquet file schema.
// parquetFilePath: Path to the input Parquet file.
// sqliteDBPath: Path to the target SQLite database file.
// sqliteTableName: Name of the target table (used for logging).
// insertSQL: The parameterized INSERT statement (e.g., "INSERT INTO table (col1, col2) VALUES ({:p1}, {:p2})").
// binder: A function that takes an item of type T and returns a dbx.Params map suitable for binding to insertSQL.
func FromParquet[T any](parquetFilePath string, sqliteDBPath string, sqliteTableName string, insertSQL string, binder func(item T) dbx.Params) {
	// --- Parquet Reading (Generic) ---
	// Open the Parquet file
	fr, err := local.NewLocalFileReader(parquetFilePath)
	if err != nil {
		panic(err)
	}
	defer fr.Close()

	// Create a Parquet reader
	pr, err := reader.NewParquetReader(fr, new(T), 4)
	if err != nil {
		panic(err)
	}
	defer pr.ReadStop()

	// Setup the destination slice
	numRows := int(pr.GetNumRows())
	items := make([]T, 0, numRows)

	// Read the data in batches
	const batchSize = 10
	for i := 0; i < numRows; i += batchSize {
		rows := make([]T, min(batchSize, numRows-i))
		if err = pr.Read(&rows); err != nil {
			panic(err)
		}
		items = append(items, rows...)
	}

	// Diagnostic print:
	fmt.Printf("Expected rows: %d, Actual items read: %d\n", numRows, len(items))

	// NOTE: Potential issue with parquet-go reader:
	// Observations during testing with Clients.parquet indicate that the first
	// element read into the items slice (items[0]) contains zero-value data
	// (e.g., "\uFFFD\uFFFD") instead of the actual first row from the Parquet file.
	// The actual first row appears in items[1]. This seems to occur specifically
	// during the first pr.Read(&rows) call. Subsequent reads and the total item
	// count appear correct. For now, we are not skipping items[0], but this may
	// need to be addressed if the garbage row causes issues during insertion.

	// connect to the sqlite database with the dbx package
	db, err := dbx.Open("sqlite", sqliteDBPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	fmt.Printf("Inserting %d items into %s...\n", len(items), sqliteTableName)
	insertedCount := 0
	failureCount := 0

	// Prepare the query once (more efficient)
	q := db.NewQuery(insertSQL)

	// Iterate through the items (still includes potentially bad first item)
	for _, item := range items {
		params := binder(item) // Use the provided binder function
		_, err := q.Bind(params).Execute()
		if err != nil {
			// Log error and continue? Or stop? For now, log and count failures.
			log.Printf("WARN: Failed to insert item into %s: %v. Item: %+v", sqliteTableName, err, item)
			failureCount++
			continue // Continue with the next item
		}
		insertedCount++
	}

	fmt.Printf("Finished insertion into %s: %d successful, %d failures\n", sqliteTableName, insertedCount, failureCount)

}
