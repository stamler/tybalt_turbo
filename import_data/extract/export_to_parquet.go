package extract

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/marcboeker/go-duckdb" // Import DuckDB driver
	"golang.org/x/crypto/ssh"
)

// The tablesToDump variable is used to specify the tables that should be
// exported to Parquet format.
var tablesToDump = []string{"TimeEntries", "TimeSheets", "TimeAmendments", "Expenses", "MileageResetDates", "Profiles", "Jobs", "TurboClients", "TurboClientContacts"}

func ToParquet(sourceSQLiteDb string) {
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
	db, err := openDuckDB()
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

	// Validate that Profiles.defaultBranch is present for all rows before exporting.
	// We fail fast and list the offending records to prevent generating a bad export.
	validateQuery := `
		SELECT 
			COALESCE(NULLIF(p.pocketbase_uid, ''), CAST(p.id AS VARCHAR)) AS ident,
			COALESCE(p.email, '') AS email
		FROM mysql_db.Profiles p
		WHERE p.defaultBranch IS NULL OR TRIM(p.defaultBranch) = ''
	`
	rows, err := db.Query(validateQuery)
	if err != nil {
		log.Fatalf("Failed to validate Profiles.defaultBranch: %v", err)
	}
	defer rows.Close()

	offenders := []string{}
	for rows.Next() {
		var ident, email string
		if err := rows.Scan(&ident, &email); err != nil {
			log.Fatalf("Failed to scan validation row: %v", err)
		}
		label := ident
		if email != "" {
			label = fmt.Sprintf("%s <%s>", ident, email)
		}
		offenders = append(offenders, label)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("Validation iteration error: %v", err)
	}
	if len(offenders) > 0 {
		log.Fatalf("Export aborted: found %d Profiles with empty/NULL defaultBranch: %s", len(offenders), strings.Join(offenders, ", "))
	}

	for _, table := range tablesToDump {
		// Read from MySQL and export to Parquet
		var query string

		switch table {
		case "Jobs":
			// Specific query for Jobs table to format dates as strings (YYYY-MM-DD or empty)
			// and use immutableID as pocketbase_id. Use %% for literal % in fmt.Sprintf.
			// immutableID is the stable ID from Firestore that enables round-trip writebacks.
			// Include clientId, clientContactId, jobOwnerId for hybrid ID resolution on import.
			query = fmt.Sprintf(`
				COPY (
					SELECT * EXCLUDE (projectAwardDate, proposalOpeningDate, proposalSubmissionDueDate, pocketbase_id),
						immutableID AS pocketbase_id,
						strftime(projectAwardDate, '%%Y-%%m-%%d') AS projectAwardDate,
						strftime(proposalOpeningDate, '%%Y-%%m-%%d') AS proposalOpeningDate,
						strftime(proposalSubmissionDueDate, '%%Y-%%m-%%d') AS proposalSubmissionDueDate,
						clientId,
						clientContactId,
						jobOwnerId
					FROM mysql_db.Jobs
				) TO 'parquet/Jobs.parquet' (FORMAT PARQUET)
		 `)

		case "Profiles":
			// Specific query for Profiles to add both pocketbase_id and pocketbase_uid
			// Use different random seeds to avoid potential caching issues if needed.
			// pocketbase_id is already in the table derived from LEFT 15 characters of ID
			// pocketbase_uid is already in the table derived from RIGHT 15 characters of ID
			query = `
				COPY ( SELECT * FROM mysql_db.Profiles ) TO 'parquet/Profiles.parquet' (FORMAT PARQUET)
			`

		case "TimeSheets":
			// weekEnding should be a string in the format YYYY-MM-DD
			query = `
				COPY (
					SELECT * EXCLUDE (weekEnding),
						CAST(weekEnding AS VARCHAR) AS weekEnding
					FROM mysql_db.TimeSheets
				) TO 'parquet/TimeSheets.parquet' (FORMAT PARQUET)
			`
		case "TimeEntries":
			// weekEnding should be a string in the format YYYY-MM-DD
			// Generate deterministic pocketbase_id using MD5 hash of the existing id
			query = `
				COPY (
					SELECT * EXCLUDE (date),
						substr(md5(CAST(id AS VARCHAR)), 1, 15) AS pocketbase_id,
						CAST(date AS VARCHAR) AS date
					FROM mysql_db.TimeEntries
				) TO 'parquet/TimeEntries.parquet' (FORMAT PARQUET)
			`
		case "TimeAmendments":
			// weekEnding should be a string in the format YYYY-MM-DD
			// pocketbase_id is already in the table derived from LEFT 15 characters of ID
			query = `
				COPY (
					SELECT * EXCLUDE (committedWeekEnding, weekEnding, date),
						CAST(committedWeekEnding AS VARCHAR) AS committedWeekEnding,
						CAST(weekEnding AS VARCHAR) AS weekEnding,
						CAST(date AS VARCHAR) AS date
					FROM mysql_db.TimeAmendments
				) TO 'parquet/TimeAmendments.parquet' (FORMAT PARQUET)
			`
		case "Expenses":
			// pocketbase_id is already in the table
			query = `
				COPY ( SELECT * FROM mysql_db.Expenses ) TO 'parquet/Expenses.parquet' (FORMAT PARQUET)
			`
		case "MileageResetDates":
			// pocketbase_id is already in the table
			query = `
				COPY ( SELECT pocketbase_id, CAST(date AS VARCHAR) AS date FROM mysql_db.MileageResetDates ) TO 'parquet/MileageResetDates.parquet' (FORMAT PARQUET)
				`
		case "TurboClients":
			// Export TurboClients for hybrid ID resolution during import.
			// These are clients that were written back from Turbo with preserved PocketBase IDs.
			query = `
				COPY (
					SELECT id, name, businessDevelopmentLeadUid
					FROM mysql_db.TurboClients
				) TO 'parquet/TurboClients.parquet' (FORMAT PARQUET)
			`
		case "TurboClientContacts":
			// Export TurboClientContacts for hybrid ID resolution during import.
			// These are contacts that were written back from Turbo with preserved PocketBase IDs.
			query = `
				COPY (
					SELECT id, surname, givenName, email, clientId
					FROM mysql_db.TurboClientContacts
				) TO 'parquet/TurboClientContacts.parquet' (FORMAT PARQUET)
			`
		default:
			// Generic query for other tables, just adding pocketbase_id
			// Generate deterministic pocketbase_id using MD5 hash of the existing id
			query = fmt.Sprintf(`
				COPY (
					SELECT *,
						substr(md5(CAST(id AS VARCHAR)), 1, 15) AS pocketbase_id
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
	// contacts via foreign keys.
	jobsToClientsAndContacts()

	// Augment Clients.parquet to resolve businessDevelopmentLeadUid (legacy Firebase UID)
	// to PocketBase UID by joining with Profiles.parquet.
	augmentClients()

	// Normalize the Expenses.parquet data by creating Vendors.parquet and
	// updating Expenses.parquet to reference vendors via foreign keys.
	expensesToVendors()

	// Dump the pre-populated tables from the sqlite test database.
	sqliteTableDumps(sourceSQLiteDb, "divisions")
	sqliteTableDumps(sourceSQLiteDb, "time_types")
	sqliteTableDumps(sourceSQLiteDb, "claims")
	sqliteTableDumps(sourceSQLiteDb, "branches")

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

	// Augment Expenses.parquet data
	augmentExpenses()

	// Now that Expenses.parquet is augmented with pocketbase_* and *_id fields,
	// create purchase_orders.parquet and wire expenses to purchase_orders
	expensesToPurchaseOrders()

	// Extract the UserClaims.parquet file from the Profiles.parquet file.
	profilesToUserClaims()

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
