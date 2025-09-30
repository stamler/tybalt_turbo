package extract

import (
	"log"

	_ "github.com/marcboeker/go-duckdb" // Import DuckDB driver
)

// The function uses DuckDB to normalize the Expenses.parquet data by:
// 1. Creating a separate Vendors.parquet file with unique vendor records and PocketBase formatted ids
// 2. Updating Expenses.parquet to reference vendors via foreign keys instead of names
func expensesToVendors() {

	db, err := openDuckDB()
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Create deterministic ID generation macro in DuckDB
	_, err = db.Exec(`
CREATE OR REPLACE MACRO make_pocketbase_id(source_value, length)
AS substr(md5(CAST(source_value AS VARCHAR)), 1, length);
`)
	if err != nil {
		log.Fatalf("Failed to create make_pocketbase_id macro: %v", err)
	}

	defer db.Close()

	splitQuery := `
		-- Load Expenses.parquet into a table called expenses
		CREATE TABLE expenses AS
		SELECT *, TRIM(vendorName) AS t_vendor FROM read_parquet('parquet/Expenses.parquet');

		-- Create the vendors table, removing duplicate names
		CREATE TABLE vendors AS
		SELECT make_pocketbase_id(name, 15) AS id, name
		FROM (
		    SELECT DISTINCT t_vendor AS name FROM expenses WHERE t_vendor IS NOT NULL AND t_vendor != ''
		    ORDER BY name
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
