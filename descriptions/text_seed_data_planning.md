# Text Seed Data Plan

## Goal

Replace the committed binary test database (`app/test_pb_data/data.db`) with text fixtures (Frictionless-style `datapackage.json` + per-table CSV files), and make tests build a fresh DB from migrations + seed data at runtime.

This plan also updates `import_data/tool.go --init` so it no longer depends on copying `app/test_pb_data/data.db`.

## Current State (what we are changing)

- Test fixtures are currently stored as a committed SQLite binary at `app/test_pb_data/data.db` (~976 KB).
- Most tests load from `tests.NewTestApp("./test_pb_data")` or `tests.NewTestApp("../test_pb_data")`.
- `import_data/tool.go` currently:
  - defaults `--db` to `../app/test_pb_data/data.db`
  - uses `--init` to copy `../app/test_pb_data/data.db` to `../app/pb_data/data.db`
  - then deletes many rows via `cleanupFreshDatabase`.

## Target State

- Source of truth is text under `app/test_seed_data/`:
  - `app/test_seed_data/datapackage.json`
  - `app/test_seed_data/data/<table>.csv` (one CSV per table)
- Tests create a fresh DB from scratch by:
  1. creating an empty DB
  2. applying migrations
  3. loading CSV resources from `datapackage.json`
- `app/test_pb_data/data.db` is no longer committed.
- `import_data --init` initializes without copying from test DB.

---

## 1) Seed Data Format

### Directory layout

```text
app/test_seed_data/
  datapackage.json
  data/
    _superusers.csv
    app_config.csv
    branches.csv
    ...
```

### `datapackage.json` conventions

- Use Frictionless-style `resources` entries with:
  - `name` = table name
  - `path` = `data/<table>.csv`
  - `schema.fields` derived from SQLite `PRAGMA table_info`
  - `schema.primaryKey` from SQLite PK columns
- Add custom metadata key (e.g. `x-groups`) so the same package can support:
  - `test-full` (full test fixtures)
  - `init-baseline` (minimal tables needed for `import_data --init`, if different)

### CSV rules

- UTF-8, header row, LF line endings.
- Deterministic row ordering per table (`ORDER BY` PK columns, fallback `rowid`).
- Represent SQL `NULL` as `\\N` (so empty string remains a real empty string).
- Keep values as DB literals (timestamps/JSON/bools) to avoid lossy conversions.

---

## 2) One-Time Dump from Existing `data.db`

Add a small Go utility in `app/cmd/testseed` (or equivalent) with subcommands:

- `dump`
  - input: existing `app/test_pb_data/data.db`
  - output: `app/test_seed_data/datapackage.json` + CSV files
- `load`
  - input: empty/fresh DB path + datapackage dir + group (`test-full` / `init-baseline`)
  - action: load CSV rows into DB inside one transaction
- `verify` (recommended)
  - compares table counts/checksums between a source DB and a rebuilt DB.

Initial bootstrap steps:

1. Run `dump` from the current binary DB.
2. Rebuild a scratch DB via migrations + `load --group test-full`.
3. Run `verify` to confirm parity with the old fixture.
4. Run `go test ./...` from `app/`.

---

## 3) Test Runtime Workflow Changes

### New internal seeding helper

Create a helper package (example: `app/internal/testseed`) that does:

- `EnsureTemplateDB(t)` via `sync.Once` per test process:
  1. create temporary seed workspace
  2. create fresh DB
  3. apply migrations
  4. load `test-full` CSV seed data
- `NewSeededTestApp(t)`:
  - calls `tests.NewTestApp(<template-dir>)` so each test still gets isolated DB copies.

Important: keep this package free of `hooks` and `routes` imports to avoid cycles in `hooks` package tests.

### Update test call sites

- Update `app/internal/testutils/testutils.go` to use `testseed.NewSeededTestApp`.
- Replace direct `tests.NewTestApp("../test_pb_data")` and `tests.NewTestApp("./test_pb_data")` usages in:
  - `app/hooks/*_test.go`
  - `app/routes/*_test.go`
  - `app/utilities/*_test.go`
  - `app/absorb_test.go`

This removes hard dependency on a committed `app/test_pb_data/data.db`.

### Migration application note

Use the same migration path that actually creates the app collections in practice (not just PocketBase core auth tables). Validate this with an automated check in the seeding helper (e.g. assert key tables like `jobs`, `purchase_orders`, `divisions` exist before loading CSV).

---

## 4) `import_data --init` Refactor Plan

### Current issue

`import_data/tool.go --init` copies `../app/test_pb_data/data.db` and cleans it. This will break once the binary DB is removed.

### Proposed behavior

Refactor `--init` to:

1. Create/overwrite `../app/pb_data/data.db` from scratch.
2. Apply migrations.
3. Optionally load seed group `init-baseline` from `app/test_seed_data` (if import depends on lookup/config rows not produced by migrations alone).

### Concrete code/docs updates in `import_data`

- In `import_data/tool.go`:
  - remove `copyFile` + `cleanupFreshDatabase` path for `--init`
  - stop hardcoding `src := "../app/test_pb_data/data.db"`
  - make `--init` call the new init path (fresh + migrate + optional baseline load)
  - update default `--db` away from test DB path
- Update docs:
  - `import_data/README.md`
  - `import_data/deployment_phases.md` (`--init` section)

---

## 5) Remove Binary DB from Git

1. `git rm app/test_pb_data/data.db`
2. Add ignore rule(s), e.g.:
   - `app/test_pb_data/data.db`
   - optionally `app/test_pb_data/*.db` (if all runtime DB artifacts should stay untracked)
3. Keep text fixtures tracked under `app/test_seed_data/`.

Also update comments/docs that explicitly say fixtures live in `test_pb_data/data.db`.

---

## 6) Ongoing Developer Workflow

When fixture data needs to change:

1. Make data changes in a local scratch DB (or via app UI against a local DB).
2. Run `cmd/testseed dump` to regenerate CSV/datapackage text files.
3. Run `cmd/testseed verify` and `go test ./...`.
4. Commit only text fixture diffs.

Result: fixture diffs become reviewable, merge conflicts are manageable, and tests remain reproducible.

---

## 7) Validation Checklist (acceptance criteria)

- `app/test_pb_data/data.db` is not tracked in git.
- Running `go test ./...` from `app/` passes without requiring a committed binary DB.
- Seeded DB is created from migrations + datapackage on test startup.
- `import_data --init` succeeds with no dependency on `app/test_pb_data/data.db`.
- Fixture updates produce human-reviewable CSV/JSON diffs.

---

## 8) Design Questions and Scope Controls (additive)

This section is intentionally additive. It does not replace the plan above; it captures decisions worth validating before implementation.

### A) Frictionless `datapackage.json` vs lighter manifest

The plan currently assumes Frictionless-style `datapackage.json`. That adds a specification layer and ecosystem compatibility, but it also adds abstraction and schema metadata that may duplicate what migrations already define.

Options to evaluate:

1. Keep Frictionless-style datapackage.
   - Pros: standard structure, explicit resources, easier external interoperability.
   - Cons: extra schema metadata to maintain.
2. Use a lightweight custom manifest (JSON/YAML).
   - Pros: simpler parser and lower maintenance burden.
   - Cons: custom format, less portable.
3. Use convention-only mapping (`<table>.csv` and code-defined load order).
   - Pros: smallest surface area.
   - Cons: less self-describing and more logic hidden in code.

Decision criterion: pick the smallest approach that still gives deterministic ordering, validation, and maintainable fixture updates.

### B) CSV NULL encoding (`\\N`) and parser implications

The current draft uses `\\N` as a NULL sentinel. This is a common bulk-load convention but not a CSV standard, and Go's `encoding/csv` does not treat it as NULL automatically.

Alternatives to consider:

1. Keep `\\N` and implement explicit sentinel decoding in the loader.
2. Use empty fields and rely on schema nullability + explicit column rules.
3. Use JSON/JSONL fixtures where `null` is first-class.

If CSV remains the format, include an explicit parsing contract in code and docs so empty string and NULL remain unambiguous.

### C) One seed group vs two (`test-full` and `init-baseline`)

The plan currently includes two seed groups. This is flexible, but it also increases complexity.

Validation gate before implementing grouping:

- Confirm whether `import_data --init` genuinely needs seed data that differs from test fixtures.
- If not, start with one seed set and add grouping only when a concrete need appears.

### D) Scope boundary for `import_data --init`

Refactoring `import_data --init` is a real dependency because it currently copies `app/test_pb_data/data.db`, but combining it with fixture-format migration increases scope and risk.

Two rollout options:

1. Single-phase rollout: fixtures + test runtime + `--init` refactor together.
2. Two-phase rollout:
   - Phase 1: text seed fixtures + test workflow migration.
   - Phase 2: `import_data --init` refactor and related docs.

Use the two-phase rollout if reducing blast radius and simplifying rollback is more important than completing all related changes in one PR.

### E) Suggested pre-implementation checkpoints

Before coding, record answers to:

1. Which fixture packaging format is selected (Frictionless, custom manifest, or convention-only)?
2. Which NULL strategy is selected (`\\N`, empty-field rules, or JSON fixtures)?
3. Is one seed group enough initially?
4. Is `import_data --init` in scope for the first implementation phase?

Document these in the PR description so reviewers understand the chosen tradeoffs.
