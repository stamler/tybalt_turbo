package extract

import (
	"database/sql"
	"log"

	_ "github.com/marcboeker/go-duckdb" // Import DuckDB driver
)

// The function uses DuckDB to normalize the Expenses.parquet data by:
// 1. Creating a separate purchase_orders.parquet file with unique purchase_order records and PocketBase formatted ids
// 2. Updating Expenses.parquet to reference vendors via foreign keys instead of names
func expensesToPurchaseOrders() {

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
		SELECT *, regexp_replace(po, '^[^0-9]+|[^0-9]+$', '', 'g') AS t_po FROM read_parquet('parquet/Expenses.parquet');

		-- Create the purchase_orders table, removing duplicate names
		CREATE TABLE purchase_orders AS
		SELECT make_pocketbase_id(15) AS id, number
		FROM (
		    SELECT DISTINCT t_po AS number FROM expenses WHERE t_po IS NOT NULL AND t_po != ''
		);

		COPY purchase_orders TO 'parquet/purchase_orders.parquet' (FORMAT PARQUET);

		-- Update the expenses table to use the new purchase_order id instead of the old purchase_order column
		ALTER TABLE expenses ADD COLUMN purchase_order_id string;

		UPDATE expenses SET purchase_order_id = purchase_orders.id FROM purchase_orders WHERE expenses.t_po = purchase_orders.number;

		-- ALTER TABLE expenses DROP po;
		-- ALTER TABLE expenses DROP t_po;
		-- ALTER TABLE expenses RENAME purchase_order_id TO po;

		COPY expenses TO 'parquet/Expenses.parquet' (FORMAT PARQUET);`

	_, err = db.Exec(splitQuery)
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}
}
