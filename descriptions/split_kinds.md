# Split `standard` expenditure kind into `capital` + `project`

## Summary

Replace the single `expenditure_kinds.name='standard'` kind (currently used for both "capital (no job)" and "project (with job)") with two explicit kinds:

- `capital` (no job; `allow_job=false`; `second_approval_threshold=500`)
- `project` (requires job; `allow_job=true`; `second_approval_threshold=5000`)

Migrate existing data so any existing PO/expense that previously used `standard` is reclassified as `capital` vs `project` based on whether a job is present. Update backend, UI, import pipeline, tests, and the test fixture DB.

---

## Data / Migration

### Migration strategy (chosen)

- Rename the existing `standard` kind row to `capital` (keep the same PocketBase record ID).
- Insert a new `project` kind row.

### New migration (PocketBase Go migration in `app/migrations/`)

Create a new migration file (new timestamp) that:

#### Up migration

1. Resolve the current "legacy standard / capital" record:
   - `SELECT id, second_approval_threshold FROM expenditure_kinds WHERE name='standard' LIMIT 1`
   - If not found (edge case: already migrated), then `SELECT id FROM expenditure_kinds WHERE name='capital' LIMIT 1`
   - Treat the resolved ID as `capitalKindID`.
2. Update that record in-place to be `capital`:
   - `name='capital'`
   - `en_ui_label='Capital'` (and optionally update `description`)
   - `allow_job=0`
   - keep `second_approval_threshold` as-is **unless** it's `<=0`, then set it to `500`
3. Insert `project` kind record (if missing):
   - `name='project'`
   - `en_ui_label='Project'`
   - `allow_job=1`
   - `second_approval_threshold=5000`
4. Backfill `purchase_orders.kind`:
   - For rows whose `kind` equals `capitalKindID` (the old "standard" ID) **and** `TRIM(job) != ''`, set `kind=projectKindID`.
   - Additionally, for safety, handle legacy blanks: if any `purchase_orders.kind` is blank/NULL, set:
     - `projectKindID` when `TRIM(job) != ''`
     - else `capitalKindID`
5. Backfill `expenses.kind` to eliminate blanks and remove legacy ambiguity:
   - If `TRIM(purchase_order) != ''`, set `expenses.kind = (SELECT po.kind FROM purchase_orders po WHERE po.id = expenses.purchase_order)`.
   - Else set based on job presence:
     - `projectKindID` when `TRIM(job) != ''`
     - else `capitalKindID`
   - Finally, any remaining blank/NULL `expenses.kind` → set using the same job-based rule.

#### Down migration

- Set any `purchase_orders.kind == projectKindID` back to `capitalKindID`.
- Same for `expenses`.
- Rename `capital` back to `standard`, set `allow_job=1`, `en_ui_label='Capital/Project'`.
- Delete the `project` kind record.

---

## Backend changes (Go / PocketBase)

### `app/utilities/expenditure_kinds.go`

Refactor to remove `standard` as a semantic kind and add:

- Kind name constants:
  - `ExpenditureKindNameCapital = "capital"`
  - `ExpenditureKindNameProject = "project"`
  - Keep `ExpenditureKindNameStandard = "standard"` only as a *legacy string* where needed for import/back-compat (not in "expected kinds").
- Default kind helpers:
  - `DefaultCapitalExpenditureKindID()`
  - `DefaultProjectExpenditureKindID()`
  - `DefaultExpenditureKindIDForJob(hasJob bool)` → project when `true`, else capital.
  - Replace `NormalizeExpenditureKindID(kindID string)` with something job-aware, e.g.:
    - `NormalizeExpenditureKindID(kindID string, hasJob bool)`:
      - blank/unknown → default by job
      - if kind resolves to `capital` but `hasJob=true` → return project default (prevents invalid capital+job combos)
- Update `nameToPOApproverColumn` map:
  - Add `"capital": "max_amount"` and `"project": "project_max"` entries.
  - Remove the separate standard-specific branching.
- Update `expectedExpenditureKindNames`:
  - Replace `standard` with `capital` and `project`.
- Update approval-limit mapping:
  - `capital` → `max_amount`
  - `project` → `project_max`
  - other kinds unchanged
  - `ResolvePOApproverLimitColumn(kindID, hasJob)` should no longer special-case "standard"; it should resolve kind-name and fall back by `hasJob`.

### Purchase order job/kind validation

Update `app/hooks/purchase_orders.go`:

- Keep existing `allow_job` behavior (capital disallows job).
- Add "project requires job" validation:
  - If kind name is `project` and `TRIM(job) == ''`, return a 400 with a specific error code/message (e.g. `job_required_for_kind`).

### Routes / policy evaluation

Update all call sites that previously assumed "blank => standard" in `app/routes/purchase_orders.go`:

- **Line ~1134** (approver request struct default): Replace `DefaultExpenditureKindID()` with `DefaultCapitalExpenditureKindID()` (the endpoint also receives `has_job` and can resolve from there).
- **Line ~1200** (`NormalizeExpenditureKindID` before policy lookup): Update call to pass `hasJob`.
- **Lines ~225, ~696** (approve/reject actions): `NormalizeExpenditureKindID(po.GetString("kind"))` → pass `hasJob` derived from `po.GetString("job")`.
- In `app/utilities/po_approvers.go`:
  - Compute `hasJob := TRIM(job) != ''`
  - Normalize kind with `hasJob`
  - Keep the existing `has_job` query param in `/api/purchase_orders/approvers` contract, but after this refactor it only matters for legacy/blank/unknown kinds.

### Expenses defaulting and legacy kind repair

**`app/hooks/expenses.go`**:

- **Lines ~40-43** (`cleanExpense`): Replace `DefaultExpenditureKindID()` with job-aware defaulting:
  - no PO → set kind to `project` if job present else `capital`
- **Line ~472** (PO-linked expense kind inheritance): `NormalizeExpenditureKindID(poRecord.GetString("kind"))` → update call to pass `hasJob` (derived from the PO's job field, since expense inherits from PO).

**`app/hooks/validate_expenses.go`**:

- **Lines ~169-179** (legacy-record blank-kind defaulting path): When an existing expense record has blank kind and is being updated, the code defaults to `DefaultExpenditureKindID()`. Replace with job-aware defaulting using `DefaultExpenditureKindIDForJob(hasJob)`.

### SQL used by runtime endpoints

Update `app/routes/po_visibility_base.sql`:

- Replace the `standard` + `po.job != ''` CASE branches with explicit `capital`/`project` branches.
- Remove `ek_standard` join.
- Keep a robust fallback for missing/unknown kind rows:
  - "effective kind name" defaults to `project` when job present else `capital`
  - threshold and resolved_limit computed from that effective kind name

### DB-stored view query updates

Because old migrations won't re-run, add a *new* migration to update the stored query for `pending_items_for_qualified_po_second_approvers` (currently v3):

- Create v4 query that:
  - Uses `capital`/`project` names
  - Uses project/capital thresholds
  - Resolves limits by kind (project → `project_max`, capital → `max_amount`)
  - Has the same safe fallback for missing kind rows (job-based effective kind)

---

## Import pipeline changes (`import_data/`)

### Kind ID resolution (`import_data/extract/kind.go`)

Refactor from "standard-only" to:

- Resolve IDs for `capital` and `project` by name from the target SQLite DB.
- Back-compat: if `capital` isn't present but `standard` is, treat that ID as `capital`.

Expose helpers like:

- `GetExpenditureKindIDByName(dbPath, name)`
- `GetCapitalAndProjectKindIDs(dbPath)` (with fallback to legacy standard)

### Import normalization (`import_data/tool.go`)

Replace the single `defaultExpenditureKindID` with:

- `defaultCapitalKindID`
- `defaultProjectKindID`

Update `normalizeExpenditureKindID(...)` to accept job presence:

- Blank kind → default by job presence
- "Legacy standard/capital ID" + job present → force to project (prevents importing capital+job)
- Project + no job → force to capital (optional but aligns with "defaulting" requirement)

Apply this normalization to both `purchase_orders.kind` and `expenses.kind` imports.

### Export / parquet generation

Update:

- `import_data/extract/export_to_parquet.go` (TurboPurchaseOrders export):
  - **Lines ~289-291**: Replace `StandardExpenditureKindID()` default with job-aware capital/project selection using a SQL CASE expression on the job column.
  - **Line ~152**: Update log message from "defaulting export kind to standard" to reflect new behavior.
- `import_data/extract/expenses_to_pos.go`:
  - **Line ~57**: Replace the single `{{STANDARD_KIND_ID}}` template substitution with two: `{{CAPITAL_KIND_ID}}` and `{{PROJECT_KIND_ID}}`.
  - **Lines ~314-315, ~375, ~428** (inside `buildPurchaseOrderQuery`): All three `COALESCE(NULLIF(...kind..., ''), '{{STANDARD_KIND_ID}}')` expressions must become job-aware CASE expressions selecting between capital and project IDs based on job presence.
  - **Lines ~83-89** (`prepareTurboPOsTable`): Replace the blanket `UPDATE turbo_pos SET kind = standardID WHERE kind IS NULL` with a job-aware update: `CASE WHEN job IS NOT NULL AND TRIM(job) != '' THEN projectID ELSE capitalID END`.

### Expense kind backfill (`import_data/extract/augment_expenses.go`)

- **Lines ~23-30** (`ensureKindColumnWithDefault`): This function backfills blank kinds on any table with `StandardExpenditureKindID()`. Must become job-aware:
  - Accept both `capitalKindID` and `projectKindID` parameters.
  - Use `CASE WHEN TRIM(CAST(job AS VARCHAR)) != '' THEN projectKindID ELSE capitalKindID END` instead of a flat default.
  - If the table doesn't have a `job` column, fall back to `capitalKindID`.

---

## Frontend changes (SvelteKit)

### Default PO kind selection (`ui/src/lib/components/PurchaseOrdersEditor.svelte`)

Replace "default to standard" with:

- If `item.kind` is empty:
  - If `item.job` is non-empty → default to `project`
  - Else → default to `capital`
Implementation detail:
- Find the kind IDs from `$expenditureKindsStore.items` by `name === "capital"` / `"project"`.

**Two locations** must be updated:

- **Lines ~343-344** (kind defaulting on component load): `kinds.find((r) => r.name === "standard")` → job-aware lookup.
- **Lines ~481-482** (kind defaulting on submit): `$expenditureKindsStore.items.find((k) => k.name === "standard")` → same job-aware lookup.

### Expenses kind display (`ui/src/lib/components/ExpensesEditor.svelte`)

Stop hardcoding "standard" label when no PO:

- **Lines ~46-55**: Replace `$expenditureKindsStore.items.find((k) => k.name === "standard")?.en_ui_label` with a lookup by `item.kind` ID.
- If `item.kind` is blank (shouldn't be after fixture update), fall back to job-based label selection (project vs capital) for display only.

### Admin profile labels

Update copy only (no schema change):

- `ui/src/lib/components/AdminProfilesEditor.svelte`:
  - "Standard (No Job) Max" → "Capital Max"
  - "Standard (With Job) Project Max" → "Project Max"
- `ui/src/routes/admin_profiles/[id]/details/+page.svelte`:
  - "Standard (No Job)" → "Capital"
  - "Standard (With Job)" → "Project"

---

## Test fixture DB update

### `app/test_pb_data/data.db`

After implementing migrations + code, update the fixture DB by applying migrations to it (so tests don't rely on runtime migration execution):

- Run the backend in a way that applies migrations against `--dir="./test_pb_data"` (e.g., `cd app && go run main.go serve --dir="./test_pb_data"` once) and then shut it down.
- Verify in SQLite:
  - `expenditure_kinds` contains `capital` and `project` with correct thresholds and `allow_job` settings.
  - `purchase_orders` previously "standard" with jobs are now `project`.
  - `expenses.kind` is no longer blank and matches PO or job-based defaults.

---

## Tests to update (Go)

### Threshold helper

Update `app/internal/testutils/testutils.go`:

- `GetApprovalTiers` should query the capital kind threshold (`name='capital'`) for tier1.
- If any tests need "project tier1", add a helper that accepts a kind name.

### Test helper default kind

Update `app/hooks/hooks_test.go`:

- **Line ~23** (`buildRecordFromMap`): `DefaultExpenditureKindID()` → replace with `DefaultCapitalExpenditureKindID()` (or make the helper accept a kind parameter, since some test cases will need `project`).

### Scenario rewrites

Update:

- `app/purchase_orders_test.go`:
  - **Lines ~66-78**: Replace `name = 'standard'` lookup with `name = 'capital'` to get `capitalKindID`.
  - Add a second lookup for `name = 'project'` to get `projectKindID` for project-related test cases.
  - Rename `standardKindID` variable to `capitalKindID` throughout.
  - **Lines ~272, ~1292, ~1332, ~1639, ~1725**: Update all test bodies that pass `standardKindID` to use `capitalKindID` or `projectKindID` as appropriate for each scenario.
- `app/purchase_orders_approvers_test.go`:
  - **Line ~250**: Rename/rework the scenario currently asserting "standard kind with job uses project_max" to assert "project kind uses project_max". Use `projectKindID` instead of `DefaultExpenditureKindID()` + `has_job=true`.
- `app/expenses_test.go`:
  - Update expectations:
    - No-PO expense without job → capital
    - No-PO expense with job → project
  - **Lines ~182-204**: Update test name from "no-po create without kind is set to standard kind" to reflect capital/project behavior. Add a new test case for "no-PO expense with job → project kind".
- Any remaining "standard" references in tests should become:
  - `capital` or `project` depending on the case being tested.

### New test cases to add

- **"project kind without job is rejected"**: Test that creating a PO with `kind=projectKindID` and blank `job` returns 400 with `job_required_for_kind` error code (validates the new `purchase_orders.go` hook logic).
- **"no-PO expense with job defaults to project kind"**: Test in `expenses_test.go` that a no-PO expense with a job field set gets `kind=projectKindID`.

---

## Documentation updates

### `descriptions/authority_matrix.md`

- Replace references to "standard" kind with `capital` and `project` (e.g., "standard (Capital Expense) uses max_amount" → "capital uses max_amount").

### `descriptions/purchase_orders.md`

- **Lines ~33-34, ~507-508**: Replace "standard with no job → max_amount, standard with job → project_max" with "capital → max_amount, project → project_max".

### `CLAUDE.md`

- Update the "Key Business Concepts > Expenditure Kinds" section: replace "The `standard` kind uses `max_amount` (no job) or `project_max` (with job)" with the new `capital`/`project` semantics.

---

## Acceptance criteria / Verification

### Core behavior

- Expenditure kinds present and validated at startup: `capital`, `project`, and existing non-standard kinds.
- New PO creation:
  - selecting `project` without job fails with a clear error
  - selecting `capital` with a job is prevented by `allow_job=false` and/or migration/import normalization
- Approver selection and second-approval behavior:
  - `capital` uses `max_amount` + threshold 500
  - `project` uses `project_max` + threshold 5000
- Import pipeline:
  - Defaults to `capital` when no job; `project` when job present
  - Legacy "standard/capital ID" + job present is normalized to `project`

### Regression suite

- `go test ./...` under `app/` passes.
- UI builds and PO editor defaults kind correctly based on job presence.

---

## Assumptions and defaults

- `capital.second_approval_threshold = 500` and `project.second_approval_threshold = 5000`.
- `capital.allow_job=false`, `project.allow_job=true`, and backend enforces "project requires job".
- The existing `standard` PocketBase record ID becomes the `capital` ID (no new ID for capital).

---

## Complete file inventory

Every file that needs changes, grouped by area:

| #  | File                                                                          | What changes                                                                                                                                                                                                                                   |
|----|-------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| 1  | `app/utilities/expenditure_kinds.go`                                          | Remove `standard` from expected kinds; add `capital`/`project` constants; replace `DefaultExpenditureKindID` with job-aware helpers; update `ResolvePOApproverLimitColumn`; update `nameToPOApproverColumn` and `expectedExpenditureKindNames` |
| 2  | `app/hooks/purchase_orders.go`                                                | Add "project requires job" validation                                                                                                                                                                                                          |
| 3  | `app/hooks/expenses.go` (~L40-43)                                             | `DefaultExpenditureKindID()` → job-aware capital/project                                                                                                                                                                                       |
| 4  | `app/hooks/expenses.go` (~L472)                                               | `NormalizeExpenditureKindID` signature update (add `hasJob`)                                                                                                                                                                                   |
| 5  | `app/hooks/validate_expenses.go` (~L169-179)                                  | Legacy blank kind defaulting → job-aware                                                                                                                                                                                                       |
| 6  | `app/routes/purchase_orders.go` (~L225, L696, L1134, L1200)                   | Update `NormalizeExpenditureKindID` calls + approver default kind                                                                                                                                                                              |
| 7  | `app/utilities/po_approvers.go`                                               | Indirect — uses `ResolvePOApproverLimitColumn` (changes flow through from expenditure_kinds.go)                                                                                                                                                |
| 8  | `app/routes/po_visibility_base.sql`                                           | Replace `'standard'` CASE branches with `capital`/`project`; remove `ek_standard` join                                                                                                                                                         |
| 9  | `app/migrations/` (new data migration)                                        | Rename `standard`→`capital`; insert `project`; backfill PO/expense kinds                                                                                                                                                                       |
| 10 | `app/migrations/` (new v4 view migration)                                     | Update `pending_items_for_qualified_po_second_approvers` stored query                                                                                                                                                                          |
| 11 | `import_data/extract/kind.go`                                                 | Refactor to resolve `capital`/`project` IDs with `standard` fallback                                                                                                                                                                           |
| 12 | `import_data/tool.go` (~L31-88)                                               | Replace `defaultExpenditureKindID` with two vars; update `normalizeExpenditureKindID`                                                                                                                                                          |
| 13 | `import_data/extract/export_to_parquet.go` (~L152, L289-291)                  | Replace `StandardExpenditureKindID()` with job-aware selection                                                                                                                                                                                 |
| 14 | `import_data/extract/expenses_to_pos.go` (~L57, L83-89, L314-315, L375, L428) | Replace `{{STANDARD_KIND_ID}}` + `StandardExpenditureKindID()` with job-aware capital/project                                                                                                                                                  |
| 15 | `import_data/extract/augment_expenses.go` (~L23-30)                           | `ensureKindColumnWithDefault` → job-aware backfill                                                                                                                                                                                             |
| 16 | `ui/src/lib/components/PurchaseOrdersEditor.svelte` (~L343-344, L481-482)     | Two locations: `name === "standard"` → job-aware capital/project                                                                                                                                                                               |
| 17 | `ui/src/lib/components/ExpensesEditor.svelte` (~L46-55)                       | Remove `name === "standard"` hardcode; look up by `item.kind`                                                                                                                                                                                  |
| 18 | `ui/src/lib/components/AdminProfilesEditor.svelte` (~L756, L766)              | Label text updates                                                                                                                                                                                                                             |
| 19 | `ui/src/routes/admin_profiles/[id]/details/+page.svelte` (~L172, L176)        | Label text updates                                                                                                                                                                                                                             |
| 20 | `app/internal/testutils/testutils.go` (~L122-128)                             | `WHERE name = 'standard'` → `'capital'`                                                                                                                                                                                                        |
| 21 | `app/hooks/hooks_test.go` (~L23)                                              | `DefaultExpenditureKindID()` → `DefaultCapitalExpenditureKindID()`                                                                                                                                                                             |
| 22 | `app/purchase_orders_test.go`                                                 | `standardKindID` → `capitalKindID`; add `projectKindID`; update all scenarios                                                                                                                                                                  |
| 23 | `app/purchase_orders_approvers_test.go` (~L250)                               | Rename test; use project kind ID                                                                                                                                                                                                               |
| 24 | `app/expenses_test.go` (~L182-204)                                            | Update test names; add "no-PO with job → project" case                                                                                                                                                                                         |
| 25 | `app/test_pb_data/data.db`                                                    | Apply migrations to update fixture                                                                                                                                                                                                             |
| 26 | `descriptions/authority_matrix.md`                                            | Replace "standard" references                                                                                                                                                                                                                  |
| 27 | `descriptions/purchase_orders.md`                                             | Replace "standard" references                                                                                                                                                                                                                  |
| 28 | `CLAUDE.md`                                                                   | Update "Expenditure Kinds" business concept description                                                                                                                                                                                        |
