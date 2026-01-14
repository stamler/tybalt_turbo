package extract

import (
	"log"
	"os"

	_ "github.com/marcboeker/go-duckdb" // Import DuckDB driver
)

// jobsToClientsAndContacts normalizes the Jobs.parquet data by creating
// Clients.parquet and Contacts.parquet files, then updating Jobs.parquet
// to reference clients and contacts via IDs.
//
// HYBRID ID RESOLUTION:
// This function implements hybrid ID resolution for clients and contacts:
//  1. If a job has clientId/jobOwnerId/clientContactId populated (from Turbo writeback),
//     use that preserved PocketBase ID and pull data from TurboClients/TurboClientContacts.
//  2. If a job does NOT have these ID fields populated (legacy-only job),
//     generate a deterministic ID using MD5 hash of the name.
//
// This ensures that:
// - Turbo-created/edited jobs preserve their original PocketBase IDs
// - Legacy-only jobs get consistent generated IDs
// - Only clients/contacts referenced by jobs are imported (no orphans)
func jobsToClientsAndContacts() {

	db, err := openDuckDB()
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create deterministic ID generation macro in DuckDB
	_, err = db.Exec(`
CREATE OR REPLACE MACRO make_pocketbase_id(source_value, length)
AS substr(md5(CAST(source_value AS VARCHAR)), 1, length);
`)
	if err != nil {
		log.Fatalf("Failed to create make_pocketbase_id macro: %v", err)
	}

	// Check if TurboClients.parquet exists
	turboClientsExists := fileExists("parquet/TurboClients.parquet")
	turboContactsExists := fileExists("parquet/TurboClientContacts.parquet")

	// Log what we're working with
	if turboClientsExists {
		log.Println("TurboClients.parquet found - will use hybrid ID resolution for clients")
	} else {
		log.Println("TurboClients.parquet not found - will generate all client IDs from names")
	}
	if turboContactsExists {
		log.Println("TurboClientContacts.parquet found - will use hybrid ID resolution for contacts")
	} else {
		log.Println("TurboClientContacts.parquet not found - will generate all contact IDs from names")
	}

	// Build the SQL query for hybrid ID resolution
	hybridQuery := buildHybridQuery(turboClientsExists, turboContactsExists)

	_, err = db.Exec(hybridQuery)
	if err != nil {
		log.Fatalf("Failed to execute hybrid ID resolution query: %v", err)
	}
}

// fileExists checks if a file exists and is not a directory
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// buildHybridQuery constructs the SQL query for hybrid ID resolution
func buildHybridQuery(turboClientsExists, turboContactsExists bool) string {
	query := `
		-- Load Jobs.parquet into a table called jobs
		CREATE TABLE jobs AS
		SELECT *,
			TRIM(client) AS t_client,
			TRIM(clientContact) AS t_clientContact,
			TRIM(jobOwner) AS t_jobOwner
		FROM read_parquet('parquet/Jobs.parquet');
	`

	// Load TurboClients if available
	if turboClientsExists {
		query += `
		-- Load TurboClients.parquet as lookup table for preserved IDs
		CREATE TABLE turbo_clients AS
		SELECT * FROM read_parquet('parquet/TurboClients.parquet');
	`
	}

	// Load TurboClientContacts if available
	if turboContactsExists {
		query += `
		-- Load TurboClientContacts.parquet as lookup table for preserved IDs
		CREATE TABLE turbo_contacts AS
		SELECT * FROM read_parquet('parquet/TurboClientContacts.parquet');
	`
	}

	// Build clients table with hybrid ID resolution
	if turboClientsExists {
		query += `
		-- Build clients with hybrid ID resolution
		-- Priority: Use preserved ID from TurboClients if job has clientId/jobOwnerId,
		-- otherwise generate ID from client name
		CREATE TABLE clients AS
		WITH client_refs AS (
			-- Get all client references from jobs (both client and jobOwner)
			SELECT DISTINCT
				clientId AS turbo_id,
				t_client AS name
			FROM jobs
			WHERE t_client IS NOT NULL AND t_client != ''
			UNION
			SELECT DISTINCT
				jobOwnerId AS turbo_id,
				t_jobOwner AS name
			FROM jobs
			WHERE t_jobOwner IS NOT NULL AND t_jobOwner != ''
		),
		resolved AS (
			SELECT
				-- If turbo_id is populated, use it; otherwise generate from name
				CASE
					WHEN cr.turbo_id IS NOT NULL AND cr.turbo_id != ''
					THEN cr.turbo_id
					ELSE make_pocketbase_id(cr.name, 15)
				END AS id,
				-- If turbo_id is populated, get name from TurboClients; otherwise use job's client name
				CASE
					WHEN cr.turbo_id IS NOT NULL AND cr.turbo_id != ''
					THEN COALESCE(tc.name, cr.name)
					ELSE cr.name
				END AS name,
				-- businessDevelopmentLeadUid only available from TurboClients
				CASE
					WHEN cr.turbo_id IS NOT NULL AND cr.turbo_id != ''
					THEN tc.businessDevelopmentLeadUid
					ELSE NULL
				END AS businessDevelopmentLeadUid
			FROM client_refs cr
			LEFT JOIN turbo_clients tc ON cr.turbo_id = tc.id
		)
		SELECT DISTINCT id, name, businessDevelopmentLeadUid
		FROM resolved
		WHERE name IS NOT NULL AND name != ''
		ORDER BY name;

		COPY clients TO 'parquet/Clients.parquet' (FORMAT PARQUET);
	`
	} else {
		// No TurboClients - generate all IDs from names (original behavior)
		query += `
		-- Build clients (no TurboClients available - generate all IDs from names)
		CREATE TABLE clients AS
		SELECT
			make_pocketbase_id(name, 15) AS id,
			name,
			NULL AS businessDevelopmentLeadUid
		FROM (
			SELECT DISTINCT t_client AS name FROM jobs WHERE t_client IS NOT NULL AND t_client != ''
			UNION
			SELECT DISTINCT t_jobOwner AS name FROM jobs WHERE t_jobOwner IS NOT NULL AND t_jobOwner != ''
		)
		ORDER BY name;

		COPY clients TO 'parquet/Clients.parquet' (FORMAT PARQUET);
	`
	}

	// Build contacts table with hybrid ID resolution
	if turboContactsExists {
		query += `
		-- Build contacts with hybrid ID resolution
		CREATE TABLE contacts_resolved AS
		WITH contact_refs AS (
			-- Get all contact references from jobs
			SELECT DISTINCT
				clientContactId AS turbo_id,
				t_clientContact AS contact_name,
				t_client AS client_name,
				clientId AS client_turbo_id
			FROM jobs
			WHERE t_clientContact IS NOT NULL AND t_clientContact != ''
		),
		resolved AS (
			SELECT
				-- If turbo_id is populated, use it; otherwise generate from contact+client name
				CASE
					WHEN cr.turbo_id IS NOT NULL AND cr.turbo_id != ''
					THEN cr.turbo_id
					ELSE make_pocketbase_id(CONCAT(cr.contact_name, '|', cr.client_name), 15)
				END AS id,
				-- Get data from TurboClientContacts if available
				CASE
					WHEN cr.turbo_id IS NOT NULL AND cr.turbo_id != ''
					THEN COALESCE(tc.surname, '')
					ELSE ''
				END AS surname,
				CASE
					WHEN cr.turbo_id IS NOT NULL AND cr.turbo_id != ''
					THEN COALESCE(tc.givenName, '')
					ELSE ''
				END AS givenName,
				CASE
					WHEN cr.turbo_id IS NOT NULL AND cr.turbo_id != ''
					THEN COALESCE(tc.email, '')
					ELSE ''
				END AS email,
				-- Determine client_id: use TurboClientContacts.clientId if available,
				-- else use job's clientId if available, else generate from client name
				CASE
					WHEN cr.turbo_id IS NOT NULL AND cr.turbo_id != '' AND tc.clientId IS NOT NULL
					THEN tc.clientId
					WHEN cr.client_turbo_id IS NOT NULL AND cr.client_turbo_id != ''
					THEN cr.client_turbo_id
					ELSE make_pocketbase_id(cr.client_name, 15)
				END AS client_id,
				cr.contact_name AS name
			FROM contact_refs cr
			LEFT JOIN turbo_contacts tc ON cr.turbo_id = tc.id
		)
		SELECT DISTINCT id, name, client_id, email, surname, givenName
		FROM resolved
		WHERE name IS NOT NULL AND name != '';

		-- For contacts without preserved data, extract surname/givenName from name
		CREATE TABLE contacts AS
		WITH NameParts AS (
			SELECT
				*,
				string_split(trim(name), ' ') AS parts
			FROM contacts_resolved
		)
		SELECT
			id,
			name,
			client_id,
			email,
			CASE
				WHEN surname != '' THEN surname
				WHEN len(parts) = 0 THEN ''
				ELSE parts[len(parts)]
			END AS surname,
			CASE
				WHEN givenName != '' THEN givenName
				WHEN len(parts) <= 1 THEN ''
				ELSE array_to_string(list_slice(parts, 1, len(parts) - 1), ' ')
			END AS givenName
		FROM NameParts
		ORDER BY name, client_id;

		COPY contacts TO 'parquet/Contacts.parquet' (FORMAT PARQUET);
	`
	} else {
		// No TurboClientContacts - generate all IDs from names (original behavior)
		query += `
		-- Build contacts (no TurboClientContacts available - generate all IDs from names)
		CREATE TABLE contacts AS
		SELECT make_pocketbase_id(CONCAT(clientContact, '|', client), 15) AS id, clientContact AS name, c.id AS client_id
		FROM (
			SELECT DISTINCT t_clientContact AS clientContact, t_client AS client
			FROM jobs
			WHERE t_clientContact IS NOT NULL AND t_clientContact != ''
				AND t_client IS NOT NULL AND t_client != ''
			ORDER BY t_clientContact, t_client
		) unique_contacts
		JOIN clients c ON unique_contacts.client = c.name;

		-- Augment the contacts table with surname, givenName, and empty email
		CREATE TABLE contacts_augmented AS
		WITH NameParts AS (
			SELECT
				*,
				string_split(trim(name), ' ') AS parts
			FROM contacts
		)
		SELECT
			id, name, client_id, '' as email,
			CASE
				WHEN len(parts) = 0 THEN ''
				ELSE parts[len(parts)]
			END AS surname,
			CASE
				WHEN len(parts) <= 1 THEN ''
				ELSE array_to_string(list_slice(parts, 1, len(parts) - 1), ' ')
			END AS givenName
		FROM NameParts
		ORDER BY name, client_id;

		COPY contacts_augmented TO 'parquet/Contacts.parquet' (FORMAT PARQUET);

		-- Rename for consistency with the rest of the query
		DROP TABLE contacts;
		ALTER TABLE contacts_augmented RENAME TO contacts;
	`
	}

	// Update jobs with resolved client/contact IDs
	if turboClientsExists || turboContactsExists {
		query += `
		-- Update jobs table with resolved IDs
		-- Add columns for the resolved IDs
		ALTER TABLE jobs ADD COLUMN client_id string;
		ALTER TABLE jobs ADD COLUMN job_owner_id string;
		ALTER TABLE jobs ADD COLUMN contact_id string;

		-- Set client_id: use preserved ID if available, else generate from name
		UPDATE jobs SET client_id = 
			CASE
				WHEN clientId IS NOT NULL AND clientId != '' THEN clientId
				ELSE make_pocketbase_id(t_client, 15)
			END
		WHERE t_client IS NOT NULL AND t_client != '';

		-- Set job_owner_id: use preserved ID if available, else generate from name
		UPDATE jobs SET job_owner_id = 
			CASE
				WHEN jobOwnerId IS NOT NULL AND jobOwnerId != '' THEN jobOwnerId
				ELSE make_pocketbase_id(t_jobOwner, 15)
			END
		WHERE t_jobOwner IS NOT NULL AND t_jobOwner != '';

		-- Set contact_id: use preserved ID if available, else generate from name+client
		UPDATE jobs SET contact_id = 
			CASE
				WHEN clientContactId IS NOT NULL AND clientContactId != '' THEN clientContactId
				ELSE make_pocketbase_id(CONCAT(t_clientContact, '|', t_client), 15)
			END
		WHERE t_clientContact IS NOT NULL AND t_clientContact != '';
	`
	} else {
		// Original behavior - join on names
		query += `
		-- Update jobs table to use resolved IDs (original name-based approach)
		ALTER TABLE jobs ADD COLUMN client_id string;
		ALTER TABLE jobs ADD COLUMN job_owner_id string;
		ALTER TABLE jobs ADD COLUMN contact_id string;

		UPDATE jobs SET client_id = clients.id FROM clients WHERE jobs.t_client = clients.name;
		UPDATE jobs SET job_owner_id = clients.id FROM clients WHERE jobs.t_jobOwner = clients.name;
		UPDATE jobs SET contact_id = contacts.id FROM contacts WHERE jobs.t_clientContact = contacts.name;
	`
	}

	// Create audit table and output final Jobs.parquet
	query += `
		-- Audit jobs by joining with clients and contacts
		CREATE TABLE jobs_audit AS
		SELECT jobs.*, clients.name AS client_match, contacts.name AS contact_match
		FROM jobs
		LEFT JOIN clients ON jobs.client_id = clients.id
		LEFT JOIN clients AS job_owners ON jobs.job_owner_id = clients.id
		LEFT JOIN contacts ON jobs.contact_id = contacts.id
		ORDER BY jobs.id;

		COPY jobs_audit TO 'parquet/Jobs_audit.parquet' (FORMAT PARQUET);

		COPY (SELECT * FROM jobs ORDER BY id) TO 'parquet/Jobs.parquet' (FORMAT PARQUET);
	`

	return query
}
