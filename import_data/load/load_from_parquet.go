package load

import (
	"fmt"

	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/reader"
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

	// Count the number of rows in the parquet file
	numRows := int(pr.GetNumRows())
	fmt.Printf("%s has %d rows\n", parquetFilePath, numRows)

	// Read all rows into a slice of structs
	items := make([]Client, 0, numRows)

	const batchSize = 10
	for i := 0; i < numRows; i += batchSize {
		toRead := batchSize
		if numRows-i < batchSize {
			toRead = numRows - i
		}

		rows := make([]Client, toRead)
		if err = pr.Read(&rows); err != nil {
			panic(err)
		}
		items = append(items, rows...)
	}

	for _, item := range items {
		// TODO: write data into the corresponding collection in pocketbase, either
		// using the pocketbase client with a super admin token or building this
		// into the app itself as an endpoint that accepts a parquet file and a
		// target collection name.
		fmt.Printf("Client: %s, %s\n", item.Id, item.Name)
	}
}
