package extract

import (
	"fmt"
	"log"
)

// This function will use DuckDB to take the divisions.parquet file and the
// profiles.parquet file and augment the profiles.parquet file with the division
// id from the divisions.parquet file joined on the code field (see
// descriptions/UserMigration.md for more details). It will also add the
// pocketbase_manager and pocketbase_alternateManager fields to the
// profiles.parquet file by joining the profiles.parquet file to itself on the
// managerUid and alternateManagerUid fields respectively and writing the joined
// values of pocketbase_uid to the pocketbase_manager and
// pocketbase_alternateManager fields. (also see descriptions/UserMigration.md
// for more details)
func augmentProfiles() {
	// open a connection to the duckdb database
	db, err := openDuckDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// load divisions.parquet and Profiles.parquet into duckdb
	db.Exec("CREATE TABLE divisions AS SELECT * FROM read_parquet('parquet/divisions.parquet')")
	db.Exec("CREATE TABLE profiles AS SELECT * FROM read_parquet('parquet/Profiles.parquet')")

	// join the divisions and profiles tables on the code field
	db.Exec("CREATE TABLE profiles_with_divisions AS SELECT profiles.*, divisions.id AS pocketbase_defaultDivision FROM profiles LEFT JOIN divisions ON profiles.defaultDivision = divisions.code")

	// now we join the profiles_with_divisions table to itself twice, once on the
	// managerUid field and once on the alternateManagerUid field and write the
	// joined values of pocketbase_uid to the pocketbase_manager and
	// pocketbase_alternateManager fields.
	db.Exec(`CREATE TABLE profiles_with_managers 
		AS SELECT pd.*, pdm.pocketbase_uid AS pocketbase_manager, pdam.pocketbase_uid AS pocketbase_alternateManager 
		FROM profiles_with_divisions pd 
		LEFT JOIN profiles_with_divisions pdm ON pd.managerUid = pdm.id
		LEFT JOIN profiles_with_divisions pdam ON pd.alternateManager = pdam.id
	`)

	// Fail fast when profile manager relationships cannot map to PocketBase UIDs.
	rows, err := db.Query(`
		SELECT pocketbase_id, 'managerUid' AS field, managerUid AS legacy_uid
		FROM profiles_with_managers
		WHERE managerUid IS NOT NULL AND managerUid != '' AND pocketbase_manager IS NULL
		UNION ALL
		SELECT pocketbase_id, 'alternateManager' AS field, alternateManager AS legacy_uid
		FROM profiles_with_managers
		WHERE alternateManager IS NOT NULL AND alternateManager != '' AND pocketbase_alternateManager IS NULL
	`)
	if err != nil {
		log.Fatalf("Failed to query profiles_with_managers: %v", err)
	}
	defer rows.Close()

	var missingUIDs []string
	for rows.Next() {
		var id, field, legacyUid string
		if err := rows.Scan(&id, &field, &legacyUid); err != nil {
			log.Fatalf("Failed to scan profiles_with_managers row: %v", err)
		}
		missingUIDs = append(missingUIDs, fmt.Sprintf("%s (%s): %s", id, field, legacyUid))
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating profiles_with_managers rows: %v", err)
	}

	if len(missingUIDs) > 0 {
		log.Println("Missing PocketBase UID mappings for Profiles.parquet")
		for _, missing := range missingUIDs {
			log.Println(missing)
		}
		log.Fatal("Please update uid_replacements.csv with the missing PocketBase UID mappings and rerun this script.")
	}

	// overwrite the Profiles.parquet file with the profiles_with_managers table
	db.Exec("COPY (SELECT * FROM profiles_with_managers) TO 'parquet/Profiles.parquet' (FORMAT PARQUET)")

	// drop everything from the database and close the connection
	db.Exec("DROP TABLE profiles_with_divisions")
	db.Exec("DROP TABLE profiles_with_managers")
	db.Exec("DROP TABLE divisions")
	db.Exec("DROP TABLE profiles")
}
