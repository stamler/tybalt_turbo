# Deployment Phases for Turbo Data Authority Migration

This document describes the phased rollout strategy for migrating data authority from Tybalt (legacy Firebase/Firestore) to Turbo (PocketBase/SQLite).

## Overview

As Turbo becomes increasingly authoritative for different data types, we need to prevent the import process from overwriting authoritative Turbo data while still allowing fresh imports of non-authoritative data from Tybalt.

The solution uses **inclusive phase flags** in `tool.go`:

- `--jobs` - Import jobs, clients, contacts, categories, job_time_allocations
- `--expenses` - Import vendors, purchase_orders, expenses
- `--time` - Import time_sheets, time_entries, time_amendments
- `--users` - Import users, profiles, admin_profiles, user_claims, mileage_reset_dates

**Running `--import` with no phase flags is a safe no-op** to prevent accidental data overwrites.

## Import Behavior: Full Replace

When importing with a phase flag, **all existing records in that phase's tables are deleted first**, then replaced with fresh data from the Parquet files. This ensures:

1. **Clean slate:** No stale or orphaned records remain from previous imports
2. **Consistent state:** The database exactly matches what's in Tybalt at export time
3. **Safe for testing:** Test data in non-live phases is automatically cleared

The `_imported` flag is set to `true` for all imported records. This flag is used by the writeback mechanism to identify records that have been modified in Turbo (where `_imported = false` or `0`).

## Data Flow

```text
Tybalt (Firestore) → sync.ts → MySQL → tool.go --export → Parquet files
                                                              ↓
Turbo (SQLite) ← tool.go --import --<phase flags> ← Parquet files
       ↓
Writeback API → turboSync.ts → Tybalt (Firestore) → sync.ts → MySQL
```

When a phase becomes authoritative in Turbo:

1. That phase's data flows Turbo → Tybalt (via writeback)
2. The writeback continues to MySQL → Parquet
3. But `--import` omits that phase flag, so it doesn't flow back into Turbo

## Phase Definitions

### Phase 1: Jobs (`--jobs`)

**Tables:** `clients`, `client_contacts`, `jobs`, `categories`, `job_time_allocations`

**Dependencies:** None (UID resolution for `clients.business_development_lead` is done via DuckDB join at export time in `augment_clients_export.go`)

**Writeback:** Already implemented in `jobs_writeback.go` and `turboSync.ts`

### Phase 2: Expenses (`--expenses`)

**Tables:** `vendors`, `purchase_orders`, `expenses`

**Dependencies:** Requires Phase 1 (jobs) and Phase 4 (profiles) to be imported first, as expenses reference jobs and profiles.

**Writeback:** Not yet implemented

### Phase 3: Time (`--time`)

**Tables:** `time_sheets`, `time_entries`, `time_amendments`

**Dependencies:** Requires Phase 1 (jobs) and Phase 4 (profiles) to be imported first, as time entries reference jobs and profiles.

**Writeback:** Not yet implemented

### Phase 4: Users (`--users`)

**Tables:** `users`, `profiles`, `admin_profiles`, `_externalAuths`, `user_claims`, `mileage_reset_dates`

**Dependencies:** None (foundational, but used by all other phases)

**Writeback:** Not yet implemented. Should remain import-only for longest as profiles are referenced by nearly everything else.

## Example Commands

```bash
# Full import (all phases) - use during development/testing
./tool --export --import --db ../app/pb_data/data.db --jobs --expenses --time --users

# After Phase 1 (jobs) goes live in Turbo
./tool --export --import --db ../app/pb_data/data.db --expenses --time --users

# After Phase 2 (expenses) goes live in Turbo
./tool --export --import --db ../app/pb_data/data.db --time --users

# After Phase 3 (time) goes live in Turbo
./tool --export --import --db ../app/pb_data/data.db --users

# Import with no phase flags = no-op (safe default)
./tool --import --db ../app/pb_data/data.db
# Output: "No phases specified. Use --jobs, --expenses, --time, --users to select what to import."
```

## Deployment Procedures

### Phase 1 Deployment (Jobs)

1. **Disable jobs editing in Turbo** (if currently enabled for testing)
2. **Disable jobs editing in Tybalt** (set `Config/Enable.jobs = false`)
3. **Perform full export/import** with tool.go to sync latest Tybalt data to Turbo:

   ```bash
   ./tool --export --import --db ../app/pb_data/data.db --jobs --expenses --time --users
   ```

4. **Enable jobs editing in Turbo production**
5. **Future `--import` runs:** `--expenses --time --users` (omit `--jobs`)
6. Jobs writeback continues flowing Turbo → Tybalt → MySQL → Parquet (but not back into Turbo)

### Phase 2 Deployment (Expenses/POs)

1. **Build expense/PO writebacks in Turbo**
2. **Test thoroughly**
3. **Disable expenses/PO editing in Turbo** (end of testing)
4. **Disable expenses/PO editing in Tybalt**
5. **Perform full export/import** with tool.go to sync latest Tybalt data to Turbo:

   ```bash
   ./tool --export --import --db ../app/pb_data/data.db --expenses --time --users
   ```

6. **Enable expenses/PO editing in Turbo production**
7. **Future `--import` runs:** `--time --users`

### Phase 3 Deployment (Time)

1. **Build time entry/amendment writebacks in Turbo**
2. **Test thoroughly**
3. **Disable time editing in Turbo** (end of testing)
4. **Disable time editing in Tybalt**
5. **Perform full export/import** with tool.go to sync latest Tybalt data to Turbo:

   ```bash
   ./tool --export --import --db ../app/pb_data/data.db --time --users
   ```

6. **Enable time editing in Turbo production**
7. **Future `--import` runs:** `--users`

### Phase 4 Deployment (Users)

1. **Build user/profile writebacks in Turbo** (if needed)
2. **Test thoroughly**
3. **Disable user/profile editing in Turbo** (end of testing)
4. **Disable user/profile editing in Tybalt**
5. **Perform full export/import** with tool.go to sync latest Tybalt data to Turbo:

   ```bash
   ./tool --export --import --db ../app/pb_data/data.db --users
   ```

6. **Enable user/profile editing in Turbo production**
7. **Future `--import` runs:** No phase flags needed (all data is authoritative in Turbo)

## Key Principles

1. **Disable first, then enable:** Never have two systems enabled for editing the same data simultaneously.

2. **Inclusive flags (opt-in):** Running `--import` with no phase flags is a no-op. You must explicitly specify which phases to import.

3. **Final sync before cutover:** Always perform a full export/import immediately before enabling a phase in Turbo to ensure data is current.

4. **Full replace per phase:** Each phase flag triggers a DELETE ALL → INSERT for those tables. This keeps the database clean and consistent.

## Technical Notes

### The `_imported` Flag

Records imported from Tybalt have `_imported = true`. When a record is created or modified directly in Turbo, the `_imported` flag is set to `false`.

The writeback mechanism (e.g., `jobs_writeback.go`) uses `WHERE _imported = 0` to identify records that have been modified in Turbo and need to be synced back to Tybalt. This ensures only Turbo-native changes flow back, preventing infinite sync loops.

### Phase Flag Implementation

The phase flags wrap import sections in `tool.go`:

```go
if *jobsFlag {
    // DELETE FROM jobs, job_time_allocations, client_contacts, categories, clients
    // Load clients, contacts, jobs, categories, job_time_allocations
}

if *usersFlag {
    // DELETE FROM user_claims, mileage_reset_dates, _externalAuths, admin_profiles, profiles, users
    // Load users, admin_profiles, profiles, _externalAuths, user_claims, mileage_reset_dates
}

if *timeFlag {
    // DELETE FROM time_entries, time_amendments, time_sheets
    // Load time_sheets, time_entries, time_amendments
}

if *expensesFlag {
    // DELETE FROM expenses, purchase_orders, vendors
    // Load vendors, purchase_orders, expenses
}
```

### Interaction with `--init`

The `--init` flag copies the test database schema and clears data tables. It is orthogonal to `--import`:

- Use `--init` to set up a fresh database with the correct schema
- Use `--import --<phases>` to populate data

They can be combined: `./tool --init --import --jobs --expenses --time --users`
