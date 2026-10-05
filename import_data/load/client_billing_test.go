package load

import (
	"database/sql"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/pocketbase/dbx"
)

func billingCSV(t *testing.T, name string) [][]string {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return rows[1:]
}

func TestClientBillingImportGuard(t *testing.T) {
	for _, row := range billingCSV(t, "client_billing.csv") {
		t.Run(row[0], func(t *testing.T) {
			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			db.SetMaxOpenConns(1)
			if row[1] == "true" {
				if _, err = db.Exec("CREATE TABLE client_invoicing_information (id TEXT)"); err != nil {
					t.Fatal(err)
				}
				if row[2] != "" {
					if _, err = db.Exec("INSERT INTO client_invoicing_information VALUES (?)", row[2]); err != nil {
						t.Fatal(err)
					}
				}
			}
			tx, err := db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			if blocked := CheckClientBillingImport(tx) != nil; blocked != (row[3] == "true") {
				t.Fatalf("blocked=%v", blocked)
			}
		})
	}
}

type billingImportRow struct {
	ID    string `parquet:"id"`
	Value string `parquet:"value"`
}

func TestClientBillingImportPreservesExistingRowsAndLegacyBehavior(t *testing.T) {
	rows := []billingImportRow{}
	for _, row := range billingCSV(t, "client_billing_rows.csv") {
		rows = append(rows, billingImportRow{ID: row[0], Value: row[1]})
	}
	for _, table := range []string{"jobs", "clients", "client_contacts", "unrelated"} {
		for _, profiles := range []bool{false, true} {
			t.Run(table+map[bool]string{false: " empty", true: " used"}[profiles], func(t *testing.T) {
				dir := t.TempDir()
				dbPath := filepath.Join(dir, "data.db")
				parquetPath := filepath.Join(dir, "rows.parquet")
				if err := parquet.WriteFile(parquetPath, rows); err != nil {
					t.Fatal(err)
				}
				db, err := sql.Open("sqlite", dbPath)
				if err != nil {
					t.Fatal(err)
				}
				defer db.Close()
				if _, err = db.Exec("CREATE TABLE " + table + " (id TEXT PRIMARY KEY,value TEXT)"); err != nil {
					t.Fatal(err)
				}
				if _, err = db.Exec("CREATE TABLE client_invoicing_information (id TEXT)"); err != nil {
					t.Fatal(err)
				}
				if profiles {
					if _, err = db.Exec("INSERT INTO client_invoicing_information VALUES (?)", billingCSV(t, "client_billing.csv")[2][2]); err != nil {
						t.Fatal(err)
					}
				}
				var failure any
				func() {
					defer func() { failure = recover() }()
					FromParquet[billingImportRow](parquetPath, dbPath, table, "INSERT INTO "+table+" (id,value) VALUES ({:id},{:value})", func(row billingImportRow) dbx.Params { return dbx.Params{"id": row.ID, "value": row.Value} }, false)
				}()
				blocked := profiles && table != "unrelated"
				if (failure != nil) != blocked {
					t.Fatalf("blocked=%v, panic=%v", blocked, failure)
				}
				var count int
				if err = db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
					t.Fatal(err)
				}
				if blocked {
					if count != 0 {
						t.Fatal("guard allowed partial writes")
					}
					return
				}
				if count != 2 {
					t.Fatalf("legacy imports must continue past duplicate-row failure: got %d rows", count)
				}
				var value string
				if err = db.QueryRow("SELECT value FROM "+table+" WHERE id=?", rows[0].ID).Scan(&value); err != nil {
					t.Fatal(err)
				}
				if value != rows[0].Value {
					t.Fatal("non-upsert import replaced original row")
				}
			})
		}
	}
}
