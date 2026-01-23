package extract

import (
	"fmt"
	"log"
)

// create pocketbase_uid column that references the pocketbase_uid column in Profiles.parquet
// create pocketbase_tsid column that references the pocketbase_id column in TimeSheets.parquet
// create job column that references the pocketbase_id column in Jobs.parquet by joining on id
// create timetype_id column that references the id column in time_types.parquet by joining on code
// create division_id column that references the id column in divisions.parquet by joining on code
func augmentTimeAmendments() {
	db, err := openDuckDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Load the ICU extension
	_, err = db.Exec("INSTALL icu; LOAD icu;")
	if err != nil {
		// It might fail if already installed/loaded, potentially log this instead of Fatal
		log.Printf("Warning: Failed to install/load ICU extension (might be already available): %v", err)
	}

	// Load base tables from Parquet
	_, err = db.Exec("CREATE TABLE time_sheets AS SELECT * FROM read_parquet('parquet/TimeSheets.parquet')")
	if err != nil {
		log.Fatalf("Failed to load TimeSheets.parquet: %v", err)
	}
	_, err = db.Exec("CREATE TABLE profiles AS SELECT * FROM read_parquet('parquet/Profiles.parquet')")
	if err != nil {
		log.Fatalf("Failed to load Profiles.parquet: %v", err)
	}
	_, err = db.Exec("CREATE TABLE jobs AS SELECT * FROM read_parquet('parquet/Jobs.parquet')")
	if err != nil {
		log.Fatalf("Failed to load Jobs.parquet: %v", err)
	}
	_, err = db.Exec("CREATE TABLE time_types AS SELECT * FROM read_parquet('parquet/time_types.parquet')")
	if err != nil {
		log.Fatalf("Failed to load time_types.parquet: %v", err)
	}
	_, err = db.Exec("CREATE TABLE divisions AS SELECT * FROM read_parquet('parquet/divisions.parquet')")
	if err != nil {
		log.Fatalf("Failed to load divisions.parquet: %v", err)
	}
	_, err = db.Exec("CREATE TABLE time_amendments AS SELECT * FROM read_parquet('parquet/TimeAmendments.parquet')")
	if err != nil {
		log.Fatalf("Failed to load TimeAmendments.parquet: %v", err)
	}
	_, err = db.Exec("CREATE TABLE categories AS SELECT * FROM read_parquet('parquet/Categories.parquet')")
	if err != nil {
		log.Fatalf("Failed to load Categories.parquet: %v", err)
	}

	// Load uid_replacements.csv into a duckdb table and replace old uids with new uids in the time_amendments table
	_, err = db.Exec("CREATE TABLE uid_replacements AS SELECT * FROM read_csv('uid_replacements.csv')")
	if err != nil {
		log.Fatalf("Failed to create uid_replacements: %v", err)
	}

	// Replace any instances of the previousUid column value with the currentUid column value in the time_amendments table
	// This must be done before the fold in so the uids are correct
	_, err = db.Exec(`
		UPDATE time_amendments
		SET uid = (SELECT currentUid FROM uid_replacements WHERE previousUid = time_amendments.uid)
		WHERE EXISTS (SELECT 1 FROM uid_replacements WHERE previousUid = time_amendments.uid)
	`)
	if err != nil {
		log.Fatalf("Failed to replace uids: %v", err)
	}

	// Replace any instances of the previousUid column value with the currentUid column value in the time_amendments table
	// for the creator column
	_, err = db.Exec(`
		UPDATE time_amendments
		SET creator = (SELECT currentUid FROM uid_replacements WHERE previousUid = time_amendments.creator)
		WHERE EXISTS (SELECT 1 FROM uid_replacements WHERE previousUid = time_amendments.creator)
	`)
	if err != nil {
		log.Fatalf("Failed to replace creator uids: %v", err)
	}

	// Replace any instances of the previousUid column value with the currentUid column value in the time_amendments table
	// for the commitUid column
	_, err = db.Exec(`
		UPDATE time_amendments
		SET commitUid = (SELECT currentUid FROM uid_replacements WHERE previousUid = time_amendments.commitUid)
		WHERE EXISTS (SELECT 1 FROM uid_replacements WHERE previousUid = time_amendments.commitUid)
	`)
	if err != nil {
		log.Fatalf("Failed to replace commitUid uids: %v", err)
	}

	// fold in pocketbase_uid and pocketbase_approver_uid
	_, err = db.Exec(`
		CREATE TABLE time_amendmentsA AS 
			SELECT time_amendments.*, 
				p.pocketbase_uid AS pocketbase_uid,
				p2.pocketbase_uid AS pocketbase_commit_uid,
				p3.pocketbase_uid AS pocketbase_creator_uid,
				ts.pocketbase_id AS pocketbase_tsid,
				j.pocketbase_id AS pocketbase_jobid,
				tt.id AS timetype_id,
				d.id AS division_id,
			FROM time_amendments 
			LEFT JOIN profiles p ON time_amendments.uid = p.id
			LEFT JOIN profiles p2 ON time_amendments.commitUid = p2.id
			LEFT JOIN profiles p3 ON time_amendments.creator = p3.id
			LEFT JOIN time_sheets ts ON time_amendments.uid = ts.uid AND time_amendments.weekEnding = ts.weekEnding
			LEFT JOIN jobs j ON time_amendments.job = j.id
			LEFT JOIN time_types tt ON time_amendments.timetype = tt.code
			LEFT JOIN divisions d ON time_amendments.division = d.code
	`)
	if err != nil {
		log.Fatalf("Failed to create time_amendmentsA: %v", err)
	}

	// Fail fast when time amendments reference legacy UIDs that don't map to PocketBase.
	rows, err := db.Query(`
		SELECT pocketbase_id, 'uid' AS field, uid AS legacy_uid
		FROM time_amendmentsA
		WHERE uid IS NOT NULL AND uid != '' AND pocketbase_uid IS NULL
		UNION ALL
		SELECT pocketbase_id, 'creator' AS field, creator AS legacy_uid
		FROM time_amendmentsA
		WHERE creator IS NOT NULL AND creator != '' AND pocketbase_creator_uid IS NULL
		UNION ALL
		SELECT pocketbase_id, 'commitUid' AS field, commitUid AS legacy_uid
		FROM time_amendmentsA
		WHERE commitUid IS NOT NULL AND commitUid != '' AND pocketbase_commit_uid IS NULL
	`)
	if err != nil {
		log.Fatalf("Failed to query time_amendmentsA: %v", err)
	}
	defer rows.Close()

	var missingUIDs []string
	for rows.Next() {
		var id, field, legacyUid string
		err = rows.Scan(&id, &field, &legacyUid)
		if err != nil {
			log.Fatalf("Failed to scan time_amendmentsA row: %v", err)
		}
		missingUIDs = append(missingUIDs, fmt.Sprintf("%s (%s): %s", id, field, legacyUid))
	}

	if err = rows.Err(); err != nil {
		log.Fatalf("Error iterating time_amendmentsA rows: %v", err)
	}

	if len(missingUIDs) > 0 {
		log.Println("Missing PocketBase UID mappings for TimeAmendments.parquet")
		for _, missing := range missingUIDs {
			log.Println(missing)
		}
		log.Fatal("Please update uid_replacements.csv with the missing PocketBase UID mappings and rerun this script.")
	}

	// add commited and created_date columns by converting commitTime and created
	// timestamps from UTC to America/Thunder_Bay then format as YYYY-MM-DD
	_, err = db.Exec(`
		CREATE TABLE time_amendmentsB AS
			SELECT time_amendmentsA.*,
				strftime(timezone('America/Thunder_Bay', commitTime), '%Y-%m-%d') AS committed,
				strftime(timezone('America/Thunder_Bay', created), '%Y-%m-%d') AS created_date
			FROM time_amendmentsA
	`)
	if err != nil {
		log.Fatalf("Failed to create time_amendmentsB: %v", err)
	}

	// augment time_amendmentsA with category_id
	// COMMENTED OUT BECAUSE CATEGORIES ARE NOT PRESENT IN THE DATA
	// SHOULD THIS FEATURE BE CREATED? (CATEGORIES FOR AMENDMENTS)
	// _, err = db.Exec(`
	// 	CREATE TABLE time_amendmentsB AS
	// 		SELECT time_amendmentsA.*,
	// 			c.id AS category_id
	// 		FROM time_amendmentsA
	// 		LEFT JOIN categories c ON time_amendmentsA.category = c.name AND time_amendmentsA.pocketbase_jobid = c.job
	// `)
	// if err != nil {
	// 	log.Fatalf("Failed to create time_amendmentsB: %v", err)
	// }

	// overwrite the time_amendments table with the final augmented table
	_, err = db.Exec("COPY time_amendmentsB TO 'parquet/TimeAmendments.parquet' (FORMAT PARQUET)") // Output to TimeAmendments.parquet
	if err != nil {
		log.Fatalf("Failed to copy time_amendments to Parquet: %v", err)
	}

}
