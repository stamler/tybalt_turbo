package main

import (
	"database/sql"
	"log"

	_ "github.com/marcboeker/go-duckdb" // Import DuckDB driver
)

// The function uses DuckDB to normalize the Jobs.parquet data by:
// 1. Creating a separate Clients.parquet file with unique client records and UUIDs
// 2. Creating a separate Contacts.parquet file with unique contact records, UUIDs, and corresponding client relationships
// 3. Updating Jobs.parquet to reference clients and contacts via UUID foreign keys instead of names
func jobsToClientsAndContacts() {

	db, err := sql.Open("duckdb", "")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	splitQuery := `
		-- Load Jobs.parquet into a table called jobs
		CREATE TABLE jobs AS
		SELECT *, TRIM(client) AS t_client, TRIM(clientContact) AS t_clientContact FROM read_parquet('Jobs.parquet');

		-- Create the clients table, removing duplicate names
		CREATE TABLE clients AS
		SELECT uuid() AS id, t_client AS name
		FROM (SELECT DISTINCT t_client FROM jobs);

		COPY clients TO 'Clients.parquet' (FORMAT PARQUET);

		-- Create the contacts table where name is trimmed
		CREATE TABLE contacts AS
		SELECT uuid() AS id, clientContact AS name, c.id AS client_id
		FROM (
				SELECT DISTINCT t_clientContact AS clientContact, t_client AS client
				FROM jobs
				WHERE t_clientContact IS NOT NULL AND t_clientContact != '' 
					AND t_client IS NOT NULL AND t_client != ''
		) unique_contacts
		JOIN clients c ON unique_contacts.client = c.name;

		COPY contacts TO 'Contacts.parquet' (FORMAT PARQUET);
		
		-- Update the jobs table to use the new client and contact ids instead of the old client and clientContact columns
		ALTER TABLE jobs ADD COLUMN client_id uuid;
		ALTER TABLE jobs ADD COLUMN contact_id uuid;

		UPDATE jobs SET client_id = clients.id FROM clients WHERE jobs.t_client = clients.name;
		UPDATE jobs SET contact_id = contacts.id FROM contacts WHERE jobs.t_clientContact = contacts.name;
		
		-- Audit jobs by joining with clients and contacts
		CREATE TABLE jobs_audit AS
		SELECT jobs.*, clients.name AS client_match, contacts.name AS contact_match
		FROM jobs
		LEFT JOIN clients ON jobs.client_id = clients.id
		LEFT JOIN contacts ON jobs.contact_id = contacts.id;

		COPY jobs_audit TO 'Jobs_audit.parquet' (FORMAT PARQUET);

		-- ALTER TABLE jobs DROP client;
		-- ALTER TABLE jobs DROP clientContact;
		-- ALTER TABLE jobs DROP t_client;
		-- ALTER TABLE jobs DROP t_clientContact;
		-- ALTER TABLE jobs RENAME client_id TO client;
		-- ALTER TABLE jobs RENAME contact_id TO clientContact;

		COPY jobs TO 'Jobs.parquet' (FORMAT PARQUET);`

	_, err = db.Exec(splitQuery)
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}
}
