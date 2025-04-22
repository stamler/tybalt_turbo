package load

import (
	"fmt"
	"log"

	"github.com/parquet-go/parquet-go"
	"github.com/pocketbase/dbx"
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
	Id   string `parquet:"id"`
	Name string `parquet:"name"`
}

type ClientContact struct {
	Id        string `parquet:"id"`
	Surname   string `parquet:"surname"`
	GivenName string `parquet:"givenName"`
	Client    string `parquet:"client_id"`
}

type Job struct {
	Number                      string `parquet:"id"`
	AlternateManagerDisplayName string `parquet:"alternateManagerDisplayName"`
	AlternateManagerUid         string `parquet:"alternateManagerUid"`
	Categories                  string `parquet:"categories"`
	ClientName                  string `parquet:"client"`
	ClientContact               string `parquet:"clientContact"`
	Description                 string `parquet:"description"`
	Divisions                   string `parquet:"divisions"`
	DivisionsIds                string `parquet:"divisions_ids"`
	FnAgreement                 bool   `parquet:"fnAgreement"`
	HasTimeEntries              bool   `parquet:"hasTimeEntries"`
	JobOwner                    string `parquet:"jobOwner"`
	JobOwnerId                  string `parquet:"job_owner_id"`
	LastTimeEntryDate           string `parquet:"lastTimeEntryDate"`
	ManagerName                 string `parquet:"manager"`
	ManagerDisplayName          string `parquet:"managerDisplayName"`
	ManagerUid                  string `parquet:"managerUid"`
	ProjectAwardDate            string `parquet:"projectAwardDate"`
	Proposal                    string `parquet:"proposal"`
	ProposalId                  string `parquet:"proposal_id"`
	ProposalOpeningDate         string `parquet:"proposalOpeningDate"`
	ProposalSubmissionDueDate   string `parquet:"proposalSubmissionDueDate"`
	Status                      string `parquet:"status"`
	Timestamp                   string `parquet:"timestamp"`
	Id                          string `parquet:"pocketbase_id"`
	TClient                     string `parquet:"t_client"`
	TClientContact              string `parquet:"t_clientContact"`
	Client                      string `parquet:"client_id"`
	Contact                     string `parquet:"contact_id"`
	Manager                     string `parquet:"manager_id"`
	AlternateManagerId          string `parquet:"alternate_manager_id"`
}

type Profile struct {
	PocketbaseId     string `parquet:"pocketbase_id"`
	PocketbaseUserId string `parquet:"pocketbase_uid"`
	Email            string `parquet:"email"`
	Surname          string `parquet:"surname"`
	GivenName        string `parquet:"givenName"`
}

type Category struct {
	Id   string `parquet:"id"`
	Name string `parquet:"name"`
	Job  string `parquet:"job"`
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
	items, err := parquet.ReadFile[T](parquetFilePath)
	if err != nil {
		panic(err)
	}

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
