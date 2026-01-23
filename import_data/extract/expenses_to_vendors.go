package extract

import (
	"log"

	_ "github.com/marcboeker/go-duckdb" // Import DuckDB driver
)

// expensesToVendors normalizes the Expenses.parquet data by:
// 1. Creating a separate Vendors.parquet file with unique vendor records
// 2. Updating Expenses.parquet to reference vendors via foreign keys instead of names
//
// HYBRID ID RESOLUTION (similar to jobs_to_clients_and_contacts.go):
// When TurboVendors.parquet exists, vendor IDs are resolved using this priority:
//  1. EXACT NAME MATCH: If exactly ONE TurboVendor exists with the same name,
//     use that TurboVendor's ID. This preserves PocketBase IDs for vendors
//     that were written back from Turbo.
//  2. GENERATED ID: Otherwise, generate a deterministic ID from MD5 hash of name.
//
// This ensures:
// - Turbo-originated vendors keep their PocketBase IDs on re-import
// - Legacy-only vendors get consistent generated IDs
// - Expenses link to the correct vendor IDs
func expensesToVendors() {

	db, err := openDuckDB()
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create deterministic ID generation macro in DuckDB
	_, err = db.Exec(`
CREATE OR REPLACE MACRO make_pocketbase_id(source_value, length)
AS substr(md5(CAST(source_value AS VARCHAR)), 1, length);
`)
	if err != nil {
		log.Fatalf("Failed to create make_pocketbase_id macro: %v", err)
	}

	// Check if TurboVendors.parquet exists for hybrid ID resolution
	turboVendorsExists := fileExists("parquet/TurboVendors.parquet")

	if turboVendorsExists {
		log.Println("TurboVendors.parquet found - will use hybrid ID resolution for vendors")
	} else {
		log.Println("TurboVendors.parquet not found - will generate all vendor IDs from names")
	}

	query := buildVendorQuery(turboVendorsExists)

	_, err = db.Exec(query)
	if err != nil {
		log.Fatalf("Failed to execute vendor extraction query: %v", err)
	}
}

func buildVendorQuery(turboVendorsExists bool) string {
	query := `
		-- Load Expenses.parquet into a table called expenses
		CREATE TABLE expenses AS
		SELECT *, TRIM(vendorName) AS t_vendor FROM read_parquet('parquet/Expenses.parquet');
`

	if turboVendorsExists {
		query += `
		-- Load TurboVendors.parquet as lookup table for preserved IDs
		CREATE TABLE turbo_vendors AS
		SELECT * FROM read_parquet('parquet/TurboVendors.parquet');

		-- Create the vendors table with hybrid ID resolution
		-- Priority 1: Exact name match against TurboVendors (if exactly 1 match)
		-- Priority 2: Generate ID from MD5 hash of name
		CREATE TABLE vendors AS
		SELECT
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_vendors tv WHERE tv.name = vr.name) = 1
				THEN (SELECT tv.id FROM turbo_vendors tv WHERE tv.name = vr.name)
				ELSE make_pocketbase_id(vr.name, 15)
			END AS id,
			vr.name,
			-- Pull alias and status from TurboVendors when matched
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_vendors tv WHERE tv.name = vr.name) = 1
				THEN (SELECT tv.alias FROM turbo_vendors tv WHERE tv.name = vr.name)
				ELSE NULL
			END AS alias,
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_vendors tv WHERE tv.name = vr.name) = 1
				THEN (SELECT tv.status FROM turbo_vendors tv WHERE tv.name = vr.name)
				ELSE 'Active'
			END AS status
		FROM (
			SELECT DISTINCT t_vendor AS name FROM expenses WHERE t_vendor IS NOT NULL AND t_vendor != ''
			ORDER BY name
		) vr;

		-- Also include TurboVendors that don't appear in any expenses (Turbo-only vendors)
		INSERT INTO vendors (id, name, alias, status)
		SELECT tv.id, tv.name, tv.alias, tv.status
		FROM turbo_vendors tv
		WHERE NOT EXISTS (SELECT 1 FROM vendors v WHERE v.id = tv.id);
`
	} else {
		query += `
		-- Create the vendors table (no TurboVendors - generate all IDs from names)
		CREATE TABLE vendors AS
		SELECT
			make_pocketbase_id(name, 15) AS id,
			name,
			NULL AS alias,
			'Active' AS status
		FROM (
			SELECT DISTINCT t_vendor AS name FROM expenses WHERE t_vendor IS NOT NULL AND t_vendor != ''
			ORDER BY name
		);
`
	}

	query += `
		COPY vendors TO 'parquet/Vendors.parquet' (FORMAT PARQUET);

		-- Update the expenses table to use the new vendor id instead of the old vendor column
		ALTER TABLE expenses ADD COLUMN vendor_id string;
		UPDATE expenses SET vendor_id = vendors.id FROM vendors WHERE expenses.t_vendor = vendors.name;

		COPY expenses TO 'parquet/Expenses.parquet' (FORMAT PARQUET);
`

	return query
}
