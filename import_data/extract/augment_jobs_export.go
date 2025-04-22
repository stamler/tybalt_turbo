package extract

import (
	"database/sql"
	"log"
)

func augmentJobs() {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.Exec("CREATE TABLE jobs_raw AS SELECT * FROM read_parquet('parquet/Jobs.parquet')")
	db.Exec("CREATE TABLE profiles AS SELECT * FROM read_parquet('parquet/Profiles.parquet')")
	db.Exec("CREATE TABLE divisions AS SELECT * FROM read_parquet('parquet/divisions.parquet')")

	// fold in manager_id
	db.Exec("CREATE TABLE jobsA AS SELECT jobs_raw.*, profiles.pocketbase_uid AS manager_id FROM jobs_raw LEFT JOIN profiles ON jobs_raw.managerUid = profiles.id")

	// fold in alternate_manager_id
	db.Exec("CREATE TABLE jobsB AS SELECT jobsA.*, profiles.pocketbase_uid AS alternate_manager_id FROM jobsA LEFT JOIN profiles ON jobsA.alternateManagerUid = profiles.id")

	// fold in the proposal_id
	db.Exec("CREATE TABLE jobsC AS SELECT jobsB.*, proposals.pocketbase_id AS proposal_id FROM jobsB LEFT JOIN jobsB proposals ON jobsB.proposal = proposals.id")

	// fold in the divisions_ids
	_, err = db.Exec(`
	CREATE TABLE jobsD AS
	SELECT
	    jobsC.*,
	    to_json(
	        CASE
	            WHEN jobsC.divisions IS NULL OR jobsC.divisions = '' THEN list_value() -- Handle empty/NULL divisions string
	            ELSE
	                list_filter( -- Remove NULLs resulting from codes not found
	                    list_transform(
	                        str_split(jobsC.divisions, ','), -- Split codes string into a list
	                        code -> (SELECT id FROM divisions WHERE divisions.code = trim(code)) -- Find ID for each code
	                    ),
	                    id -> id IS NOT NULL
	                )
	        END
	    ) AS divisions_ids
	FROM jobsC
	`)
	if err != nil {
		log.Fatal(err)
	}

	db.Exec("COPY jobsD TO 'parquet/JobsD.parquet' (FORMAT PARQUET)")
}
