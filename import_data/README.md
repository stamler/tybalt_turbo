# Import Tool Documentation

## Overview

The import tool (`tool.go`) provides **unidirectional synchronization** from MySQL to SQLite/PocketBase via Parquet files. It handles data export, import, and deletion cleanup while preserving local SQLite data.

## CLI Options

```bash
./tool [OPTIONS]
```

| Flag | Description |
|------|-------------|
| `--export` | Export MySQL data to Parquet files |
| `--import` | Import Parquet files into SQLite/PocketBase |
| `--cleanup` | Clean up deleted MySQL records from SQLite |
| `--attachments` | Migrate attachments from GCS to S3 |
| `--db PATH` | Path to target database (default: `../app/test_pb_data/data.db`) |

Options can be combined: `./tool --import --cleanup --db /path/to/custom.db`

## Core Concepts

### The `_imported` Field

- **Purpose**: Distinguishes between local SQLite data and MySQL-imported data
- **Values**: `true` (from MySQL) / `false` (local, default)
- **Scope**: Added to 14 collections that sync with MySQL
- **Safety**: Cleanup only affects records with `_imported = true`

### Idempotency

- **Export**: Deterministic hash-based IDs, fixed timestamps, ordered results
- **Import**: Uses `INSERT OR REPLACE` (upserts) to handle duplicates
- **Cleanup**: Removes orphaned imported records safely

## Operations

### Export (`--export`)

**Purpose**: Extract MySQL data to Parquet files
**Output**: `./parquet/*.parquet` files  
**Behavior**:

- Generates deterministic IDs using MD5 hashing
- Uses fixed timestamps for consistency
- Orders results for reproducible output
- **Fully idempotent**: Multiple runs produce identical files

### Import (`--import`)

**Purpose**: Load Parquet data into SQLite/PocketBase
**Target**: Configurable via `--db` flag (default: `../app/test_pb_data/data.db`)
**Collections Imported**:

- Core: clients, client_contacts, jobs, categories, vendors, purchase_orders, expenses
- Users: users, profiles, admin_profiles, user_claims, _externalAuths
- Time: time_sheets, time_entries, time_amendments, mileage_reset_dates

**Behavior**:

- Sets `_imported = true` on all imported records
- Uses upserts to handle existing records
- Automatically runs cleanup if `--cleanup` flag included

### Cleanup (`--cleanup`)

**Purpose**: Remove imported records that no longer exist in MySQL export
**Safety**: Only deletes records where `_imported = true`
**Logic**:

1. Read current IDs from each Parquet file
2. Find SQLite records with `_imported = true` but ID not in current export
3. Delete orphaned records
4. Special handling for `user_claims` composite key (uid+cid)

**Protection**: Local SQLite data (`_imported = false`) is never touched

### Attachments (`--attachments`)

**Purpose**: Migrate expense attachments from Google Cloud Storage to S3
**Scope**: Only processes attachment files referenced in Expenses.parquet

## Data Flow

```fixed
MySQL → [--export] → Parquet Files → [--import] → SQLite/PocketBase
                                   → [--cleanup] → Remove Deletions
```

### po_approver_props Precedence

- `--export` includes `TurboPoApproverProps` as `parquet/PoApproverProps.parquet`.
- `--import --users` resolves po approver props per user:
  - Turbo row is authoritative when present.
  - Synthesis is used only as fallback for users without Turbo rows.
- Import recreates/links `user_claims` (`po_approver`) and writes `po_approver_props` against that claim.
- Turbo rows are strict-validated (identity, limits, timestamps, and valid JSON array in `divisions`).

## Usage Examples

```bash
# Full sync: export + import + cleanup
./tool --export && ./tool --import --cleanup

# Import existing Parquet files with cleanup
./tool --import --cleanup

# Only check for deletions
./tool --cleanup

# Export only (for testing idempotency)
./tool --export

# Migrate attachments
./tool --attachments

# Use custom database path
./tool --import --cleanup --db /path/to/production.db

# Export with custom sqlite database source
./tool --export --db /path/to/custom.db
```

## Technical Details

### Database Connections

- **DuckDB**: Used for Parquet file processing (in-memory)
- **SQLite**: Target database (configurable via `--db` flag)
- **MySQL**: Source database (via export process)

### ID Field Mapping

Different collections use different ID field names in Parquet:

- `id`: clients, client_contacts, categories, vendors, purchase_orders
- `pocketbase_id`: jobs, expenses, time_sheets, time_entries, time_amendments, mileage_reset_dates, profiles
- `pocketbase_uid`: admin_profiles
- `uid`+`cid`: user_claims (composite key)

### Error Handling

- Graceful handling of missing Parquet files
- Continues processing other collections if one fails
- Detailed logging of success/failure counts
- Non-destructive: won't delete everything if export appears empty

### Performance

- Processes collections sequentially
- Batch operations using DuckDB for efficiency
- Memory-efficient Parquet reading
- Uses prepared statements for SQLite inserts

## Migration Requirements

Before using the tool, ensure the `_imported` field migration has been applied:

```bash
# In the app directory
go run cmd/main.go migrate up
```

This adds the `_imported` boolean field (default `false`) to all relevant collections.
