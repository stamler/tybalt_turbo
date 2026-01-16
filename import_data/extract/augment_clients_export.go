package extract

import (
	"log"
)

// augmentClients resolves businessDevelopmentLeadUid (legacy Firebase UID) to the
// PocketBase UID by joining Clients.parquet with Profiles.parquet.
// This is done during export so the import doesn't depend on admin_profiles
// existing in the destination database.
func augmentClients() {
	db, err := openDuckDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Load Clients.parquet and Profiles.parquet, join to resolve the UID,
	// then write back to Clients.parquet with the resolved business_development_lead column.
	_, err = db.Exec(`
		-- Load base tables
		CREATE TABLE clients_raw AS SELECT * FROM read_parquet('parquet/Clients.parquet');
		CREATE TABLE profiles AS SELECT * FROM read_parquet('parquet/Profiles.parquet');

		-- Join to resolve businessDevelopmentLeadUid -> pocketbase_uid
		-- Keep businessDevelopmentLeadUid for reference, add resolved business_development_lead
		CREATE TABLE clients_augmented AS
		SELECT 
			c.id,
			c.name,
			COALESCE(p.pocketbase_uid, '') AS business_development_lead
		FROM clients_raw c
		LEFT JOIN profiles p ON c.businessDevelopmentLeadUid = p.id 
			AND c.businessDevelopmentLeadUid IS NOT NULL 
			AND c.businessDevelopmentLeadUid != '';

		-- Overwrite Clients.parquet with augmented data
		COPY clients_augmented TO 'parquet/Clients.parquet' (FORMAT PARQUET);
	`)
	if err != nil {
		log.Fatalf("Failed to augment clients: %v", err)
	}
}
