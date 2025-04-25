package extract

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/marcboeker/go-duckdb" // Import DuckDB driver
	"golang.org/x/crypto/ssh"
)

// The tablesToDump variable is used to specify the tables that should be
// exported to Parquet format.
var tablesToDump = []string{"TimeEntries", "TimeSheets", "TimeAmendments", "Expenses", "Profiles", "Jobs"}

func ToParquet() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Create the parquet directory if it doesn't exist
	parquetDir := "parquet"
	if _, err := os.Stat(parquetDir); os.IsNotExist(err) {
		err = os.Mkdir(parquetDir, 0755)
		if err != nil {
			log.Fatal("Failed to create parquet directory:", err)
		}
	}

	// SSH configuration
	sshConfig := &ssh.ClientConfig{
		User: os.Getenv("SSH_USER"),
		Auth: []ssh.AuthMethod{
			ssh.Password(os.Getenv("SSH_PASSWORD")), // For better security, consider using SSH keys
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // In production, use a proper callback
	}

	// Connect to SSH server
	sshClient, err := ssh.Dial("tcp", os.Getenv("SSH_HOST")+":"+os.Getenv("SSH_PORT"), sshConfig)
	if err != nil {
		log.Fatal("Failed to connect to SSH server:", err)
	}
	defer sshClient.Close()

	// Set up local port forwarding
	localPort := "3307"                   // Local port
	remoteHost := os.Getenv("MYSQL_HOST") // Remote MySQL host
	remotePort := os.Getenv("MYSQL_PORT") // Remote MySQL port
	listener, err := net.Listen("tcp", "localhost:"+localPort)
	if err != nil {
		log.Fatal("Failed to set up local listener:", err)
	}
	// I need to comment this out because it closes the listener before the
	// forwarding is done. Why?
	// defer listener.Close()

	// Start forwarding in a separate goroutine
	go func() {
		for {
			localConn, err := listener.Accept()
			if err != nil {
				log.Println("Failed to accept local connection:", err)
				continue
			}

			// Dial the remote MySQL server via SSH
			remoteConn, err := sshClient.Dial("tcp", remoteHost+":"+remotePort)
			if err != nil {
				log.Println("Failed to dial remote MySQL server:", err)
				localConn.Close()
				continue
			}

			// Bidirectional copy
			go func() {
				defer localConn.Close()
				defer remoteConn.Close()
				go copyData(localConn, remoteConn)
				copyData(remoteConn, localConn)
			}()
		}
	}()

	// Continue with your DuckDB attachment and export logic...

	// Example continuation (simplified):
	db, err := sql.Open("duckdb", "")
	if err != nil {
		log.Fatal("Failed to connect to DuckDB:", err)
	}
	defer db.Close()

	attachQuery := fmt.Sprintf(
		"INSTALL mysql; LOAD mysql; ATTACH 'host=%s user=%s password=%s port=%s database=%s' AS mysql_db (TYPE mysql_scanner, READ_ONLY); USE mysql_db;",
		"localhost", os.Getenv("MYSQL_USER"), os.Getenv("MYSQL_PASSWORD"), localPort, os.Getenv("MYSQL_DB"))
	_, err = db.Exec(attachQuery)
	if err != nil {
		log.Fatalf("Failed to attach MySQL database: %v", err)
	}

	for _, table := range tablesToDump {
		// Read from MySQL and export to Parquet
		var query string

		if table == "Jobs" {
			// Specific query for Jobs table to format dates as strings (YYYY-MM-DD or empty)
			// and add pocketbase_id. Use %% for literal % in fmt.Sprintf.
			query = fmt.Sprintf(`
				COPY (
					SELECT * EXCLUDE (projectAwardDate, proposalOpeningDate, proposalSubmissionDueDate),
						strftime(projectAwardDate, '%%Y-%%m-%%d') AS projectAwardDate,
						strftime(proposalOpeningDate, '%%Y-%%m-%%d') AS proposalOpeningDate,
						strftime(proposalSubmissionDueDate, '%%Y-%%m-%%d') AS proposalSubmissionDueDate,
						array_to_string(array_slice(array_apply(range(15), i -> CASE WHEN random() < 0.72 THEN chr(CAST(floor(random() * 26) + 97 AS INTEGER)) ELSE CAST(CAST(floor(random() * 10) AS INTEGER) AS VARCHAR) END), 1, 15), '') AS pocketbase_id
					FROM mysql_db.Jobs
				) TO 'parquet/Jobs.parquet' (FORMAT PARQUET)
		 `)

		} else if table == "Profiles" {
			// Specific query for Profiles to add both pocketbase_id and pocketbase_uid
			// Use different random seeds to avoid potential caching issues if needed.
			query = `
				COPY (
					SELECT *,
						array_to_string(array_slice(array_apply(range(15), i -> CASE WHEN random() < 0.72 THEN chr(CAST(floor(random() * 26) + 97 AS INTEGER)) ELSE CAST(CAST(floor(random() * 10) AS INTEGER) AS VARCHAR) END), 1, 15), '') AS pocketbase_id,
						array_to_string(array_slice(array_apply(range(15), i -> CASE WHEN random() < 0.71 THEN chr(CAST(floor(random() * 26) + 97 AS INTEGER)) ELSE CAST(CAST(floor(random() * 10) AS INTEGER) AS VARCHAR) END), 1, 15), '') AS pocketbase_uid
					FROM mysql_db.Profiles
				) TO 'parquet/Profiles.parquet' (FORMAT PARQUET)
			`

		} else if table == "TimeSheets" {
			// weekEnding should be a string in the format YYYY-MM-DD
			query = `
				COPY (
					SELECT 
						array_to_string(array_slice(array_apply(range(15), i -> CASE WHEN random() < 0.72 THEN chr(CAST(floor(random() * 26) + 97 AS INTEGER)) ELSE CAST(CAST(floor(random() * 10) AS INTEGER) AS VARCHAR) END), 1, 15), '') AS pocketbase_id,
						* EXCLUDE (weekEnding),
						CAST(weekEnding AS VARCHAR) AS weekEnding
					FROM mysql_db.TimeSheets
				) TO 'parquet/TimeSheets.parquet' (FORMAT PARQUET)
			`
		} else if table == "TimeEntries" {
			// weekEnding should be a string in the format YYYY-MM-DD
			query = `
				COPY (
					SELECT * EXCLUDE (date),
						array_to_string(array_slice(array_apply(range(15), i -> CASE WHEN random() < 0.72 THEN chr(CAST(floor(random() * 26) + 97 AS INTEGER)) ELSE CAST(CAST(floor(random() * 10) AS INTEGER) AS VARCHAR) END), 1, 15), '') AS pocketbase_id,
						CAST(date AS VARCHAR) AS date
					FROM mysql_db.TimeEntries
				) TO 'parquet/TimeEntries.parquet' (FORMAT PARQUET)
			`
		} else if table == "TimeAmendments" {
			// weekEnding should be a string in the format YYYY-MM-DD
			query = `
				COPY (
					SELECT * EXCLUDE (committedWeekEnding, weekEnding, date),
						array_to_string(array_slice(array_apply(range(15), i -> CASE WHEN random() < 0.72 THEN chr(CAST(floor(random() * 26) + 97 AS INTEGER)) ELSE CAST(CAST(floor(random() * 10) AS INTEGER) AS VARCHAR) END), 1, 15), '') AS pocketbase_id,
						CAST(committedWeekEnding AS VARCHAR) AS committedWeekEnding,
						CAST(weekEnding AS VARCHAR) AS weekEnding,
						CAST(date AS VARCHAR) AS date
					FROM mysql_db.TimeAmendments
				) TO 'parquet/TimeAmendments.parquet' (FORMAT PARQUET)
			`
		} else {
			// Generic query for other tables, just adding pocketbase_id
			query = fmt.Sprintf(`
				COPY (
					SELECT *,
						array_to_string(array_slice(array_apply(range(15), i -> CASE WHEN random() < 0.72 THEN chr(CAST(floor(random() * 26) + 97 AS INTEGER)) ELSE CAST(CAST(floor(random() * 10) AS INTEGER) AS VARCHAR) END), 1, 15), '') AS pocketbase_id
					FROM mysql_db.%s
				) TO 'parquet/%s.parquet' (FORMAT PARQUET)
			`, table, table)
		}

		_, err = db.Exec(query)
		if err != nil {
			// Provide more context in the error message
			log.Fatalf("Failed to export table %s to Parquet: %v", table, err)
		}
	}

	// Normalize the Jobs.parquet data by creating Clients.parquet and
	// Contacts.parquet and updating Jobs.parquet to reference clients and
	// contacts via UUID foreign keys.
	jobsToClientsAndContacts()

	// Dump the pre-populated tables from the sqlite test database.
	sqliteTableDumps("../app/test_pb_data/data.db", "divisions")
	sqliteTableDumps("../app/test_pb_data/data.db", "time_types")

	// Augment the Profiles.parquet data by adding the pocketbase_uid column.
	augmentProfiles()

	// Augment the Jobs.parquet data by adding the manager_id column.
	augmentJobs()

	// Augment TimeSheets.parquet data creating the pocketbase_uid and
	// pocketbase_approver_uid columns.
	augmentTimeSheets()

	// Augment TimeEntries.parquet data
	augmentTimeEntries()

	// Augment TimeAmendments.parquet data
	augmentTimeAmendments()

	// Independent Collections (Profiles, Jobs) must be loaded first.
	// TimeSheets can be loaded next because TimeEntries references TimeSheets.
	// TimeEntries can be loaded next because it references TimeSheets and Jobs.
	// TimeAmendments can be loaded next because it references TimeSheets and Profiles.
	// Expenses can be loaded last because it references TimeSheets and Jobs.

	// For TimeEntries:
	// LEFT JOIN sqlite.time_types ON sqlite.time_types.code = main.TimeEntries.timetype
	// LEFT JOIN sqlite.divisions ON sqlite.divisions.code = main.TimeEntries.division
	// This will allow us to write the id of the time_types and divisions tables
	// to the TimeEntries table rather than the code from the parquet file.
}

func copyData(dst net.Conn, src net.Conn) {
	_, err := io.Copy(dst, src)
	if err != nil {
		log.Println("Data copy error:", err)
	}
}
