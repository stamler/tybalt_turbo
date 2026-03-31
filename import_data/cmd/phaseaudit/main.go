package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

var phaseTables = map[string][]string{
	"jobs": {
		"clients",
		"client_contacts",
		"client_notes",
		"jobs",
		"categories",
		"job_time_allocations",
	},
	"expenses": {
		"vendors",
		"purchase_orders",
		"expenses",
	},
	"time": {
		"time_sheets",
		"time_entries",
		"time_amendments",
	},
	"users": {
		"users",
		"profiles",
		"admin_profiles",
		"_externalAuths",
		"user_claims",
		"po_approver_props",
		"mileage_reset_dates",
	},
}

type config struct {
	before string
	after  string
	phases []string
}

type tableDiff struct {
	name         string
	err          error
	rowCountA    int64
	rowCountB    int64
	missingCount int64
	addedCount   int64
	changedCount int64
	missingKeys  []string
	addedKeys    []string
	changedKeys  []string
	fallbackNote string
}

func (d tableDiff) changed() bool {
	return d.err != nil ||
		d.rowCountA != d.rowCountB ||
		d.missingCount > 0 ||
		d.addedCount > 0 ||
		d.changedCount > 0 ||
		d.fallbackNote != ""
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	cfg, err := parseConfig(args)
	if err != nil {
		fmt.Fprintf(stderr, "phaseaudit: %v\n", err)
		return 2
	}

	db, err := sql.Open("sqlite", sqliteReadOnlyDSN(cfg.before))
	if err != nil {
		fmt.Fprintf(stderr, "phaseaudit: open before db: %v\n", err)
		return 2
	}
	defer db.Close()

	if _, err := db.Exec(`ATTACH DATABASE ? AS afterdb`, sqliteReadOnlyURI(cfg.after)); err != nil {
		fmt.Fprintf(stderr, "phaseaudit: attach after db: %v\n", err)
		return 2
	}
	defer db.Exec(`DETACH DATABASE afterdb`)

	overallChanged := false
	for _, phase := range cfg.phases {
		fmt.Fprintf(stdout, "phase %s\n", phase)
		phaseChanged := false
		for _, table := range phaseTables[phase] {
			diff := diffTable(db, table)
			if !diff.changed() {
				fmt.Fprintf(stdout, "  OK %s (%d rows)\n", table, diff.rowCountA)
				continue
			}

			phaseChanged = true
			overallChanged = true
			if diff.err != nil {
				fmt.Fprintf(stdout, "  CHANGED %s: %v\n", table, diff.err)
				continue
			}

			fmt.Fprintf(stdout, "  CHANGED %s: before=%d after=%d\n", table, diff.rowCountA, diff.rowCountB)
			if diff.missingCount > 0 {
				fmt.Fprintf(stdout, "    missing in after (%d): %s\n", diff.missingCount, strings.Join(diff.missingKeys, ", "))
			}
			if diff.addedCount > 0 {
				fmt.Fprintf(stdout, "    added in after (%d): %s\n", diff.addedCount, strings.Join(diff.addedKeys, ", "))
			}
			if diff.changedCount > 0 {
				fmt.Fprintf(stdout, "    changed rows (%d): %s\n", diff.changedCount, strings.Join(diff.changedKeys, ", "))
			}
			if diff.fallbackNote != "" {
				fmt.Fprintf(stdout, "    note: %s\n", diff.fallbackNote)
			}
		}

		if !phaseChanged {
			fmt.Fprintln(stdout, "  phase unchanged")
		}
	}

	if overallChanged {
		return 1
	}
	return 0
}

func parseConfig(args []string) (config, error) {
	var cfg config
	fs := flag.NewFlagSet("phaseaudit", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jobs := fs.Bool("jobs", false, "Check jobs tables")
	expenses := fs.Bool("expenses", false, "Check expenses tables")
	timeFlag := fs.Bool("time", false, "Check time tables")
	users := fs.Bool("users", false, "Check users tables")
	all := fs.Bool("all", false, "Check all phases")

	fs.StringVar(&cfg.before, "before", "", "Path to the pre-import SQLite database")
	fs.StringVar(&cfg.after, "after", "", "Path to the post-import SQLite database")

	if err := fs.Parse(args); err != nil {
		return cfg, err
	}

	if cfg.before == "" || cfg.after == "" {
		return cfg, errors.New("--before and --after are required")
	}

	if *all {
		cfg.phases = sortedPhaseNames()
		return cfg, nil
	}

	if *jobs {
		cfg.phases = append(cfg.phases, "jobs")
	}
	if *expenses {
		cfg.phases = append(cfg.phases, "expenses")
	}
	if *timeFlag {
		cfg.phases = append(cfg.phases, "time")
	}
	if *users {
		cfg.phases = append(cfg.phases, "users")
	}

	if len(cfg.phases) == 0 {
		return cfg, errors.New("select at least one phase flag: --jobs, --expenses, --time, --users, or --all")
	}

	return cfg, nil
}

func sortedPhaseNames() []string {
	names := make([]string, 0, len(phaseTables))
	for name := range phaseTables {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func diffTable(db *sql.DB, table string) tableDiff {
	diff := tableDiff{name: table}

	beforeCols, err := getTableColumns(db, "main", table)
	if err != nil {
		diff.err = err
		return diff
	}
	afterCols, err := getTableColumns(db, "afterdb", table)
	if err != nil {
		diff.err = err
		return diff
	}

	if len(beforeCols) == 0 || len(afterCols) == 0 {
		diff.err = fmt.Errorf("table missing in one database")
		return diff
	}
	if !sameColumnSet(beforeCols, afterCols) {
		diff.err = fmt.Errorf("column mismatch: before=%s after=%s", strings.Join(beforeCols, ","), strings.Join(afterCols, ","))
		return diff
	}

	diff.rowCountA, err = tableRowCount(db, "main", table)
	if err != nil {
		diff.err = err
		return diff
	}
	diff.rowCountB, err = tableRowCount(db, "afterdb", table)
	if err != nil {
		diff.err = err
		return diff
	}

	equal, err := tableDataEqual(db, table, beforeCols)
	if err != nil {
		diff.err = err
		return diff
	}
	if equal {
		return diff
	}

	keyCols, err := bestKeyColumns(db, table)
	if err != nil {
		diff.err = err
		return diff
	}
	if len(keyCols) == 0 {
		diff.fallbackNote = "table contents differ but no primary key or id column was available for row-level reporting"
		return diff
	}

	diff.missingCount, err = countMissingKeys(db, table, keyCols, "main", "afterdb")
	if err != nil {
		diff.err = err
		return diff
	}
	diff.missingKeys, err = sampleMissingKeys(db, table, keyCols, "main", "afterdb")
	if err != nil {
		diff.err = err
		return diff
	}
	diff.addedCount, err = countMissingKeys(db, table, keyCols, "afterdb", "main")
	if err != nil {
		diff.err = err
		return diff
	}
	diff.addedKeys, err = sampleMissingKeys(db, table, keyCols, "afterdb", "main")
	if err != nil {
		diff.err = err
		return diff
	}
	diff.changedCount, err = countChangedKeys(db, table, beforeCols, keyCols)
	if err != nil {
		diff.err = err
		return diff
	}
	diff.changedKeys, err = sampleChangedKeys(db, table, beforeCols, keyCols)
	if err != nil {
		diff.err = err
		return diff
	}

	return diff
}

func getTableColumns(db *sql.DB, schema, table string) ([]string, error) {
	query := fmt.Sprintf(`PRAGMA %s.table_info(%s)`, quoteIdent(schema), quoteString(table))
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type column struct {
		cid        int
		name       string
		ctype      string
		notNull    int
		defaultVal sql.NullString
		pk         int
	}

	var cols []column
	for rows.Next() {
		var c column
		if err := rows.Scan(&c.cid, &c.name, &c.ctype, &c.notNull, &c.defaultVal, &c.pk); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(cols, func(i, j int) bool { return cols[i].cid < cols[j].cid })
	names := make([]string, 0, len(cols))
	for _, c := range cols {
		names = append(names, c.name)
	}
	return names, nil
}

func sameColumnSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func tableRowCount(db *sql.DB, schema, table string) (int64, error) {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s.%s`, quoteIdent(schema), quoteIdent(table))
	var n int64
	err := db.QueryRow(query).Scan(&n)
	return n, err
}

func tableDataEqual(db *sql.DB, table string, cols []string) (bool, error) {
	groupCols := joinIdentifiers(cols)
	query := fmt.Sprintf(`
WITH before_rows AS (
	SELECT %s, COUNT(*) AS __row_count
	FROM main.%s
	GROUP BY %s
),
after_rows AS (
	SELECT %s, COUNT(*) AS __row_count
	FROM afterdb.%s
	GROUP BY %s
),
before_minus_after AS (
	SELECT %s, __row_count FROM before_rows
	EXCEPT
	SELECT %s, __row_count FROM after_rows
),
after_minus_before AS (
	SELECT %s, __row_count FROM after_rows
	EXCEPT
	SELECT %s, __row_count FROM before_rows
)
SELECT
	(SELECT COUNT(*) FROM before_minus_after) = 0
	AND
	(SELECT COUNT(*) FROM after_minus_before) = 0
`, groupCols, quoteIdent(table), groupCols, groupCols, quoteIdent(table), groupCols, groupCols, groupCols, groupCols, groupCols)

	var equal bool
	if err := db.QueryRow(query).Scan(&equal); err != nil {
		return false, err
	}
	return equal, nil
}

func bestKeyColumns(db *sql.DB, table string) ([]string, error) {
	query := fmt.Sprintf(`PRAGMA main.table_info(%s)`, quoteString(table))
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type keyCol struct {
		name string
		pk   int
		cid  int
	}

	var cols []keyCol
	var hasID bool
	for rows.Next() {
		var c keyCol
		var ctype string
		var notNull int
		var defaultVal sql.NullString
		if err := rows.Scan(&c.cid, &c.name, &ctype, &notNull, &defaultVal, &c.pk); err != nil {
			return nil, err
		}
		if c.name == "id" {
			hasID = true
		}
		if c.pk > 0 {
			cols = append(cols, c)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(cols) > 0 {
		sort.Slice(cols, func(i, j int) bool { return cols[i].pk < cols[j].pk })
		names := make([]string, 0, len(cols))
		for _, c := range cols {
			names = append(names, c.name)
		}
		return names, nil
	}
	if hasID {
		return []string{"id"}, nil
	}
	return nil, nil
}

func sampleMissingKeys(db *sql.DB, table string, keyCols []string, fromSchema, againstSchema string) ([]string, error) {
	keyExpr := joinIdentifiers(keyCols)
	query := fmt.Sprintf(`
SELECT %s
FROM %s.%s
EXCEPT
SELECT %s
FROM %s.%s
LIMIT 5
`, keyExpr, quoteIdent(fromSchema), quoteIdent(table), keyExpr, quoteIdent(againstSchema), quoteIdent(table))

	return scanKeyRows(db, query, keyCols)
}

func countMissingKeys(db *sql.DB, table string, keyCols []string, fromSchema, againstSchema string) (int64, error) {
	keyExpr := joinIdentifiers(keyCols)
	query := fmt.Sprintf(`
SELECT COUNT(*)
FROM (
	SELECT %s
	FROM %s.%s
	EXCEPT
	SELECT %s
	FROM %s.%s
)
`, keyExpr, quoteIdent(fromSchema), quoteIdent(table), keyExpr, quoteIdent(againstSchema), quoteIdent(table))

	var n int64
	err := db.QueryRow(query).Scan(&n)
	return n, err
}

func sampleChangedKeys(db *sql.DB, table string, allCols, keyCols []string) ([]string, error) {
	query, err := changedKeysQuery(table, allCols, keyCols, true)
	if err != nil {
		return nil, err
	}
	if query == "" {
		return nil, nil
	}

	return scanKeyRows(db, query, keyCols)
}

func countChangedKeys(db *sql.DB, table string, allCols, keyCols []string) (int64, error) {
	query, err := changedKeysQuery(table, allCols, keyCols, false)
	if err != nil {
		return 0, err
	}

	var n int64
	err = db.QueryRow(query).Scan(&n)
	return n, err
}

func changedKeysQuery(table string, allCols, keyCols []string, includeKeys bool) (string, error) {
	nonKeyCols := make([]string, 0, len(allCols))
	keySet := make(map[string]struct{}, len(keyCols))
	for _, col := range keyCols {
		keySet[col] = struct{}{}
	}
	for _, col := range allCols {
		if _, ok := keySet[col]; !ok {
			nonKeyCols = append(nonKeyCols, col)
		}
	}
	if len(nonKeyCols) == 0 {
		if includeKeys {
			return "", nil
		}
		return "SELECT 0", nil
	}

	selectClause := "COUNT(*)"
	limitClause := ""
	if includeKeys {
		selectClause = qualifiedColumns("b", keyCols)
		limitClause = "\nLIMIT 5"
	}
	onClause := joinConditions("b", "a", keyCols)
	diffClause := joinDifferencePredicates("b", "a", nonKeyCols)
	return fmt.Sprintf(`
SELECT %s
FROM main.%s b
JOIN afterdb.%s a ON %s
WHERE %s%s
`, selectClause, quoteIdent(table), quoteIdent(table), onClause, diffClause, limitClause), nil
}

func scanKeyRows(db *sql.DB, query string, keyCols []string) ([]string, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		values := make([]sql.NullString, len(keyCols))
		args := make([]any, len(keyCols))
		for i := range values {
			args[i] = &values[i]
		}
		if err := rows.Scan(args...); err != nil {
			return nil, err
		}
		parts := make([]string, 0, len(keyCols))
		for i, key := range keyCols {
			val := "NULL"
			if values[i].Valid {
				val = values[i].String
			}
			parts = append(parts, fmt.Sprintf("%s=%q", key, val))
		}
		out = append(out, strings.Join(parts, ","))
	}
	return out, rows.Err()
}

func joinIdentifiers(cols []string) string {
	parts := make([]string, 0, len(cols))
	for _, col := range cols {
		parts = append(parts, quoteIdent(col))
	}
	return strings.Join(parts, ", ")
}

func qualifiedColumns(alias string, cols []string) string {
	parts := make([]string, 0, len(cols))
	for _, col := range cols {
		parts = append(parts, alias+"."+quoteIdent(col))
	}
	return strings.Join(parts, ", ")
}

func joinConditions(leftAlias, rightAlias string, cols []string) string {
	parts := make([]string, 0, len(cols))
	for _, col := range cols {
		part := fmt.Sprintf("%s.%s = %s.%s", leftAlias, quoteIdent(col), rightAlias, quoteIdent(col))
		parts = append(parts, part)
	}
	return strings.Join(parts, " AND ")
}

func joinDifferencePredicates(leftAlias, rightAlias string, cols []string) string {
	parts := make([]string, 0, len(cols))
	for _, col := range cols {
		part := fmt.Sprintf("%s.%s IS NOT %s.%s", leftAlias, quoteIdent(col), rightAlias, quoteIdent(col))
		parts = append(parts, part)
	}
	return strings.Join(parts, " OR ")
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func quoteString(s string) string {
	return `'` + strings.ReplaceAll(s, `'`, `''`) + `'`
}

func sqliteReadOnlyDSN(path string) string {
	return sqliteReadOnlyURI(path)
}

func sqliteReadOnlyURI(path string) string {
	return "file:" + path + "?mode=ro"
}
