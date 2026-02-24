package extract

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/marcboeker/go-duckdb" // Import DuckDB driver
)

// expensesToPurchaseOrders normalizes the Expenses.parquet data by:
// 1. Creating a separate purchase_orders.parquet file with unique purchase_order records
// 2. Updating Expenses.parquet to reference purchase orders via foreign keys
//
// HYBRID ID RESOLUTION (similar to expensesToVendors and jobs_to_clients_and_contacts):
// When TurboPurchaseOrders.parquet exists, PO IDs are resolved using this priority:
//  1. EXACT PO NUMBER MATCH: If exactly ONE TurboPurchaseOrder exists with the same
//     poNumber, use that TurboPurchaseOrder's ID and fields. This preserves PocketBase
//     IDs for POs that were written back from Turbo.
//  2. GENERATED ID: Otherwise, generate a deterministic ID from MD5 hash of PO number.
//
// This ensures:
// - Turbo-originated POs keep their PocketBase IDs on re-import
// - Legacy-only POs get consistent generated IDs
// - Expenses link to the correct PO IDs
// - Turbo-only POs (with no expenses) are also included
func expensesToPurchaseOrders() {

	db, err := openDuckDB()
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create deterministic ID generation macro in DuckDB
	_, err = db.Exec(`
CREATE OR REPLACE MACRO make_pocketbase_id(source_value, length)
AS substr(md5(CAST(source_value AS VARCHAR)), 1, length);
`)
	if err != nil {
		log.Fatalf("Failed to create make_pocketbase_id macro: %v", err)
	}

	// Check if TurboPurchaseOrders.parquet exists for hybrid ID resolution
	turboPOsExists := fileExists("parquet/TurboPurchaseOrders.parquet")

	if turboPOsExists {
		log.Println("TurboPurchaseOrders.parquet found - will use hybrid ID resolution for purchase orders")
		if err := prepareTurboPOsTable(db); err != nil {
			log.Fatalf("Failed to prepare TurboPurchaseOrders kind data: %v", err)
		}
	} else {
		log.Println("TurboPurchaseOrders.parquet not found - will generate all PO IDs from PO numbers")
	}

	query := buildPurchaseOrderQuery(turboPOsExists)
	query = strings.ReplaceAll(query, "{{CAPITAL_KIND_ID}}", CapitalExpenditureKindID())
	query = strings.ReplaceAll(query, "{{PROJECT_KIND_ID}}", ProjectExpenditureKindID())

	_, err = db.Exec(query)
	if err != nil {
		log.Fatalf("Failed to execute purchase order extraction query: %v", err)
	}

	if turboPOsExists {
		// Fail fast when TurboPurchaseOrders user references cannot map to PocketBase UIDs.
		if err := validateTurboPOUserMappings(db); err != nil {
			log.Fatalf("TurboPurchaseOrders UID mapping validation failed: %v", err)
		}
	}
}

func prepareTurboPOsTable(db *sql.DB) error {
	_, err := db.Exec("CREATE TABLE turbo_pos AS SELECT * FROM read_parquet('parquet/TurboPurchaseOrders.parquet')")
	if err != nil {
		return fmt.Errorf("create turbo_pos table: %w", err)
	}

	_, err = db.Exec("ALTER TABLE turbo_pos ADD COLUMN kind TEXT")
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "already exists") {
		return fmt.Errorf("add turbo_pos.kind column: %w", err)
	}

	_, err = db.Exec(fmt.Sprintf(
		"UPDATE turbo_pos SET kind = CASE WHEN job IS NOT NULL AND TRIM(CAST(job AS VARCHAR)) != '' THEN '%s' ELSE '%s' END WHERE kind IS NULL OR TRIM(CAST(kind AS VARCHAR)) = ''",
		ProjectExpenditureKindID(),
		CapitalExpenditureKindID(),
	))
	if err != nil {
		return fmt.Errorf("backfill turbo_pos.kind values: %w", err)
	}

	return nil
}

func buildPurchaseOrderQuery(turboPOsExists bool) string {
	query := `
		-- Load Expenses.parquet into a table called expenses
		CREATE TABLE expenses AS
		SELECT * FROM read_parquet('parquet/Expenses.parquet');

		-- Normalize historical plain-digit PO numbers (4-5 digits) by prefixing YY- from the date
		-- - Trim whitespace and cast PO to text to handle numeric types and stray spaces
		-- - Only update rows where trimmed PO is exactly 4-5 digits and date is present
		UPDATE expenses
		SET po = concat(
			strftime(CAST(date AS DATE), '%y'),
			'-',
			TRIM(CAST(po AS VARCHAR))
		)
		WHERE po IS NOT NULL AND date IS NOT NULL
		  AND regexp_matches(TRIM(CAST(po AS VARCHAR)), '^[0-9]{4,5}$');

		-- Establish the first (earliest) expense row per PO for representative fields
		CREATE TABLE first_expense_per_po AS
		SELECT * FROM (
		  SELECT e.*,
		         row_number() OVER (
		             PARTITION BY po
		             ORDER BY date, pocketbase_id
		         ) AS rn
		  FROM expenses e
		  WHERE po IS NOT NULL AND po != ''
		) WHERE rn = 1;

		-- Sum totals per PO (raw units as in Expenses.parquet; conversion happens on import)
		CREATE TABLE sum_total_per_po AS
		SELECT po, CAST(SUM(total) AS DOUBLE) AS total
		FROM expenses
		WHERE po IS NOT NULL AND po != ''
		GROUP BY po;
`

	if turboPOsExists {
		query += `
		-- turbo_pos is preloaded in prepareTurboPOsTable(), including kind defaults

		-- Load Profiles.parquet to convert legacy UIDs to PocketBase UIDs
		-- TurboPurchaseOrders has legacy Firebase UIDs (from writeback), we need PocketBase UIDs
		CREATE TABLE profiles AS
		SELECT * FROM read_parquet('parquet/Profiles.parquet');

		-- Create purchase_orders with hybrid ID resolution
		-- Priority 1: Exact PO number match against TurboPurchaseOrders (if exactly 1 match) - use preserved IDs
		-- Priority 2: Generate ID from MD5 hash of PO number
		-- For TurboPurchaseOrders: use their vendor ID directly (it already exists in Vendors.parquet via TurboVendors)
		-- For derived POs: use expense vendor_id (already resolved by expensesToVendors())
		CREATE TABLE purchase_orders AS
		SELECT
			-- ID: prefer TurboPurchaseOrders ID when matched by PO number
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.id FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE make_pocketbase_id(fe.po, 15)
			END AS id,
			fe.po AS number,
			-- Approver: prefer TurboPurchaseOrders when matched, but convert legacy UID to PocketBase UID
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT p.pocketbase_uid FROM profiles p WHERE p.id = (SELECT tp.approver FROM turbo_pos tp WHERE tp.number = fe.po))
				ELSE fe.pocketbase_approver_uid
			END AS approver,
			-- Date: prefer TurboPurchaseOrders when matched (cast to VARCHAR for consistency)
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.date FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE CAST(fe.date AS VARCHAR)
			END AS date,
			-- Vendor: use TurboPurchaseOrders vendor ID directly when matched
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.vendor FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE fe.vendor_id
			END AS vendor,
			-- Uid: prefer TurboPurchaseOrders when matched, but convert legacy UID to PocketBase UID
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT p.pocketbase_uid FROM profiles p WHERE p.id = (SELECT tp.uid FROM turbo_pos tp WHERE tp.number = fe.po))
				ELSE fe.pocketbase_uid
			END AS uid,
			-- Total: prefer TurboPurchaseOrders when matched (already in correct units)
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.total FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE st.total
			END AS total,
			-- Approval total: only from TurboPurchaseOrders
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.approval_total FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE NULL
			END AS approval_total,
			-- Payment type: prefer TurboPurchaseOrders when matched
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.payment_type FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE fe.paymentType
			END AS payment_type,
			-- Job: prefer TurboPurchaseOrders when matched
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.job FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE fe.pocketbase_jobid
			END AS job,
			-- Description: only from TurboPurchaseOrders
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.description FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE NULL
			END AS description,
			-- Division: prefer TurboPurchaseOrders when matched
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.division FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE fe.division_id
			END AS division,
			-- Type: only from TurboPurchaseOrders (legacy defaults to NULL, import handles default)
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.type FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE NULL
			END AS type,
			-- Frequency: only from TurboPurchaseOrders
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.frequency FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE NULL
			END AS frequency,
			-- Status: only from TurboPurchaseOrders (legacy defaults to NULL, import handles default)
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.status FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE NULL
			END AS status,
			-- Approved timestamp: only from TurboPurchaseOrders
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.approved FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE NULL
			END AS approved,
			-- Second approval timestamp: only from TurboPurchaseOrders
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.second_approval FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE NULL
			END AS second_approval,
			-- Second approver: convert legacy UID to PocketBase UID
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT p.pocketbase_uid FROM profiles p WHERE p.id = (SELECT tp.second_approver FROM turbo_pos tp WHERE tp.number = fe.po))
				ELSE NULL
			END AS second_approver,
			-- Priority second approver: convert legacy UID to PocketBase UID
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT p.pocketbase_uid FROM profiles p WHERE p.id = (SELECT tp.priority_second_approver FROM turbo_pos tp WHERE tp.number = fe.po))
				ELSE NULL
			END AS priority_second_approver,
			-- Closed timestamp: only from TurboPurchaseOrders
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.closed FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE NULL
			END AS closed,
			-- Closer: convert legacy UID to PocketBase UID
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT p.pocketbase_uid FROM profiles p WHERE p.id = (SELECT tp.closer FROM turbo_pos tp WHERE tp.number = fe.po))
				ELSE NULL
			END AS closer,
			-- Cancelled timestamp: only from TurboPurchaseOrders
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.cancelled FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE NULL
			END AS cancelled,
			-- Canceller: convert legacy UID to PocketBase UID
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT p.pocketbase_uid FROM profiles p WHERE p.id = (SELECT tp.canceller FROM turbo_pos tp WHERE tp.number = fe.po))
				ELSE NULL
			END AS canceller,
			-- Rejected timestamp: only from TurboPurchaseOrders
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.rejected FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE NULL
			END AS rejected,
			-- Rejector: convert legacy UID to PocketBase UID
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT p.pocketbase_uid FROM profiles p WHERE p.id = (SELECT tp.rejector FROM turbo_pos tp WHERE tp.number = fe.po))
				ELSE NULL
			END AS rejector,
			-- Rejection reason: only from TurboPurchaseOrders
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.rejection_reason FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE NULL
			END AS rejection_reason,
			-- End date: only from TurboPurchaseOrders (for recurring POs)
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.end_date FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE NULL
			END AS end_date,
			-- Category: only from TurboPurchaseOrders (PocketBase ID, no conversion needed)
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.category FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE NULL
			END AS category,
			-- Kind: prefer TurboPurchaseOrders kind when present, otherwise job-aware default
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN COALESCE(
					NULLIF((SELECT tp.kind FROM turbo_pos tp WHERE tp.number = fe.po), ''),
					NULLIF(fe.kind, ''),
					CASE WHEN COALESCE(
						(SELECT tp.job FROM turbo_pos tp WHERE tp.number = fe.po),
						fe.pocketbase_jobid
					) IS NOT NULL AND TRIM(CAST(COALESCE(
						(SELECT tp.job FROM turbo_pos tp WHERE tp.number = fe.po),
						fe.pocketbase_jobid
					) AS VARCHAR)) != '' THEN '{{PROJECT_KIND_ID}}' ELSE '{{CAPITAL_KIND_ID}}' END
				)
				ELSE COALESCE(
					NULLIF(fe.kind, ''),
					CASE WHEN fe.pocketbase_jobid IS NOT NULL AND TRIM(CAST(fe.pocketbase_jobid AS VARCHAR)) != '' THEN '{{PROJECT_KIND_ID}}' ELSE '{{CAPITAL_KIND_ID}}' END
				)
			END AS kind,
			-- Parent PO: only from TurboPurchaseOrders (PocketBase ID, no conversion needed)
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.parent_po FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE NULL
			END AS parent_po,
			-- Branch: only from TurboPurchaseOrders (PocketBase ID, no conversion needed)
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.branch FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE NULL
			END AS branch,
			-- Attachment fields: only from TurboPurchaseOrders (legacy POs don't have attachments)
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.attachment FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE NULL
			END AS attachment,
			CASE
				WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = fe.po) = 1
				THEN (SELECT tp.attachment_hash FROM turbo_pos tp WHERE tp.number = fe.po)
				ELSE NULL
			END AS attachment_hash
		FROM first_expense_per_po fe
		JOIN sum_total_per_po st ON st.po = fe.po;

		-- Also include Turbo-only POs (those not matching any expense PO number)
		-- Convert legacy UIDs to PocketBase UIDs
		INSERT INTO purchase_orders (id, number, approver, date, end_date, vendor, uid, total, approval_total, payment_type, job, description, division, type, frequency, status, approved, second_approval, second_approver, priority_second_approver, closed, closer, cancelled, canceller, rejected, rejector, rejection_reason, category, kind, parent_po, branch, attachment, attachment_hash)
		SELECT
			tp.id,
			tp.number,
			(SELECT p.pocketbase_uid FROM profiles p WHERE p.id = tp.approver) AS approver,
			tp.date,
			tp.end_date,
			tp.vendor,
			(SELECT p.pocketbase_uid FROM profiles p WHERE p.id = tp.uid) AS uid,
			tp.total,
			tp.approval_total,
			tp.payment_type,
			tp.job,
			tp.description,
			tp.division,
			tp.type,
			tp.frequency,
			tp.status,
			tp.approved,
			tp.second_approval,
			(SELECT p.pocketbase_uid FROM profiles p WHERE p.id = tp.second_approver) AS second_approver,
			(SELECT p.pocketbase_uid FROM profiles p WHERE p.id = tp.priority_second_approver) AS priority_second_approver,
			tp.closed,
			(SELECT p.pocketbase_uid FROM profiles p WHERE p.id = tp.closer) AS closer,
			tp.cancelled,
			(SELECT p.pocketbase_uid FROM profiles p WHERE p.id = tp.canceller) AS canceller,
			tp.rejected,
			(SELECT p.pocketbase_uid FROM profiles p WHERE p.id = tp.rejector) AS rejector,
			tp.rejection_reason,
			tp.category,
			COALESCE(
				NULLIF(tp.kind, ''),
				CASE WHEN tp.job IS NOT NULL AND TRIM(CAST(tp.job AS VARCHAR)) != '' THEN '{{PROJECT_KIND_ID}}' ELSE '{{CAPITAL_KIND_ID}}' END
			) AS kind,
			tp.parent_po,
			tp.branch,
			tp.attachment,
			tp.attachment_hash
		FROM turbo_pos tp
		WHERE NOT EXISTS (SELECT 1 FROM purchase_orders po WHERE po.id = tp.id);

		-- Update expenses to use the correct PO ID (hybrid resolution)
		ALTER TABLE expenses ADD COLUMN purchase_order_id string;
		UPDATE expenses
		SET purchase_order_id = CASE
			WHEN (SELECT COUNT(*) FROM turbo_pos tp WHERE tp.number = expenses.po) = 1
			THEN (SELECT tp.id FROM turbo_pos tp WHERE tp.number = expenses.po)
			ELSE make_pocketbase_id(expenses.po, 15)
		END
		WHERE po IS NOT NULL AND po != '';
`
	} else {
		query += `
		-- Create purchase_orders (no TurboPurchaseOrders - generate all IDs from PO numbers)
		-- Legacy-derived POs have NULL for fields that only exist in TurboPurchaseOrders
		-- The import handles defaults for type, status, etc.
		CREATE TABLE purchase_orders AS
		SELECT
			make_pocketbase_id(fe.po, 15) AS id,
			fe.po AS number,
			fe.pocketbase_approver_uid AS approver,
			CAST(fe.date AS VARCHAR) AS date,
			NULL AS end_date,
			fe.vendor_id AS vendor,
			fe.pocketbase_uid AS uid,
			st.total AS total,
			NULL AS approval_total,
			fe.paymentType AS payment_type,
			fe.pocketbase_jobid AS job,
			NULL AS description,
			fe.division_id AS division,
			NULL AS type,
			NULL AS frequency,
			NULL AS status,
			NULL AS approved,
			NULL AS second_approval,
			NULL AS second_approver,
			NULL AS priority_second_approver,
			NULL AS closed,
			NULL AS closer,
			NULL AS cancelled,
			NULL AS canceller,
			NULL AS rejected,
			NULL AS rejector,
			NULL AS rejection_reason,
			NULL AS category,
			COALESCE(
				NULLIF(fe.kind, ''),
				CASE WHEN fe.pocketbase_jobid IS NOT NULL AND TRIM(CAST(fe.pocketbase_jobid AS VARCHAR)) != '' THEN '{{PROJECT_KIND_ID}}' ELSE '{{CAPITAL_KIND_ID}}' END
			) AS kind,
			NULL AS parent_po,
			NULL AS branch,
			NULL AS attachment,
			NULL AS attachment_hash
		FROM first_expense_per_po fe
		JOIN sum_total_per_po st ON st.po = fe.po;

		-- Update the expenses table to use the purchase_order id derived deterministically from PO
		ALTER TABLE expenses ADD COLUMN purchase_order_id string;
		UPDATE expenses
		SET purchase_order_id = make_pocketbase_id(po, 15)
		WHERE po IS NOT NULL AND po != '';
`
	}

	query += `
		-- Write the parquet used by the importer
		COPY purchase_orders TO 'parquet/purchase_orders.parquet' (FORMAT PARQUET);

		-- Persist updated expenses parquet
		COPY expenses TO 'parquet/Expenses.parquet' (FORMAT PARQUET);
`

	return query
}

// validateTurboPOUserMappings ensures TurboPurchaseOrders legacy UID fields
// are resolvable to PocketBase UIDs before we write parquet outputs.
func validateTurboPOUserMappings(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT
			po.id,
			po.number,
			COALESCE(po.uid, '') AS uid,
			COALESCE(po.approver, '') AS approver,
			COALESCE(po.second_approver, '') AS second_approver,
			COALESCE(po.priority_second_approver, '') AS priority_second_approver,
			COALESCE(po.closer, '') AS closer,
			COALESCE(po.canceller, '') AS canceller,
			COALESCE(po.rejector, '') AS rejector,
			COALESCE(tp.uid, '') AS legacy_uid,
			COALESCE(tp.approver, '') AS legacy_approver,
			COALESCE(tp.second_approver, '') AS legacy_second_approver,
			COALESCE(tp.priority_second_approver, '') AS legacy_priority_second_approver,
			COALESCE(tp.closer, '') AS legacy_closer,
			COALESCE(tp.canceller, '') AS legacy_canceller,
			COALESCE(tp.rejector, '') AS legacy_rejector
		FROM purchase_orders po
		JOIN turbo_pos tp ON po.id = tp.id
	`)
	if err != nil {
		return fmt.Errorf("query TurboPurchaseOrders UID mappings: %w", err)
	}
	defer rows.Close()

	var missing []string
	for rows.Next() {
		var (
			id, number                                                                    string
			uid, approver, secondApprover, prioritySecondApprover, closer, canceller      string
			rejector                                                                      string
			legacyUid, legacyApprover, legacySecondApprover, legacyPrioritySecondApprover string
			legacyCloser, legacyCanceller, legacyRejector                                 string
		)

		if err := rows.Scan(
			&id,
			&number,
			&uid,
			&approver,
			&secondApprover,
			&prioritySecondApprover,
			&closer,
			&canceller,
			&rejector,
			&legacyUid,
			&legacyApprover,
			&legacySecondApprover,
			&legacyPrioritySecondApprover,
			&legacyCloser,
			&legacyCanceller,
			&legacyRejector,
		); err != nil {
			return fmt.Errorf("scan TurboPurchaseOrders UID mappings: %w", err)
		}

		var missingFields []string
		if legacyUid != "" && uid == "" {
			missingFields = append(missingFields, "uid")
		}
		if legacyApprover != "" && approver == "" {
			missingFields = append(missingFields, "approver")
		}
		if legacySecondApprover != "" && secondApprover == "" {
			missingFields = append(missingFields, "second_approver")
		}
		if legacyPrioritySecondApprover != "" && prioritySecondApprover == "" {
			missingFields = append(missingFields, "priority_second_approver")
		}
		if legacyCloser != "" && closer == "" {
			missingFields = append(missingFields, "closer")
		}
		if legacyCanceller != "" && canceller == "" {
			missingFields = append(missingFields, "canceller")
		}
		if legacyRejector != "" && rejector == "" {
			missingFields = append(missingFields, "rejector")
		}

		if len(missingFields) > 0 {
			label := id
			if number != "" {
				label = fmt.Sprintf("%s (%s)", id, number)
			}
			missing = append(missing, fmt.Sprintf("%s: %s", label, strings.Join(missingFields, ", ")))
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate TurboPurchaseOrders UID mappings: %w", err)
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing PocketBase UID mappings for %d Turbo POs: %s", len(missing), strings.Join(missing, "; "))
	}

	return nil
}
