package extract

import (
	"database/sql"
	"log"

	_ "github.com/marcboeker/go-duckdb" // Import DuckDB driver
)

// The function uses DuckDB to normalize the Expenses.parquet data by:
// 1. Creating a separate Vendors.parquet file with unique vendor records and PocketBase formatted ids
// 2. Updating Expenses.parquet to reference vendors via foreign keys instead of names
func expensesToVendors() {

	db, err := sql.Open("duckdb", "")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Create random_string() UDF in DuckDB
	_, err = db.Exec(`
CREATE OR REPLACE MACRO make_pocketbase_id(length)
AS array_to_string(array_slice(array_apply(range(length), i -> CASE WHEN random() < 0.72 THEN chr(CAST(floor(random() * 26) + 97 AS INTEGER)) ELSE CAST(CAST(floor(random() * 10) AS INTEGER) AS VARCHAR) END), 1, length), '');
`)
	if err != nil {
		log.Fatalf("Failed to create random_string() UDF: %v", err)
	}

	defer db.Close()

	splitQuery := `
		-- Load Expenses.parquet into a table called expenses
		CREATE TABLE expenses AS
		SELECT *, TRIM(vendorName) AS t_vendor FROM read_parquet('parquet/Expenses.parquet');

		-- Create the vendors table, removing duplicate names
		CREATE TABLE vendors AS
		SELECT make_pocketbase_id(15) AS id, name
		FROM (
		    SELECT DISTINCT t_vendor AS name FROM expenses WHERE t_vendor IS NOT NULL AND t_vendor != ''
		);

		COPY vendors TO 'parquet/Vendors.parquet' (FORMAT PARQUET);

		-- Update the expenses table to use the new vendor id instead of the old vendor column
		ALTER TABLE expenses ADD COLUMN vendor_id string;

		UPDATE expenses SET vendor_id = vendors.id FROM vendors WHERE expenses.t_vendor = vendors.name;

		-- ALTER TABLE expenses DROP vendor;
		-- ALTER TABLE expenses DROP t_vendor;
		-- ALTER TABLE expenses RENAME vendor_id TO vendor;

		COPY expenses TO 'parquet/Expenses.parquet' (FORMAT PARQUET);`

	_, err = db.Exec(splitQuery)
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}
}
