package extract

import (
	"fmt"
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

	// Fail fast when business development lead UIDs cannot map to PocketBase.
	rows, err := db.Query(`
		SELECT c.id, c.businessDevelopmentLeadUid
		FROM clients_raw c
		LEFT JOIN profiles p ON c.businessDevelopmentLeadUid = p.id
		WHERE c.businessDevelopmentLeadUid IS NOT NULL
		  AND c.businessDevelopmentLeadUid != ''
		  AND p.pocketbase_uid IS NULL
	`)
	if err != nil {
		log.Fatalf("Failed to query clients_raw: %v", err)
	}
	defer rows.Close()

	var missingUIDs []string
	for rows.Next() {
		var id, legacyUid string
		if err := rows.Scan(&id, &legacyUid); err != nil {
			log.Fatalf("Failed to scan clients_raw row: %v", err)
		}
		missingUIDs = append(missingUIDs, fmt.Sprintf("%s: %s", id, legacyUid))
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating clients_raw rows: %v", err)
	}

	if len(missingUIDs) > 0 {
		log.Println("Missing PocketBase UID mappings for Clients.parquet (businessDevelopmentLeadUid)")
		for _, missing := range missingUIDs {
			log.Println(missing)
		}
		log.Fatal("Please update uid_replacements.csv with the missing PocketBase UID mappings and rerun this script.")
	}
}
