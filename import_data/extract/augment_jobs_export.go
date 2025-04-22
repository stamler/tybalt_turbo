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

	db.Exec("CREATE TABLE jobs AS SELECT * FROM read_parquet('parquet/Jobs.parquet')")
	db.Exec("CREATE TABLE profiles AS SELECT * FROM read_parquet('parquet/Profiles.parquet')")

	db.Exec("CREATE TABLE jobs_with_profiles AS SELECT jobs.*, profiles.pocketbase_uid AS manager_id FROM jobs LEFT JOIN profiles ON jobs.managerUid = profiles.id")

	// fold in alternate_manager_id
	db.Exec("CREATE TABLE jobsB AS SELECT jobs_with_profiles.*, profiles.pocketbase_uid AS alternate_manager_id FROM jobs_with_profiles LEFT JOIN profiles ON jobs_with_profiles.alternateManagerUid = profiles.id")

	// fold in the proposal_id
	db.Exec("CREATE TABLE jobsC AS SELECT jobsB.*, proposals.pocketbase_id AS proposal_id FROM jobsB LEFT JOIN jobsB proposals ON jobsB.proposal = proposals.id")

	// overwrite the jobs table with the jobs_with_profiles table
	db.Exec("COPY jobsC TO 'parquet/Jobs.parquet' (FORMAT PARQUET)")

}
