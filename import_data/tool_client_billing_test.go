package main

import (
	"database/sql"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
)

func TestClientBillingBulkDeleteGuard(t *testing.T) {
	file, err := os.Open("load/testdata/client_billing_rows.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"jobs", "clients", "client_contacts"} {
		t.Run(table, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "data.db")
			db, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			for _, name := range []string{table, "categories", "client_invoicing_information"} {
				if _, err = db.Exec("CREATE TABLE " + name + " (id TEXT,value TEXT)"); err != nil {
					t.Fatal(err)
				}
				// Load real values from the CSV fixture; categories is deliberately first in the deletion list.
				if _, err = db.Exec("INSERT INTO "+name+" VALUES (?,?)", rows[1][0], rows[1][1]); err != nil {
					t.Fatal(err)
				}
			}
			if err = deleteAllFromTables(path, []string{"categories", table}); err == nil {
				t.Fatal("deletion ignored a saved invoicing profile")
			}
			for _, name := range []string{table, "categories", "client_invoicing_information"} {
				var count int
				if err = db.QueryRow("SELECT COUNT(*) FROM " + name).Scan(&count); err != nil {
					t.Fatal(err)
				}
				if count != 1 {
					t.Fatalf("guard allowed partial deletion of %s", name)
				}
			}
			if err = deleteAllFromTables(path, []string{"categories"}); err != nil {
				t.Fatal("unrelated deletion blocked", err)
			}
		})
	}
}
