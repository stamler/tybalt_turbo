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
		SELECT * FROM read_parquet('parquet/Expenses.parquet');


		-- Normalize historical plain-digit PO numbers (4-5 digits) by prefixing YY- from the date
		-- - Trim whitespace and cast PO to text to handle numeric types and stray spaces
		-- - Only update rows where trimmed PO is exactly 4-5 digits and date is present
		UPDATE expenses
		SET po = concat(
			strftime(CAST(date AS DATE), '%y'),
			'-',
			TRIM(CAST(po AS VARCHAR))
		)
		WHERE po IS NOT NULL AND date IS NOT NULL
		  AND regexp_matches(TRIM(CAST(po AS VARCHAR)), '^[0-9]{4,5}$');

		-- Establish the first (earliest) expense row per PO for representative fields
		CREATE TABLE first_expense_per_po AS
		SELECT * FROM (
		  SELECT e.*,
		         row_number() OVER (
		             PARTITION BY po
		             ORDER BY date, pocketbase_id
		         ) AS rn
		  FROM expenses e
		  WHERE po IS NOT NULL AND po != ''
		) WHERE rn = 1;

		-- Sum totals per PO (raw units as in Expenses.parquet; conversion happens on import)
		CREATE TABLE sum_total_per_po AS
		SELECT po, CAST(SUM(total) AS DOUBLE) AS total
		FROM expenses
		WHERE po IS NOT NULL AND po != ''
		GROUP BY po;

		-- Create single purchase_order per PO with aggregated/representative fields
		CREATE TABLE purchase_orders AS
		SELECT
		  make_pocketbase_id(fe.po, 15) AS id,
		  fe.po AS number,
		  fe.pocketbase_approver_uid AS approver,
		  fe.date AS date,
		  fe.vendor_id AS vendor,
		  fe.pocketbase_uid AS uid,
		  st.total AS total,
		  fe.paymentType AS payment_type,
		  fe.pocketbase_jobid AS job,
		  fe.division_id AS division
		FROM first_expense_per_po fe
		JOIN sum_total_per_po st ON st.po = fe.po;

		-- Write the parquet used by the importer
		COPY purchase_orders TO 'parquet/purchase_orders.parquet' (FORMAT PARQUET);

		-- Update the expenses table to use the purchase_order id derived deterministically from PO
		ALTER TABLE expenses ADD COLUMN purchase_order_id string;
		UPDATE expenses
		SET purchase_order_id = make_pocketbase_id(po, 15)
		WHERE po IS NOT NULL AND po != '';

		-- Persist updated expenses parquet
		COPY expenses TO 'parquet/Expenses.parquet' (FORMAT PARQUET);`

	_, err = db.Exec(splitQuery)
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}
}
