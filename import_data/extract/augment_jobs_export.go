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

	// Load base tables from Parquet
	_, err = db.Exec("CREATE TABLE jobs_raw AS SELECT * FROM read_parquet('parquet/Jobs.parquet')")
	if err != nil {
		log.Fatalf("Failed to load Jobs.parquet: %v", err)
	}
	_, err = db.Exec("CREATE TABLE profiles AS SELECT * FROM read_parquet('parquet/Profiles.parquet')")
	if err != nil {
		log.Fatalf("Failed to load Profiles.parquet: %v", err)
	}
	_, err = db.Exec("CREATE TABLE divisions AS SELECT * FROM read_parquet('parquet/divisions.parquet')") // Load divisions
	if err != nil {
		log.Fatalf("Failed to load divisions.parquet: %v", err)
	}

	// fold in manager_id
	_, err = db.Exec("CREATE TABLE jobsA AS SELECT jobs_raw.*, profiles.pocketbase_uid AS manager_id FROM jobs_raw LEFT JOIN profiles ON jobs_raw.managerUid = profiles.id")
	if err != nil {
		log.Fatalf("Failed to create jobsA: %v", err)
	}

	// fold in alternate_manager_id
	_, err = db.Exec("CREATE TABLE jobsB AS SELECT jobsA.*, profiles.pocketbase_uid AS alternate_manager_id FROM jobsA LEFT JOIN profiles ON jobsA.alternateManagerUid = profiles.id")
	if err != nil {
		log.Fatalf("Failed to create jobsB: %v", err)
	}

	// fold in the proposal_id
	_, err = db.Exec("CREATE TABLE jobsC AS SELECT jobsB.*, proposals.pocketbase_id AS proposal_id FROM jobsB LEFT JOIN jobsB proposals ON jobsB.proposal = proposals.id")
	if err != nil {
		log.Fatalf("Failed to create jobsC: %v", err)
	}

	// Unnest division codes into a temporary table
	_, err = db.Exec(`
	CREATE TEMP TABLE jobsC_unnested AS
	SELECT
	    jobsC.id as job_id, -- Assuming 'id' is the unique job identifier from jobs_raw
	    trim(unnested_code) as division_code
	FROM jobsC, unnest(str_split(jobsC.divisions, ',')) AS t(unnested_code)
	WHERE jobsC.divisions IS NOT NULL AND jobsC.divisions != '';
	`)
	if err != nil {
		log.Fatalf("Failed to create jobsC_unnested: %v", err)
	}

	// Aggregate division IDs per job into a temporary table
	_, err = db.Exec(`
	CREATE TEMP TABLE division_id_lists AS
	SELECT
	    unnested.job_id,
	    list_filter(list(div.id), x -> x IS NOT NULL) AS division_id_list -- Use 'id' from divisions table
	FROM
	    jobsC_unnested unnested
	JOIN divisions div
	    ON unnested.division_code = div.code
	GROUP BY unnested.job_id;
	`)
	if err != nil {
		log.Fatalf("Failed to create division_id_lists: %v", err)
	}

	// Unnest category strings
	_, err = db.Exec(`
	CREATE TEMP TABLE categories_unnested AS
	SELECT
	    jobsC.id as job_id,
	    trim(unnested_category) as category_string
	FROM jobsC, unnest(str_split(jobsC.categories, ',')) AS t(unnested_category)
	WHERE jobsC.categories IS NOT NULL AND jobsC.categories != '';
	`)
	if err != nil {
		log.Fatalf("Failed to create categories_unnested: %v", err)
	}

	// Aggregate category strings per job
	_, err = db.Exec(`
	CREATE TEMP TABLE category_lists AS
	SELECT
	    unnested.job_id,
	    list(unnested.category_string) AS category_list
	FROM
	    categories_unnested unnested
	GROUP BY unnested.job_id;
	`)
	if err != nil {
		log.Fatalf("Failed to create category_lists: %v", err)
	}

	// Join aggregated division IDs and categories back and convert to JSON
	_, err = db.Exec(`
	CREATE TABLE jobsD AS
	SELECT
	    jobsC.* EXCLUDE (divisions, categories), -- Exclude original comma-separated fields
	    COALESCE(to_json(div_agg.division_id_list), '[]') AS divisions_ids,
	    COALESCE(to_json(cat_agg.category_list), '[]') AS categories -- Add categories as JSON list
	FROM
	    jobsC
	LEFT JOIN division_id_lists div_agg
	    ON jobsC.id = div_agg.job_id
	LEFT JOIN category_lists cat_agg
	    ON jobsC.id = cat_agg.job_id;
	`)
	if err != nil {
		log.Fatalf("Failed to create jobsD: %v", err)
	}

	// overwrite the jobs table with the final augmented table
	_, err = db.Exec("COPY jobsD TO 'parquet/Jobs.parquet' (FORMAT PARQUET)") // Output to Jobs.parquet
	if err != nil {
		log.Fatalf("Failed to copy jobsD to Parquet: %v", err)
	}
}
