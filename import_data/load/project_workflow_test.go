package load

import (
	"database/sql"
	"encoding/csv"
	"os"
	"testing"
)

func TestProjectImportProtectsWorkflow(t *testing.T) {
	file, err := os.Open("testdata/project_workflow.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	cases, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range cases[1:] {
		t.Run(row[0], func(t *testing.T) {
			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			db.SetMaxOpenConns(1)
			for _, ddl := range []string{"CREATE TABLE app_config (key TEXT,value TEXT)", "CREATE TABLE job_attachments (id TEXT)", "CREATE TABLE job_requests (id TEXT)", "CREATE TABLE client_invoicing_information (id TEXT)"} {
				if _, err = db.Exec(ddl); err != nil {
					t.Fatal(err)
				}
			}
			// Load the scenario values from the CSV fixture.
			if _, err = db.Exec("INSERT INTO app_config VALUES ('jobs', ?)", `{"require_job_request_approval":`+row[1]+`}`); err != nil {
				t.Fatal(err)
			}
			for i, table := range []string{"job_attachments", "job_requests", "client_invoicing_information"} {
				if row[i+2] == "1" {
					if _, err = db.Exec("INSERT INTO " + table + " VALUES ('fixture')"); err != nil {
						t.Fatal(err)
					}
				}
			}
			tx, err := db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			blocked := CheckLocalDataImport(tx) != nil
			if blocked != (row[5] == "true") {
				t.Fatalf("blocked=%v", blocked)
			}
		})
	}
}
