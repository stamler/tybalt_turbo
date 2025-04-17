package load

import (
	"fmt"

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

type Client struct {
	Id   string `db:"id" parquet:"name=id, type=BYTE_ARRAY, encoding=PLAIN_DICTIONARY"`
	Name string `db:"name" parquet:"name=name, type=BYTE_ARRAY, encoding=PLAIN_DICTIONARY"`
}

func FromParquet(parquetFilePath string, sqliteDBPath string, sqliteTableName string, columnNameMap map[string]string) {
	// Open the Parquet file
	fr, err := local.NewLocalFileReader(parquetFilePath)
	if err != nil {
		panic(err)
	}
	defer fr.Close()

	// Create a Parquet reader
	pr, err := reader.NewParquetReader(fr, new(Client), 4)
	if err != nil {
		panic(err)
	}
	defer pr.ReadStop()

	// Setup the destination slice
	numRows := int(pr.GetNumRows())
	items := make([]Client, 0, numRows)

	// Read the data in batches
	const batchSize = 10
	for i := 0; i < numRows; i += batchSize {
		rows := make([]Client, min(batchSize, numRows-i))
		if err = pr.Read(&rows); err != nil {
			panic(err)
		}
		// Diagnostic print for the first element of the first batch
		if i == 0 && len(rows) > 0 {
			fmt.Printf("First batch, first element: ID=%s, Name=%s\n", rows[0].Id, rows[0].Name)
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

	for _, item := range items {
		// TODO: write data into the corresponding collection in pocketbase, either
		// using the pocketbase client with a super admin token or building this
		// into the app itself as an endpoint that accepts a parquet file and a
		// target collection name.

		db.NewQuery("INSERT INTO clients (id, name) VALUES ({:id}, {:name})").Bind(dbx.Params{
			"id":   item.Id,
			"name": item.Name,
		}).Execute()
	}
}
