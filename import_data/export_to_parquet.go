package main

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

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
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
		query := fmt.Sprintf(`
    COPY (
      SELECT * FROM mysql_db.%s
		) TO '%s' (FORMAT PARQUET)`,
			table, table+".parquet")

		_, err = db.Exec(query)
		if err != nil {
			log.Fatalf("Failed to export to Parquet: %v", err)
		}
	}

	// TODO: shape the data into the target form then ATTACH the sqlite database
	// and write the data to the corresponding tables in the sqlite database,
	// honouring foreign key constraints and primary keys.

	// https://duckdb.org/2024/01/26/multi-database-support-in-duckdb.html

	// Jobs and Clients (related by foreign key) First get all distinct clients
	// from the Jobs table and insert them into the clients table in sqlite. We
	// also need to get the resulting id of the client so we can update the Jobs
	// table with the client id. Then we can insert the Jobs referencing the
	// client id. There will be duplicated clients and clients will have duplicate
	// contacts at first. This will be cleaned up later using merge functions.

	splitTable("Jobs.parquet", "Clients", []string{"client", "clientContact"}, "client")

	// Split the Contacts table out of the Clients table.

	// TODO: should we write the client id to the Contacts table? This would allow
	// us to merge contacts within the same client and seems like a good idea. By
	// not doing this a contact could associated with multiple clients or the
	// wrong client, but by doing it we can merge contacts within the same client
	// and it will be more efficient to load in the UI.
	splitTable("Clients.parquet", "Contacts", []string{"clientContact"}, "clientContact")

	// WE WILL NEED A MERGE CONTACTS FUNCTION TO MERGE DUPLICATE CONTACTS WITHIN
	// THE SAME CLIENT AND THEN UPDATE ALL THE JOBS THAT REFERENCE THE OLD
	// CONTACT TO REFERENCE THE NEWLY MERGED CONTACT.

	// WE WILL NEED A MERGE CLIENTS FUNCTION TO MERGE DUPLICATE CLIENTS AND THEN
	// UPDATE ALL THE JOBS THAT REFERENCE THE OLD CLIENT TO REFERENCE THE NEWLY
	// MERGED CLIENT.

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
