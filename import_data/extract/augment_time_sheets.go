package extract

import (
	"fmt"
	"log"
)

func augmentTimeSheets() {
	db, err := openDuckDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Define PocketBase-like ID generation macro using deterministic hash
	_, err = db.Exec(`
CREATE OR REPLACE MACRO make_pocketbase_id(source_value, length)
AS substr(md5(CAST(source_value AS VARCHAR)), 1, length);
`)
	if err != nil {
		log.Fatalf("Failed to create make_pocketbase_id macro: %v", err)
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

	// Load uid_replacements.csv into a duckdb table and replace old uids with new uids in the time_sheets table
	_, err = db.Exec("CREATE TABLE uid_replacements AS SELECT * FROM read_csv('uid_replacements.csv')")
	if err != nil {
		log.Fatalf("Failed to create uid_replacements: %v", err)
	}

	// Replace any instances of the previousUid column value with the currentUid column value in the time_sheets table
	// This must be done before the fold in so the uids are correct for the approver and committer
	_, err = db.Exec(`
		UPDATE time_sheets
		SET uid = (SELECT currentUid FROM uid_replacements WHERE previousUid = time_sheets.uid)
		WHERE EXISTS (SELECT 1 FROM uid_replacements WHERE previousUid = time_sheets.uid)
	`)
	if err != nil {
		log.Fatalf("Failed to replace uids: %v", err)
	}

	// fold in pocketbase_uid and pocketbase_approver_uid
	_, err = db.Exec("CREATE TABLE time_sheetsA AS SELECT time_sheets.*, pa.pocketbase_uid AS pocketbase_uid, pb.pocketbase_uid AS pocketbase_approver_uid FROM time_sheets LEFT JOIN profiles pa ON time_sheets.uid = pa.id LEFT JOIN profiles pb ON time_sheets.managerUid = pb.id")
	if err != nil {
		log.Fatalf("Failed to create time_sheetsA: %v", err)
	}

	// Fail fast if any time sheet is missing a uid (always required).
	missingUidRows, err := db.Query("SELECT DISTINCT id FROM time_sheetsA WHERE uid IS NULL OR uid = ''")
	if err != nil {
		log.Fatalf("Failed to query time_sheetsA missing uid: %v", err)
	}
	defer missingUidRows.Close()

	var missingUids []string
	for missingUidRows.Next() {
		var id string
		if err := missingUidRows.Scan(&id); err != nil {
			log.Fatalf("Failed to scan missing uid row: %v", err)
		}
		missingUids = append(missingUids, id)
	}

	if err = missingUidRows.Err(); err != nil {
		log.Fatalf("Error iterating missing uid rows: %v", err)
	}

	if len(missingUids) > 0 {
		log.Println("TimeSheets with missing uid values")
		for _, missing := range missingUids {
			log.Println(missing)
		}
		log.Fatal("TimeSheets must reference a valid uid; fix source data and rerun this script.")
	}

	// Look for missing pocketbase_uid mappings for existing uids.
	rows, err := db.Query("SELECT DISTINCT uid, givenName, surname FROM time_sheetsA WHERE uid IS NOT NULL AND uid != '' AND pocketbase_uid IS NULL")
	if err != nil {
		log.Fatalf("Failed to query time_sheetsA: %v", err)
	}
	defer rows.Close()

	var missingUIDs []string

	for rows.Next() {
		var uid, givenName, surname string
		err = rows.Scan(&uid, &givenName, &surname)
		if err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		missingUIDs = append(missingUIDs, fmt.Sprintf("%s: %s %s", uid, givenName, surname))
	}

	if err = rows.Err(); err != nil {
		log.Fatalf("Error iterating rows: %v", err)
	}

	if len(missingUIDs) > 0 {
		log.Println("Missing pocketbase_uid values for TimeSheets.parquet")
		for _, missing := range missingUIDs {
			log.Println(missing)
		}
		log.Fatal("Please update uid_replacements.csv with the missing pocketbase_uid values and rerun this script. Also verify that the defaultDivision of the problematic profiles is set to an existing division in the sqlite database.")
	}

	// Fail fast when a time sheet has an approver UID that cannot map to PocketBase.
	approverRows, err := db.Query("SELECT DISTINCT id, managerUid FROM time_sheetsA WHERE managerUid IS NOT NULL AND managerUid != '' AND pocketbase_approver_uid IS NULL")
	if err != nil {
		log.Fatalf("Failed to query time_sheetsA approvers: %v", err)
	}
	defer approverRows.Close()

	var missingApproverUIDs []string
	for approverRows.Next() {
		var id, uid string
		err = approverRows.Scan(&id, &uid)
		if err != nil {
			log.Fatalf("Failed to scan approver row: %v", err)
		}
		missingApproverUIDs = append(missingApproverUIDs, fmt.Sprintf("%s: %s", id, uid))
	}

	if err = approverRows.Err(); err != nil {
		log.Fatalf("Error iterating approver rows: %v", err)
	}

	if len(missingApproverUIDs) > 0 {
		log.Println("Missing pocketbase_approver_uid values for TimeSheets.parquet")
		for _, missing := range missingApproverUIDs {
			log.Println(missing)
		}
		log.Fatal("Please update uid_replacements.csv with the missing pocketbase_uid values for time sheet approvers and rerun this script.")
	}

	// overwrite the timesheets table with the final augmented table
	_, err = db.Exec("COPY time_sheetsA TO 'parquet/TimeSheets.parquet' (FORMAT PARQUET)") // Output to TimeSheets.parquet
	if err != nil {
		log.Fatalf("Failed to copy timesheets to Parquet: %v", err)
	}

}
