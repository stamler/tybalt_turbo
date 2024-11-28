package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/marcboeker/go-duckdb" // Import DuckDB driver
)

// splitTable takes a source parquet file and splits off the specified columns
// into a new parquet file. The source parquet file gets a new column added for
// the id of the corresponding row in the newly-created destination parquet
// file.

// Source and destination are table names inside duckdb and when we add .parquet
// to the end they become parquet files.
func splitTable(source string, destination string, columnsToSplit []string, keyColumn string) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// import the source parquet file into a table in duckdb
	createSourceTableQuery := "CREATE TABLE " + source + " AS SELECT * FROM read_parquet('" + source + ".parquet');"
	if _, err := db.Exec(createSourceTableQuery); err != nil {
		log.Fatalf("Failed to create source table: %v", err)
	}

	// create a destination table in duckdb with distinct combinations and
	// generate random IDs
	createDestTableQuery := "CREATE TABLE " + destination + " AS " +
		"SELECT substr(lower(to_base64(CAST(sha256(random()::text) AS BLOB))), 1, 15) as id, " +
		strings.Join(columnsToSplit, ", ") +
		" FROM (" +
		"SELECT DISTINCT " + strings.Join(trimColumns(columnsToSplit, true), ", ") +
		" FROM " + source + //");"
		" WHERE " + keyColumn + " IS NOT NULL);"

	if _, err := db.Exec(createDestTableQuery); err != nil {
		log.Fatalf("Failed to create destination table: %v", err)
	}

	copyDestTableQuery := "COPY " + destination + " TO '" + destination + ".parquet' (FORMAT PARQUET);"
	if _, err := db.Exec(copyDestTableQuery); err != nil {
		log.Fatalf("Failed to copy destination table to parquet: %v", err)
	}

	// create a new version of the source table with the columnsToSplit removed
	// and replaced with a destination_id column that references the destination
	// table

	// TODO: This query may be posing some problems because s.clientContact /
	// d.clientContact could be NULL and thus the equality will fail. How can we
	// handle this? Perhaps IS NOT DISTINCT FROM? That's what I've done but it
	// needs to be validated.
	createNewSourceTableQuery := "CREATE TABLE " + source + "_new AS " +
		"SELECT s.*, d.id AS " + destination + "_id " +
		"FROM " + source + " s LEFT JOIN " + destination + " d ON " +
		joinItems("s", "d", columnsToSplit) + ";"
	if _, err := db.Exec(createNewSourceTableQuery); err != nil {
		log.Fatalf("Failed to create new source table: %v", err)
	}

	// for each column to split, drop it from the new source table
	for _, col := range columnsToSplit {
		dropColumnQuery := "ALTER TABLE " + source + "_new DROP " + col + ";"
		if _, err := db.Exec(dropColumnQuery); err != nil {
			log.Fatalf("Failed to drop column from new source table: %v", err)
		}
	}

	// drop the old source table and rename the new one
	db.Exec("DROP TABLE " + source)
	db.Exec("ALTER TABLE " + source + "_new RENAME TO " + source)

	// write the source and destination tables to parquet files
	db.Exec("COPY " + source + " TO '" + source + ".parquet' (FORMAT PARQUET)")
}

// joinItems takes a list of columns and returns a string of SQL that specifies
// that the source and destination tables are equivalent if the specified
// columns are equivalent.
func joinItems(sourceToken string, destinationToken string, cols []string) string {
	comparisons := []string{}
	for _, col := range cols {
		comparisons = append(comparisons, "TRIM("+sourceToken+"."+col+") IS NOT DISTINCT FROM TRIM("+destinationToken+"."+col+")")
	}
	return strings.Join(comparisons, " AND ")
}

// trimColumns wraps each column name with the SQL TRIM() function. It takes a
// slice of column names and returns a new slice where each column is formatted
// as TRIM(column_name).
func trimColumns(cols []string, alias bool) []string {
	trimmedCols := []string{}
	for _, col := range cols {
		if alias {
			trimmedCols = append(trimmedCols, fmt.Sprintf("TRIM(%s) AS %s", col, col))
		} else {
			trimmedCols = append(trimmedCols, fmt.Sprintf("TRIM(%s)", col))
		}
	}
	return trimmedCols
}
