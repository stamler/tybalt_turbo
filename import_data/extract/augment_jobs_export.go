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

	db.Exec("CREATE TABLE jobs_with_profiles AS SELECT jobs.*, profiles.pocketbase_uid AS manager_id FROM jobs JOIN profiles ON jobs.managerUid = profiles.id")

	// overwrite the jobs table with the jobs_with_profiles table
	db.Exec("COPY jobs_with_profiles TO 'parquet/Jobs.parquet' (FORMAT PARQUET)")

}
