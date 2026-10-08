package load

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type LocalDataImportReader interface{ QueryRow(string, ...any) *sql.Row }

// CheckLocalDataImport stops bulk replacement of jobs, clients, and contacts
// once local records depend on them. The importer cannot restore locally
// maintained invoicing profiles, job requests, or retained project approvals,
// or their links. Imports also stop while project request approval is
// required, because imported projects would bypass it. Check within the same
// transaction as each affected import or deletion.
func CheckLocalDataImport(db LocalDataImportReader) error {
	enabled, err := requestApprovalRequired(db)
	if err != nil {
		return err
	}
	if enabled {
		return fmt.Errorf("job/client/contact import is disabled while project request approval is required")
	}
	for _, table := range []string{"client_invoicing_information", "job_requests", "job_attachments"} {
		count, err := countRows(db, table)
		if err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("job/client/contact import would replace records used by %s", table)
		}
	}
	return nil
}

func tableExists(db LocalDataImportReader, table string) (bool, error) {
	var exists int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&exists)
	return exists > 0, err
}

// countRows returns 0 for a table that does not exist yet.
func countRows(db LocalDataImportReader, table string) (int, error) {
	exists, err := tableExists(db, table)
	if err != nil || !exists {
		return 0, err
	}
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
	return count, err
}

func requestApprovalRequired(db LocalDataImportReader) (bool, error) {
	exists, err := tableExists(db, "app_config")
	if err != nil || !exists {
		return false, err
	}
	var raw string
	err = db.QueryRow("SELECT value FROM app_config WHERE key='jobs'").Scan(&raw)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var config map[string]any
	if err = json.Unmarshal([]byte(raw), &config); err != nil {
		return false, err
	}
	value, set := config["require_job_request_approval"]
	if !set {
		return false, nil
	}
	enabled, valid := value.(bool)
	if !valid {
		return false, fmt.Errorf("invalid project request approval setting")
	}
	return enabled, nil
}
