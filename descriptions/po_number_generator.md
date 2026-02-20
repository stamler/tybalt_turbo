# PO number generator: current behavior and risks

This document describes the current `GeneratePONumber` implementation: what it does, how it works, where it can fail, and how range limits behave.

## Where it lives / call path

- Generator function: `app/routes/purchase_orders.go` (`GeneratePONumber`).
- Runtime call site: `app/routes/purchase_orders.go` inside `createApprovePurchaseOrderHandler`, via `activateRecord()`.
  - The approval handler runs in `app.RunInTransaction(...)`.
  - `activateRecord()` sets `status="Active"` and assigns `po_number` only when the record has a blank `po_number`.

Practical implication: PO numbers are assigned at activation time, not when the PO is created.

## Formats generated

The generator supports two formats:

1) Parent PO number: `YYMM-NNNN`

- `YY` = last two digits of current year (currently formatted with `%d`, not `%02d`)
- `MM` = two-digit month (`01`-`12`)
- `NNNN` = auto-generated sequence `0001`-`4999`
- `5000+` is treated as reserved for manual/imported numbers

2) Child PO number: `PARENT_PO_NUMBER-XX`

- `XX` = two-digit suffix (`01`-`99`)
- Child numbers append to the parent record’s stored `po_number` as-is

## Parent generation algorithm

When `record.GetString("parent_po") == ""`:

1. Build `prefix` from server time:
   - `prefix = fmt.Sprintf("%d%02d-", currentYear%100, currentMonth)`
2. Define upper bound for auto-generated parent numbers:
   - `upperBound = prefix + "5000"`
3. Fetch the highest existing eligible parent PO by filter:
   - `parent_po = '' && po_number ~ (prefix + '%') && po_number < upperBound`
   - Sort: `-po_number`, Limit: `1`
4. Parse `lastNumber`:
   - `numericSuffix := strings.TrimPrefix(lastPO, prefix)`
   - `numericSuffix, _, _ = strings.Cut(numericSuffix, "-")` (defensive)
   - `lastNumber = strconv.Atoi(numericSuffix)`
5. Generate candidates `lastNumber+1 .. 4999`:
   - `newPONumber := fmt.Sprintf("%s%04d", prefix, i)`
6. For each candidate, query exact uniqueness (`po_number = {:poNumber}`), return first free.
7. If nothing is free up to `4999`, return `unable to generate a unique PO number`.

## Child generation algorithm

When `record.GetString("parent_po") != ""`:

1. Lookup parent record by id.
   - If missing: `parent PO not found`
2. Read parent `po_number`.
   - If blank: `parent PO does not have a PO number`
3. Find highest existing child for that parent:
   - Filter: `parent_po = {:parentId} && po_number != ''`
   - Sort: `-po_number`, Limit: `1`
4. Compute next suffix:
   - Start `nextSuffix = 1`
   - If a child exists: take last 2 chars (`lastChild[len(lastChild)-2:]`), parse with `fmt.Sscanf`, then increment
5. If `nextSuffix > 99`: error `maximum number of child POs reached (99) for parent ...`
6. Build child number `parentNumber + "-" + %02d(nextSuffix)`
7. Validate uniqueness by exact match query before returning.

## Current failure modes

### 1) Concurrency collisions

The algorithm is read-max then generate-check. Two concurrent transactions can still race to the same value. The unique index prevents duplicates, but one transaction may fail during save and surface as a generic update error.

### 2) Malformed parent `po_number` values

Parent parsing now strips trailing `-...`, but it still expects the first segment after `YYMM-` to be numeric. Bad manual data (for example `2401-ABCD`) can still cause `error parsing last PO number`.

### 3) Brittle child suffix parsing

Child parsing still slices the last two characters and does not validate `Sscanf` success. Unexpected formats can produce incorrect suffix selection or panic on too-short strings.

### 4) Hard limits

- Parent auto-generated range is capped at `4999` per `YYMM-` prefix.
- Child range is capped at `99` per parent.

If limits are reached, generation fails.

### 5) Time basis

Prefix generation uses `time.Now()` on the server clock/time zone. If business expectations use a different zone, month boundaries may differ from user expectations.

## Range exhaustion behavior

### Parent range (`YYMM-0001`..`YYMM-4999`)

- Gaps are not reused: generation starts from `(max + 1)`.
- Reserved `5000+` values are ignored for auto-generation math.
- If only `5000+` values exist for a prefix, auto-generation starts at `0001`.

### Child range (`01`..`99`)

- Gaps are not reused: generation uses `(max suffix + 1)`.
- If `99` exists, the next child generation fails by design.

## Notes on data shape

- The system can contain mixed historical parent formats (for example `2024-0008` and `2401-0009`) because child generation appends to the parent number exactly as stored.
- Collection regex/schema and historical fixtures may reflect older numbering semantics, but generation logic is the source of truth.

## Current improvement opportunities

1. Zero-pad year in prefix (`%02d`) to keep it structurally fixed-width.
2. Add retry on unique-constraint collisions for concurrent approvals.
3. Decide whether to keep strict non-reuse of gaps or implement first-free scanning.
4. Harden child suffix parsing (split-based parse + validation).
