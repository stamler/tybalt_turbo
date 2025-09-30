package extract

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb"
)

// openDuckDB opens a DuckDB connection and applies conservative PRAGMAs
// to reduce peak memory usage and direct temporary spill files to a
// configurable directory. You can override defaults via environment vars:
// - DUCKDB_TMP_DIRECTORY: path to use for temp files (default: $TMPDIR/duckdb_tmp)
// - DUCKDB_MEMORY_LIMIT: memory cap, e.g. "1GB" (default: "1GB")
// - DUCKDB_THREADS: number of threads (default: "2")
func openDuckDB() (*sql.DB, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, err
	}

	tmpDir := os.Getenv("DUCKDB_TMP_DIRECTORY")
	if tmpDir == "" {
		base := filepath.Join(os.TempDir(), "duckdb_tmp")
		_ = os.MkdirAll(base, 0o755)
		tmpDir = base
	} else {
		_ = os.MkdirAll(tmpDir, 0o755)
	}

	mem := os.Getenv("DUCKDB_MEMORY_LIMIT")
	if mem == "" {
		mem = "1GB"
	}

	threads := os.Getenv("DUCKDB_THREADS")
	if threads == "" {
		threads = "2"
	}

	if _, err := db.Exec(fmt.Sprintf("PRAGMA temp_directory='%s';", tmpDir)); err != nil {
		log.Printf("warning: failed to set DuckDB temp_directory: %v", err)
	}
	if _, err := db.Exec(fmt.Sprintf("PRAGMA memory_limit='%s';", mem)); err != nil {
		log.Printf("warning: failed to set DuckDB memory_limit: %v", err)
	}
	if _, err := db.Exec(fmt.Sprintf("PRAGMA threads=%s;", threads)); err != nil {
		log.Printf("warning: failed to set DuckDB threads: %v", err)
	}

	return db, nil
}
