package extract

import (
	"database/sql"
	"log"
)

// create pocketbase_uid column that references the pocketbase_uid column in Profiles.parquet
// create pocketbase_tsid column that references the pocketbase_id column in TimeSheets.parquet
// create job column that references the pocketbase_id column in Jobs.parquet by joining on id
// create timetype_id column that references the id column in time_types.parquet by joining on code
// create division_id column that references the id column in divisions.parquet by joining on code
func augmentTimeAmendments() {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

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
				ts.weekEnding AS week_ending,
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
	_, err = db.Exec("COPY time_amendmentsA TO 'parquet/TimeAmendments.parquet' (FORMAT PARQUET)") // Output to TimeAmendments.parquet
	if err != nil {
		log.Fatalf("Failed to copy time_amendments to Parquet: %v", err)
	}

}
