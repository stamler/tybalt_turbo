package main

import (
	"database/sql"
	"log"

	_ "github.com/marcboeker/go-duckdb" // Import DuckDB driver
)

func jobsToClientsAndContacts() {

	db, err := sql.Open("duckdb", "")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	splitQuery := `
		 -- Load Jobs.parquet into a table called jobs
		 CREATE TABLE jobs AS
		 SELECT * FROM read_parquet('Jobs.parquet');

	   -- Create the clients table where name is trimmed
	   CREATE TABLE clients AS
	   SELECT

	   	uuid() AS id,
	   	TRIM(client) AS name

	   FROM (
	   	SELECT DISTINCT client
	   	FROM jobs
	   );

	   COPY clients TO 'Clients.parquet' (FORMAT PARQUET);

	   -- Create the contacts table where name is trimmed
	   CREATE TABLE contacts AS
	   SELECT

	   	uuid() AS id,
	   	TRIM(clientContact) AS name,
	   	c.id AS client_id

	   FROM jobs
	   JOIN clients c ON TRIM(jobs.client) = c.name
	   WHERE jobs.clientContact IS NOT NULL AND TRIM(jobs.clientContact) != '' AND jobs.client IS NOT NULL AND TRIM(jobs.client) != '';

	   COPY contacts TO 'Contacts.parquet' (FORMAT PARQUET);
		 
		 -- Update the jobs table to use the new client and contact ids instead of the old client and clientContact columns
		 ALTER TABLE jobs ADD COLUMN client_id uuid;
		 ALTER TABLE jobs ADD COLUMN contact_id uuid;

		 UPDATE jobs SET client_id = clients.id FROM clients WHERE TRIM(jobs.client) = clients.name;
		 UPDATE jobs SET contact_id = contacts.id FROM contacts WHERE TRIM(jobs.clientContact) = contacts.name;

		 -- ALTER TABLE jobs DROP client;
		 -- ALTER TABLE jobs DROP clientContact;
		 -- ALTER TABLE jobs RENAME client_id TO client;
		 -- ALTER TABLE jobs RENAME contact_id TO clientContact;

		 COPY jobs TO 'Jobs.parquet' (FORMAT PARQUET);`

	_, err = db.Exec(splitQuery)
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}
}
