package extract

import (
	"log"
)

// create pocketbase_uid column that references the pocketbase_uid column in Profiles.parquet
// create pocketbase_tsid column that references the pocketbase_id column in TimeSheets.parquet
// create job column that references the pocketbase_id column in Jobs.parquet by joining on id
// create timetype_id column that references the id column in time_types.parquet by joining on code
// create division_id column that references the id column in divisions.parquet by joining on code
func augmentExpenses() {
	db, err := openDuckDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Load base tables from Parquet
	_, err = db.Exec("CREATE TABLE profiles AS SELECT * FROM read_parquet('parquet/Profiles.parquet')")
	if err != nil {
		log.Fatalf("Failed to load Profiles.parquet: %v", err)
	}
	_, err = db.Exec("CREATE TABLE jobs AS SELECT * FROM read_parquet('parquet/Jobs.parquet')")
	if err != nil {
		log.Fatalf("Failed to load Jobs.parquet: %v", err)
	}
	_, err = db.Exec("CREATE TABLE divisions AS SELECT * FROM read_parquet('parquet/divisions.parquet')")
	if err != nil {
		log.Fatalf("Failed to load divisions.parquet: %v", err)
	}
	_, err = db.Exec("CREATE TABLE expenses AS SELECT * FROM read_parquet('parquet/Expenses.parquet')")
	if err != nil {
		log.Fatalf("Failed to load Expenses.parquet: %v", err)
	}
	_, err = db.Exec("CREATE TABLE categories AS SELECT * FROM read_parquet('parquet/Categories.parquet')")
	if err != nil {
		log.Fatalf("Failed to load Categories.parquet: %v", err)
	}

	// Load uid_replacements.csv into a duckdb table and replace old uids with new uids in the expenses table
	_, err = db.Exec("CREATE TABLE uid_replacements AS SELECT * FROM read_csv('uid_replacements.csv')")
	if err != nil {
		log.Fatalf("Failed to create uid_replacements: %v", err)
	}

	// Replace any instances of the previousUid column value with the currentUid column value in the expenses table
	// This must be done before the fold in so the uids are correct
	_, err = db.Exec(`
		UPDATE expenses
		SET uid = (SELECT currentUid FROM uid_replacements WHERE previousUid = expenses.uid)
		WHERE EXISTS (SELECT 1 FROM uid_replacements WHERE previousUid = expenses.uid)
	`)
	if err != nil {
		log.Fatalf("Failed to replace uids: %v", err)
	}

	// add the ccLast4Digits_string column which is a blank string if ccLast4Digits is null
	// and the value of ccLast4Digits left padded with zeros to 4 digits if it is not null
	_, err = db.Exec(`
		ALTER TABLE expenses ADD COLUMN ccLast4Digits_string TEXT
	`)
	if err != nil {
		log.Fatalf("Failed to add ccLast4Digits_string: %v", err)
	}
	_, err = db.Exec(`
		UPDATE expenses
		SET ccLast4Digits_string = LPAD(CAST(ccLast4Digits AS TEXT), 4, '0')
		WHERE ccLast4Digits IS NOT NULL
	`)
	if err != nil {
		log.Fatalf("Failed to pad ccLast4Digits: %v", err)
	}

	// fold in pocketbase_uid and pocketbase_approver_uid
	_, err = db.Exec(`
		CREATE TABLE expensesA AS 
			SELECT expenses.*, 
				p.pocketbase_uid AS pocketbase_uid,
				p2.pocketbase_uid AS pocketbase_approver_uid,
				p3.pocketbase_uid AS pocketbase_commit_uid,
				j.pocketbase_id AS pocketbase_jobid,
				d.id AS division_id
			FROM expenses 
			LEFT JOIN profiles p ON expenses.uid = p.id
			LEFT JOIN profiles p2 ON expenses.managerUid = p2.id
			LEFT JOIN profiles p3 ON expenses.commitUid = p3.id
			LEFT JOIN jobs j ON expenses.job = j.id
			LEFT JOIN divisions d ON expenses.division = d.code
	`)
	if err != nil {
		log.Fatalf("Failed to create expensesA: %v", err)
	}

	// augment expensesA with category_id
	_, err = db.Exec(`
		CREATE TABLE expensesB AS
			SELECT expensesA.*,
				c.id AS category_id
			FROM expensesA
			LEFT JOIN categories c ON expensesA.category = c.name AND expensesA.pocketbase_jobid = c.job
	`)
	if err != nil {
		log.Fatalf("Failed to create expensesB: %v", err)
	}

	// overwrite the expenses table with the final augmented table
	_, err = db.Exec("COPY expensesB TO 'parquet/Expenses.parquet' (FORMAT PARQUET)") // Output to Expenses.parquet
	if err != nil {
		log.Fatalf("Failed to copy expenses to Parquet: %v", err)
	}

}
