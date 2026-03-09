// Package testseed builds and manages the text-backed fixture database used by
// tests.
//
// Phase 1 replaces direct per-test dependence on the committed
// app/test_pb_data/data.db binary with a workflow that:
//   - dumps the current canonical test data to text files under app/test_seed_data
//   - rebuilds a PocketBase data directory from migrations plus those text files
//   - caches one migrated-and-seeded template directory per test process
//   - gives each test its own cloned TestApp from that cached template
//
// The package also provides dump/load/verify helpers for maintaining the text
// seed data. Verify intentionally compares only the columns represented in the
// datapackage resources so it remains stable even when a fresh migration run
// adds defaulted columns that don't exist in the legacy source DB.
package testseed

import (
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
	_ "tybalt/migrations"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const (
	nullSentinel    = `\N`
	testGroup       = "test-full"
	migrationsTable = "_migrations"
)

// DataPackage is the top-level manifest written to datapackage.json.
type DataPackage struct {
	Name      string     `json:"name"`
	Resources []Resource `json:"resources"`
}

// Resource describes one table export inside datapackage.json.
type Resource struct {
	Name    string         `json:"name"`
	Path    string         `json:"path"`
	Schema  ResourceSchema `json:"schema"`
	XGroups []string       `json:"x-groups,omitempty"`
}

// ResourceSchema records the exported columns and primary key for one resource.
type ResourceSchema struct {
	Fields     []Field  `json:"fields"`
	PrimaryKey []string `json:"primaryKey,omitempty"`
}

// Field describes one exported column in a resource schema.
type Field struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	XSQLiteType string `json:"x-sqlite-type,omitempty"`
}

type tableInfo struct {
	CID        int
	Name       string
	Type       string
	NotNull    int
	DefaultVal sql.NullString
	PK         int
}

type verifyRow struct {
	Table     string
	RowCount  int
	RowDigest string
}

var (
	templateOnce sync.Once
	templateDir  string
	templateErr  error
)

// PackageRoot returns the absolute path to the app/ directory that owns this
// package.
func PackageRoot() string {
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}

// DefaultSeedDir returns the repository path that stores the canonical text
// fixtures consumed by this package.
func DefaultSeedDir() string {
	return filepath.Join(PackageRoot(), "test_seed_data")
}

// NewSeededTestApp returns a new PocketBase test app cloned from the cached
// migrated-and-seeded template directory.
//
// Each call still creates an isolated TestApp clone; only the expensive
// migrations-plus-load step is cached.
func NewSeededTestApp(tb testing.TB) *tests.TestApp {
	tb.Helper()

	dir := EnsureTemplateDir(tb)
	app, err := tests.NewTestApp(dir)
	if err != nil {
		tb.Fatal(err)
	}
	return app
}

// EnsureTemplateDir returns the cached template directory or fails the test if
// template creation was unsuccessful.
func EnsureTemplateDir(tb testing.TB) string {
	tb.Helper()

	dir, err := TemplateDir()
	if err != nil {
		tb.Fatal(err)
	}

	return dir
}

// TemplateDir returns the cached template directory used as the source for all
// seeded TestApp clones in the current process.
//
// The directory is created once on first use by running the current migration
// set against a fresh DB and then loading the test-full seed group from
// DefaultSeedDir.
func TemplateDir() (string, error) {
	templateOnce.Do(func() {
		dir, err := os.MkdirTemp("", "tybalt-testseed-template-*")
		if err != nil {
			templateErr = err
			return
		}

		if err := BuildSeededDataDir(dir, DefaultSeedDir()); err != nil {
			templateErr = err
			return
		}

		templateDir = dir
	})

	return templateDir, templateErr
}

// BuildSeededDataDir creates a PocketBase data directory at dataDir by running
// all app migrations against a fresh database and then loading the selected text
// fixtures from seedDir.
//
// The resulting directory can be used as the source for tests.NewTestApp.
func BuildSeededDataDir(dataDir string, seedDir string) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}

	app := core.NewBaseApp(core.BaseAppConfig{
		DataDir:       dataDir,
		EncryptionEnv: "pb_test_env",
	})

	if err := app.Bootstrap(); err != nil {
		return err
	}

	if _, err := app.DB().NewQuery("SELECT 1").Execute(); err != nil {
		return err
	}
	if _, err := app.AuxDB().NewQuery("SELECT 1").Execute(); err != nil {
		return err
	}

	if err := app.RunAllMigrations(); err != nil {
		return err
	}

	if err := app.ResetBootstrapState(); err != nil {
		return err
	}

	return LoadSeedData(filepath.Join(dataDir, "data.db"), seedDir, testGroup)
}

// DumpSeedData exports every table in sourceDBPath to CSV files plus a
// datapackage.json manifest under seedDir.
//
// Column order is canonicalized alphabetically so dumps and verification remain
// stable across schema histories that produce the same logical columns in a
// different physical order.
//
// The PocketBase migrations table is intentionally excluded because migration
// state should come from the current code, not from fixture data.
func DumpSeedData(sourceDBPath string, seedDir string) error {
	db, err := sql.Open("sqlite", sourceDBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	tables, err := listTables(db)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(seedDir, 0o755); err != nil {
		return err
	}

	dataDir := filepath.Join(seedDir, "data")
	if err := os.RemoveAll(dataDir); err != nil {
		return err
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}

	pkg := DataPackage{
		Name: "tybalt-test-seed-data",
	}

	for _, table := range tables {
		info, err := getTableInfo(db, table)
		if err != nil {
			return err
		}

		csvPath := filepath.Join(dataDir, table+".csv")
		if err := dumpTableCSV(db, table, info, csvPath); err != nil {
			return err
		}

		resource := Resource{
			Name:    table,
			Path:    filepath.ToSlash(filepath.Join("data", table+".csv")),
			Schema:  buildSchema(info),
			XGroups: []string{testGroup},
		}
		pkg.Resources = append(pkg.Resources, resource)
	}

	pkgBytes, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return err
	}

	pkgBytes = append(pkgBytes, '\n')
	return os.WriteFile(filepath.Join(seedDir, "datapackage.json"), pkgBytes, 0o644)
}

// DumpSeedDataFromTestApp boots a test app from sourceDataDir and dumps the
// migrated runtime database state to seedDir.
//
// This is preferred over dumping the raw source DB directly because the
// committed app/test_pb_data directory may lag behind the schema shape produced
// by current migrations.
func DumpSeedDataFromTestApp(sourceDataDir string, seedDir string) error {
	app, err := tests.NewTestApp(sourceDataDir)
	if err != nil {
		return err
	}
	defer app.Cleanup()

	return DumpSeedData(filepath.Join(app.DataDir(), "data.db"), seedDir)
}

// LoadSeedData replaces the contents of the selected seed group tables in dbPath
// with the CSV data described by seedDir/datapackage.json.
//
// The loader disables foreign-key enforcement during bulk replacement and then
// runs PRAGMA foreign_key_check afterward to ensure the final state is valid.
//
// The datapackage must not include the PocketBase migrations table; migration
// state is owned by current code and bootstrap, not by fixture rows.
func LoadSeedData(dbPath string, seedDir string, group string) error {
	pkg, err := readPackage(seedDir)
	if err != nil {
		return err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	resources := filterResources(pkg.Resources, group)

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		return err
	}

	for _, resource := range resources {
		if _, err := tx.Exec("DELETE FROM " + quoteIdent(resource.Name)); err != nil {
			return fmt.Errorf("clear %s: %w", resource.Name, err)
		}
	}

	for _, resource := range resources {
		if err := loadResourceCSV(tx, seedDir, resource); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	rows, err := db.Query("PRAGMA foreign_key_check")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var table string
		var rowID sql.NullInt64
		var parent string
		var fkID sql.NullInt64
		if err := rows.Scan(&table, &rowID, &parent, &fkID); err != nil {
			return err
		}
		return fmt.Errorf("foreign key check failed: table=%s rowid=%v parent=%s fk=%v", table, rowID, parent, fkID)
	}

	return rows.Err()
}

// VerifySeedData rebuilds a fresh PocketBase data directory from seedDir and
// checks that the resource columns and row contents match sourceDBPath.
//
// Verification is scoped to the resource columns declared in datapackage.json
// rather than the full table schemas so it remains resilient to additive schema
// differences introduced by fresh migrations.
func VerifySeedData(sourceDBPath string, seedDir string) error {
	pkg, err := readPackage(seedDir)
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "tybalt-testseed-verify-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	if err := BuildSeededDataDir(tempDir, seedDir); err != nil {
		return err
	}

	targetDBPath := filepath.Join(tempDir, "data.db")
	resources := filterResources(pkg.Resources, testGroup)

	sourceRows, err := collectVerifyRows(sourceDBPath, resources)
	if err != nil {
		return err
	}
	targetRows, err := collectVerifyRows(targetDBPath, resources)
	if err != nil {
		return err
	}

	if len(sourceRows) != len(targetRows) {
		return fmt.Errorf("table count mismatch: source=%d target=%d", len(sourceRows), len(targetRows))
	}

	for i := range sourceRows {
		if sourceRows[i] != targetRows[i] {
			return fmt.Errorf(
				"verify mismatch for %s: source(count=%d digest=%s) target(count=%d digest=%s)",
				sourceRows[i].Table,
				sourceRows[i].RowCount,
				sourceRows[i].RowDigest,
				targetRows[i].RowCount,
				targetRows[i].RowDigest,
			)
		}
	}

	return nil
}

// VerifySeedDataAgainstTestApp boots a test app from sourceDataDir and verifies
// that seedDir recreates the same migrated runtime data represented by the seed
// package.
func VerifySeedDataAgainstTestApp(sourceDataDir string, seedDir string) error {
	app, err := tests.NewTestApp(sourceDataDir)
	if err != nil {
		return err
	}
	defer app.Cleanup()

	return VerifySeedData(filepath.Join(app.DataDir(), "data.db"), seedDir)
}

func readPackage(seedDir string) (*DataPackage, error) {
	raw, err := os.ReadFile(filepath.Join(seedDir, "datapackage.json"))
	if err != nil {
		return nil, err
	}

	var pkg DataPackage
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return nil, err
	}

	return &pkg, nil
}

func filterResources(resources []Resource, group string) []Resource {
	if group == "" {
		return slices.Clone(resources)
	}

	var result []Resource
	for _, resource := range resources {
		if len(resource.XGroups) == 0 || slices.Contains(resource.XGroups, group) {
			result = append(result, resource)
		}
	}

	return result
}

func dumpTableCSV(db *sql.DB, table string, info []tableInfo, csvPath string) error {
	file, err := os.Create(csvPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	columns := sortedColumns(info)
	headers := make([]string, 0, len(columns))
	for _, column := range columns {
		headers = append(headers, column.Name)
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	query := buildSelectQuery(table, info)
	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		values := make([]any, len(info))
		scanArgs := make([]any, len(info))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return err
		}

		record := make([]string, len(values))
		for i, value := range values {
			record[i] = normalizeValue(value)
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}

	writer.Flush()
	return writer.Error()
}

func loadResourceCSV(tx *sql.Tx, seedDir string, resource Resource) error {
	file, err := os.Open(filepath.Join(seedDir, filepath.FromSlash(resource.Path)))
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return err
	}

	quotedColumns := make([]string, len(headers))
	placeholders := make([]string, len(headers))
	for i, header := range headers {
		quotedColumns[i] = quoteIdent(header)
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		quoteIdent(resource.Name),
		strings.Join(quotedColumns, ", "),
		strings.Join(placeholders, ", "),
	)

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("prepare insert for %s: %w", resource.Name, err)
	}
	defer stmt.Close()

	for {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("read csv row for %s: %w", resource.Name, err)
		}

		args := make([]any, len(record))
		for i, value := range record {
			if value == nullSentinel {
				args[i] = nil
			} else {
				args[i] = value
			}
		}

		if _, err := stmt.Exec(args...); err != nil {
			return fmt.Errorf("insert into %s: %w", resource.Name, err)
		}
	}

	return nil
}

func collectVerifyRows(dbPath string, resources []Resource) ([]verifyRow, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var result []verifyRow
	for _, resource := range resources {
		info, err := getTableInfo(db, resource.Name)
		if err != nil {
			return nil, err
		}

		columns := make([]string, 0, len(resource.Schema.Fields))
		for _, field := range resource.Schema.Fields {
			columns = append(columns, field.Name)
		}

		rows, err := db.Query(buildSelectQueryWithColumns(resource.Name, info, columns))
		if err != nil {
			return nil, err
		}

		hasher := sha256.New()
		count := 0
		for rows.Next() {
			values := make([]any, len(columns))
			scanArgs := make([]any, len(columns))
			for i := range values {
				scanArgs[i] = &values[i]
			}
			if err := rows.Scan(scanArgs...); err != nil {
				rows.Close()
				return nil, err
			}
			for _, value := range values {
				hasher.Write([]byte(normalizeValue(value)))
				hasher.Write([]byte{0})
			}
			hasher.Write([]byte{'\n'})
			count++
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()

		result = append(result, verifyRow{
			Table:     resource.Name,
			RowCount:  count,
			RowDigest: hex.EncodeToString(hasher.Sum(nil)),
		})
	}

	return result, nil
}

func listTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`
		SELECT name
		FROM sqlite_master
		WHERE type = 'table'
		  AND name NOT LIKE 'sqlite_%'
		  AND name != ?
		ORDER BY name
	`, migrationsTable)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}

	return tables, rows.Err()
}

func getTableInfo(db *sql.DB, table string) ([]tableInfo, error) {
	rows, err := db.Query("PRAGMA table_info(" + quoteLiteral(table) + ")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var info []tableInfo
	for rows.Next() {
		var column tableInfo
		if err := rows.Scan(
			&column.CID,
			&column.Name,
			&column.Type,
			&column.NotNull,
			&column.DefaultVal,
			&column.PK,
		); err != nil {
			return nil, err
		}
		info = append(info, column)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return info, nil
}

func buildSchema(info []tableInfo) ResourceSchema {
	columns := sortedColumns(info)
	fields := make([]Field, 0, len(columns))
	var primaryKey []string
	for _, column := range columns {
		fields = append(fields, Field{
			Name:        column.Name,
			Type:        frictionlessType(column.Type),
			XSQLiteType: column.Type,
		})
		if column.PK > 0 {
			primaryKey = append(primaryKey, column.Name)
		}
	}

	slices.SortFunc(primaryKey, func(a string, b string) int {
		return comparePKPosition(info, a, b)
	})

	return ResourceSchema{
		Fields:     fields,
		PrimaryKey: primaryKey,
	}
}

func comparePKPosition(info []tableInfo, a string, b string) int {
	var aPK, bPK int
	for _, column := range info {
		if column.Name == a {
			aPK = column.PK
		}
		if column.Name == b {
			bPK = column.PK
		}
	}
	return aPK - bPK
}

func frictionlessType(sqliteType string) string {
	upper := strings.ToUpper(sqliteType)

	switch {
	case strings.Contains(upper, "BOOL"):
		return "boolean"
	case strings.Contains(upper, "INT"):
		return "integer"
	case strings.Contains(upper, "REAL"),
		strings.Contains(upper, "FLOA"),
		strings.Contains(upper, "DOUB"),
		strings.Contains(upper, "NUMERIC"),
		strings.Contains(upper, "DECIMAL"):
		return "number"
	default:
		return "string"
	}
}

func buildSelectQuery(table string, info []tableInfo) string {
	columns := make([]string, 0, len(info))
	for _, column := range sortedColumns(info) {
		columns = append(columns, column.Name)
	}

	return buildSelectQueryWithColumns(table, info, columns)
}

func buildSelectQueryWithColumns(table string, info []tableInfo, columns []string) string {
	quoted := make([]string, 0, len(columns))
	for _, column := range columns {
		quoted = append(quoted, quoteIdent(column))
	}

	orderBy := buildOrderBy(info)
	return fmt.Sprintf(
		"SELECT %s FROM %s ORDER BY %s",
		strings.Join(quoted, ", "),
		quoteIdent(table),
		orderBy,
	)
}

func sortedColumns(info []tableInfo) []tableInfo {
	columns := slices.Clone(info)
	slices.SortFunc(columns, func(a tableInfo, b tableInfo) int {
		return strings.Compare(a.Name, b.Name)
	})
	return columns
}

func buildOrderBy(info []tableInfo) string {
	var pkColumns []tableInfo
	for _, column := range info {
		if column.PK > 0 {
			pkColumns = append(pkColumns, column)
		}
	}

	if len(pkColumns) > 0 {
		slices.SortFunc(pkColumns, func(a tableInfo, b tableInfo) int {
			return a.PK - b.PK
		})

		quoted := make([]string, 0, len(pkColumns))
		for _, column := range pkColumns {
			quoted = append(quoted, quoteIdent(column.Name))
		}
		return strings.Join(quoted, ", ")
	}

	return "rowid"
}

func normalizeValue(value any) string {
	switch v := value.(type) {
	case nil:
		return nullSentinel
	case []byte:
		return normalizeString(string(v))
	case string:
		return normalizeString(v)
	case bool:
		if v {
			return "1"
		}
		return "0"
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64)
	default:
		return normalizeString(fmt.Sprint(v))
	}
}

func normalizeString(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.ReplaceAll(value, "\r", "\n")
}

func quoteIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func quoteLiteral(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}
