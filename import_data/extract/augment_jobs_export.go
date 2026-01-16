package extract

import (
	"log"
)

func augmentJobs() {
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

	// Join aggregated division IDs back and convert to JSON, exclude original divisions field
	_, err = db.Exec(`
	CREATE TABLE jobsD AS
	SELECT
	    jobsC.* EXCLUDE (divisions), -- Exclude original comma-separated divisions
	    COALESCE(to_json(div_agg.division_id_list), '[]') AS divisions_ids
	FROM
	    jobsC
	LEFT JOIN division_id_lists div_agg
	    ON jobsC.id = div_agg.job_id;
	`)
	if err != nil {
		log.Fatalf("Failed to create jobsD: %v", err)
	}

	// Derive parent job ID for sub-jobs.
	// Sub-jobs have numbers like "16-105-3" or "P16-105-01" where the base job is "16-105" or "P16-105".
	// Base jobs have exactly one hyphen (e.g., "25-0123" or "P25-0123").
	// Sub-jobs have two or more hyphens. We only look for a parent when hyphen count > 1.
	// We extract the base number by removing the last "-XX" suffix and join back to find the parent's pocketbase_id.
	// COALESCE ensures empty string (not NULL) for PocketBase relation field compatibility.
	_, err = db.Exec(`
	CREATE TABLE jobsE AS
	SELECT
	    jobsD.*,
	    COALESCE(
	        CASE
	            WHEN length(jobsD.id) - length(replace(jobsD.id, '-', '')) > 1
	            THEN parent_job.pocketbase_id
	            ELSE NULL
	        END,
	        ''
	    ) AS parent_id
	FROM jobsD
	LEFT JOIN jobsD AS parent_job
	    ON regexp_replace(jobsD.id, '-[0-9]+$', '') = parent_job.id;
	`)
	if err != nil {
		log.Fatalf("Failed to create jobsE: %v", err)
	}

	// overwrite the jobs table with the final augmented table
	_, err = db.Exec("COPY jobsE TO 'parquet/Jobs.parquet' (FORMAT PARQUET)") // Output to Jobs.parquet
	if err != nil {
		log.Fatalf("Failed to copy jobsE to Parquet: %v", err)
	}

	// --- Create Categories Parquet ---

	// Unnest category strings separately
	_, err = db.Exec(`
	CREATE TEMP TABLE categories_extracted AS
	SELECT
	    jobsC.pocketbase_id as job_id, -- Use pocketbase_id instead of id
	    trim(unnested_category) as category_name
	FROM jobsC, unnest(str_split(jobsC.categories, ',')) AS t(unnested_category)
	WHERE jobsC.categories IS NOT NULL AND jobsC.categories != '';
	`)
	if err != nil {
		log.Fatalf("Failed to create categories_extracted: %v", err)
	}

	// Create the final categories table with new IDs
	_, err = db.Exec(`
	CREATE TABLE categories_export AS
	SELECT
	    make_pocketbase_id(CONCAT(category_name, '|', job_id), 15) AS id, -- Use the macro here with source value
	    category_name AS name,
	    job_id AS job
	FROM categories_extracted
	ORDER BY category_name, job_id;
	`)
	if err != nil {
		log.Fatalf("Failed to create categories_export: %v", err)
	}

	// Export categories to Parquet
	_, err = db.Exec("COPY categories_export TO 'parquet/Categories.parquet' (FORMAT PARQUET)")
	if err != nil {
		log.Fatalf("Failed to copy categories_export to Parquet: %v", err)
	}

	// --- Create JobTimeAllocations Parquet ---
	// Parse the jobTimeAllocations JSON field from MySQL Jobs table.
	// The JSON is an object mapping division codes to hours, e.g. {"ES": 100, "NRG": 50}.
	// We use json_each() to unnest the JSON object into key-value pairs,
	// then join with divisions to get the division ID from the code.
	_, err = db.Exec(`
	CREATE TEMP TABLE job_time_allocations_export AS
	SELECT
	  make_pocketbase_id(CONCAT(jobsC.pocketbase_id, '|', d.id), 15) AS id,
	  jobsC.pocketbase_id AS job,
	  d.id AS division,
	  COALESCE(CAST(jta.value AS DOUBLE), 0) AS hours
	FROM jobsC,
	LATERAL (SELECT * FROM json_each(jobsC.jobTimeAllocations)) AS jta
	JOIN divisions d ON jta.key = d.code
	WHERE jobsC.jobTimeAllocations IS NOT NULL 
	  AND jobsC.jobTimeAllocations != ''
	  AND jobsC.jobTimeAllocations != 'null'
	ORDER BY job, division;
	`)
	if err != nil {
		log.Fatalf("Failed to create job_time_allocations_export: %v", err)
	}

	_, err = db.Exec("COPY job_time_allocations_export TO 'parquet/JobTimeAllocations.parquet' (FORMAT PARQUET)")
	if err != nil {
		log.Fatalf("Failed to copy job_time_allocations_export to Parquet: %v", err)
	}
}
