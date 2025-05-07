package extract

import (
	"database/sql"
	"log"

	_ "github.com/marcboeker/go-duckdb" // Import DuckDB driver
)

// The function uses DuckDB to extract the UserClaims.parquet file from the
// Profiles.parquet file

func profilesToUserClaims() {

	db, err := sql.Open("duckdb", "")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	defer db.Close()

	splitQuery := `
		-- Load Profiles.parquet into a table called profiles
		CREATE TABLE profiles AS
		SELECT * FROM read_parquet('parquet/Profiles.parquet');

		-- Load claims.parquet into a table called claims
		CREATE TABLE claims AS
		SELECT * FROM read_parquet('parquet/claims.parquet');

		-- customClaims is a comma-separated list of claim names. We'll create the
		-- user_claims table by selecting one claim name per row along with the corresponding
		-- pocketbase_uid from the profiles table. We'll join the claims table on the claim
		-- name to get the claim id for each user-claim pair.
		CREATE TABLE user_claims AS
		SELECT
			split_part(customClaims, ',', gs.idx) AS claim,
			claims.id cid,
			pocketbase_uid uid
		FROM profiles
		CROSS JOIN generate_series(
			1,
			array_length(string_to_array(customClaims, ','))
		) AS gs(idx)
		JOIN claims ON split_part(customClaims, ',', gs.idx) = claims.name;

		-- Now we'll load the user_claims table into a parquet file
		COPY user_claims TO 'parquet/UserClaims.parquet' (FORMAT PARQUET);
`

	_, err = db.Exec(splitQuery)
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}
}
