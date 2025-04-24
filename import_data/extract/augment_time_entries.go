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
func augmentTimeEntries() {
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
	_, err = db.Exec("CREATE TABLE time_entries AS SELECT * FROM read_parquet('parquet/TimeEntries.parquet')")
	if err != nil {
		log.Fatalf("Failed to load TimeEntries.parquet: %v", err)
	}

	// Load uid_replacements.csv into a duckdb table and replace old uids with new uids in the time_entries table
	_, err = db.Exec("CREATE TABLE uid_replacements AS SELECT * FROM read_csv('uid_replacements.csv')")
	if err != nil {
		log.Fatalf("Failed to create uid_replacements: %v", err)
	}

	// Replace any instances of the previousUid column value with the currentUid column value in the time_entries table
	// This must be done before the fold in so the uids are correct
	_, err = db.Exec(`
		UPDATE time_entries
		SET uid = (SELECT currentUid FROM uid_replacements WHERE previousUid = time_entries.uid)
		WHERE EXISTS (SELECT 1 FROM uid_replacements WHERE previousUid = time_entries.uid)
	`)
	if err != nil {
		log.Fatalf("Failed to replace uids: %v", err)
	}

	// fold in pocketbase_uid and pocketbase_approver_uid
	_, err = db.Exec(`
		CREATE TABLE time_entriesA AS 
			SELECT time_entries.*, 
				p.pocketbase_uid AS pocketbase_uid,
				ts.pocketbase_id AS pocketbase_tsid,
				j.pocketbase_id AS pocketbase_jobid,
				tt.id AS timetype_id,
				d.id AS division_id
			FROM time_entries 
			LEFT JOIN profiles p ON time_entries.uid = p.id
			LEFT JOIN time_sheets ts ON time_entries.tsid = ts.id
			LEFT JOIN jobs j ON time_entries.job = j.id
			LEFT JOIN time_types tt ON time_entries.timetype = tt.code
			LEFT JOIN divisions d ON time_entries.division = d.code
	`)
	if err != nil {
		log.Fatalf("Failed to create time_entriesA: %v", err)
	}

	// overwrite the time_entries table with the final augmented table
	_, err = db.Exec("COPY time_entriesA TO 'parquet/TimeEntries.parquet' (FORMAT PARQUET)") // Output to TimeEntries.parquet
	if err != nil {
		log.Fatalf("Failed to copy time_entries to Parquet: %v", err)
	}

}
