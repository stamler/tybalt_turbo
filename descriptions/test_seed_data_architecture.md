# Text Seed Data Plan

## Goal

Replace the committed binary test database (`app/test_pb_data/data.db`) with text fixtures (Frictionless-style `datapackage.json` + per-table CSV files), and make tests build a fresh DB from migrations + seed data at runtime.

This plan updates the test workflow to use text fixtures as the source of truth and moves DB construction ownership into `app/cmd/testseed`, including an explicit `import-baseline` seed profile for import workflows.

## Status

- Phase 1 is complete.
- Phase 2 is complete except for the optional future `import_data/cmd/...` cleanup.

What is already implemented:

- canonical text fixtures now live under `app/test_seed_data/`
- `app/cmd/testseed` exists with `dump`, `load`, and `verify`
- `app/cmd/testseed load` now supports explicit seed profiles:
  - `test-full`
  - `import-baseline`
- tests build from a cached migrated-and-seeded template DB instead of each test directly reading `app/test_pb_data`
- `app/internal/testutils/testutils.go` and direct test call sites now use `app/internal/testseed`
- the historical migration chain has been collapsed to the current snapshot baseline, so fresh blank-DB migration runs are now supported
- the documented runtime-only DB mutation exceptions remain in place
- `import_data/tool.go` no longer owns DB construction and no longer exposes `--init`

What is still outstanding:

- optionally clean up `import_data/tool.go` into `import_data/cmd/...`

## Recommended Rollout

Do this in two phases.

- Phase 1:
  - introduce text fixture source under `app/test_seed_data/`
  - add dump/load/verify tooling
  - make tests build from migrations + text seed data
  - switch test helpers and direct `tests.NewTestApp(...)` call sites over to the new seeded workflow
- Phase 2:
  - remove the binary fixture DB from git
  - update remaining docs and defaults that still point at `app/test_pb_data/data.db`
  - move `import_data/tool.go` under `import_data/cmd/...` if the CLI cleanup is still desired

This kept the first PR focused on test-fixture migration and reduced blast radius.

## Current State

- Text fixtures are now stored under `app/test_seed_data/` as `datapackage.json` plus one CSV per table.
- Tests now build through `app/internal/testseed`, which creates a blank DB, applies the current migration set, and then loads text fixtures into the result.
- The current seed package intentionally excludes `_migrations`; migration state comes from the code-defined migration set, not from fixture data.
- The local generated fixture DB at `app/test_pb_data/data.db` is no longer tracked in git, but can still be created for manual inspection or `testseed dump/verify`.
- `import_data/tool.go` currently:
  - defaults `--db` to `../app/pb_data/data.db`
  - requires the target DB to already exist
  - fails with an explicit instruction to run `cd app && go run ./cmd/testseed load --profile import-baseline --out ./pb_data` when the DB is missing
- Most additive test setup has already been pushed into the canonical fixture DB.
- The remaining runtime-only pre-test DB mutations are now explicitly documented in:
  - `app/test_pb_data/runtime_test_db_exceptions.md`
- Those remaining exceptions are intentional and should be preserved during the text-fixture migration unless a later design changes the architecture (for example, multiple fixture variants).

## Target State

- Source of truth is text under `app/test_seed_data/`:
  - `app/test_seed_data/datapackage.json`
  - `app/test_seed_data/data/<table>.csv` (one CSV per table)
- Tests create a fresh DB from scratch by:
  1. creating an empty DB
  2. applying migrations
  3. loading CSV resources from `datapackage.json`
- `app/test_pb_data/data.db` is no longer committed.
- import workflows initialize app DBs via `app/cmd/testseed load --profile import-baseline`, not via `import_data/tool.go`.

## Phase 1 Result

Phase 1 included:

- text fixture source under `app/test_seed_data/`
- dump/load/verify tooling
- test runtime creation from migrations + text seed data
- helper/call-site migration away from direct `test_pb_data` usage

Phase 1 intentionally did not include:

- `import_data/tool.go` refactor
- `import_data/README.md` updates beyond notes if absolutely needed
- changing import bootstrap
- removing `app/test_pb_data/data.db` from git before parity is proven
- additional test-behavior refactors unrelated to fixture loading

Phase 1 acceptance criteria that are now satisfied:

- tests no longer directly call `tests.NewTestApp("./test_pb_data")` / `tests.NewTestApp("../test_pb_data")`
- a fresh test DB is built from migrations + text fixtures
- `go test ./...` passes
- the runtime-only exception tests continue to pass with the same behavior
- dump/load/verify tooling can prove parity against the current known-good migrated fixture runtime

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
  - `import-baseline` (minimal tables needed for import bootstrap)

Current implementation note:

- The seed package now supports two explicit groups:
  - `test-full`
  - `import-baseline`

### CSV rules

- UTF-8, header row, LF line endings.
- Deterministic row ordering per table (`ORDER BY` PK columns, fallback `rowid`).
- Represent SQL `NULL` as `\\N` (so empty string remains a real empty string).
- Keep values as DB literals (timestamps/JSON/bools) to avoid lossy conversions.

Current implementation note:

- Keep the loader contract explicit in code so `\\N` and empty string remain unambiguous.
- If CSV null handling becomes awkward, prefer simplifying the loader contract over adding hidden conversion rules.

---

## 2) Dump/Load/Verify Workflow

Add a small Go utility in `app/cmd/testseed` (or equivalent) with subcommands:

- `dump`
  - input: existing `app/test_pb_data/` data directory
  - output: `app/test_seed_data/datapackage.json` + CSV files
- `load`
  - input: empty/fresh DB path + datapackage dir + profile (`test-full` / `import-baseline`)
  - action: load CSV rows into DB inside one transaction
- `verify` (recommended)
  - compares table counts/checksums between the current migrated test runtime and a rebuilt DB.

Current bootstrap/maintenance workflow:

1. Run `dump` from the current test data directory.
2. Rebuild a scratch DB via migrations + `load --profile test-full`.
3. Run `verify` to confirm parity with the current migrated fixture runtime.
4. Run `go test ./...` from `app/`.

Important implementation notes:

- `dump` and `verify` intentionally bootstrap a PocketBase test app first, rather than reading the raw binary DB file shape directly.
- This matters because the committed `app/test_pb_data` fixture can lag behind the schema shape produced by current migrations, while tests always run against the migrated shape.
- `load`/template creation do not import `_migrations`; they rely on the current code-defined migration set and then load fixture tables.
- After the migration squash to the snapshot baseline, template creation now works from a truly blank DB.

---

## 3) Test Runtime Workflow Changes

### New internal seeding helper

The helper package `app/internal/testseed` now does:

- `TemplateDir()` / `EnsureTemplateDir(t)` via `sync.Once` per test process:
  1. create temporary seed workspace
  2. create fresh DB
  3. apply migrations
  4. load `test-full` CSV seed data
- `NewSeededTestApp(t)`:
  - calls `tests.NewTestApp(<template-dir>)` so each test still gets isolated DB copies.

Important: keep this package free of `hooks` and `routes` imports to avoid cycles in `hooks` package tests.

Recommended Phase 1 implementation order:

1. add `app/internal/testseed`
2. make `app/internal/testutils/testutils.go` use it
3. replace remaining direct `tests.NewTestApp("../test_pb_data")` / `tests.NewTestApp("./test_pb_data")` call sites

This work is now complete.

### Update test call sites

- Update `app/internal/testutils/testutils.go` to use `testseed.NewSeededTestApp`.
- Replace direct `tests.NewTestApp("../test_pb_data")` and `tests.NewTestApp("./test_pb_data")` usages in:
  - `app/hooks/*_test.go`
  - `app/routes/*_test.go`
  - `app/utilities/*_test.go`
  - `app/absorb_test.go`

This removed the direct per-test dependency on a committed `app/test_pb_data/data.db`.

Important constraint:

- Do not change the semantics of the runtime-only exceptions documented in `app/test_pb_data/runtime_test_db_exceptions.md`.
- Those tests should continue to mutate the per-test copied DB after startup.

### Migration application note

Use the same migration path that actually creates the app collections in practice (not just PocketBase core auth tables). In the current implementation, `testseed` boots a fresh PocketBase app runtime, applies the current snapshot-based migration set, and then loads fixtures so the generated template matches test runtime behavior.

---

## 4) Import Bootstrap Current Behavior

`import_data/tool.go` no longer owns DB initialization.

Import workflows now initialize a target app DB explicitly from the app module:

```bash
cd app
go run ./cmd/testseed load --profile import-baseline --out ./pb_data
```

That command:

1. Creates a blank PocketBase data directory.
2. Applies the current migration set.
3. Loads only the `import-baseline` seed profile from `app/test_seed_data`.

`import_data/tool.go` then operates against the resulting `../app/pb_data/data.db`.

### Concrete code/docs updates in `import_data`

- In `import_data/tool.go`:
  - default `--db` now points at `../app/pb_data/data.db`
  - `--init` has been removed
  - missing target DBs now produce an actionable error that points operators at `app/cmd/testseed load --profile import-baseline`
- CLI structure cleanup:
  - `import_data/tool.go` is a command-line program and should eventually follow normal Go `cmd/` layout
  - because `import_data/` is its own Go module, the right target is `import_data/cmd/importdata/main.go`, not the repo-level `app/cmd/`
  - when this moves, keep `main.go` thin and extract orchestration logic into a non-`main` package inside the `import_data` module
  - recommended split:
    - `import_data/cmd/importdata/main.go` for flag parsing and process exit behavior
    - package code elsewhere in `import_data/` for `RunInit`, `RunImport`, `RunExport`, and related helpers
  - this refactor was not required for Phase 1, but it fit naturally into Phase 2
- Update docs:
  - `import_data/README.md`
  - `import_data/deployment_phases.md` (bootstrap section)

---

## 5) Remove Binary DB From Git

This is now complete.

1. `app/test_pb_data/data.db` is no longer tracked in git.
2. `.gitignore` excludes the local generated file.
3. Text fixtures remain tracked under `app/test_seed_data/`.

Developers can regenerate the local scratch DB when needed with:

- `cd app && go run ./cmd/testseed load --profile test-full --out ./test_pb_data`

That local DB is still useful for:

- manually running the app against fixture data
- `cmd/testseed dump`
- `cmd/testseed verify`

---

## 6) Ongoing Developer Workflow

When fixture data needs to change:

1. Make data changes in a local scratch DB (or via app UI against a local DB).
2. Run `cmd/testseed dump` to regenerate CSV/datapackage text files.
3. Run `cmd/testseed verify` and `go test ./...`.
4. Commit only text fixture diffs.

Result: fixture diffs become reviewable, merge conflicts are manageable, and tests remain reproducible.

### When migrations require seed updates

Not every new migration requires regenerating `app/test_seed_data`, but many will.

Use this rule of thumb:

- If a migration is schema-only and the new columns/collections have safe defaults that do not matter to current fixture realism or test behavior:
  - seed regeneration may not be necessary immediately.
- If a migration inserts rows, renames values, backfills data, changes config/stateful defaults, or otherwise changes the data shape tests depend on:
  - regenerate the text seed data.
- If a migration adds a new field or collection that should be represented in the canonical test fixture set:
  - regenerate the text seed data.

Important nuance:

- `cmd/testseed verify` compares the columns declared in the seed package resources.
- That means a newly added column with a harmless default may not force a seed update by itself.
- Developers still need to make an intentional decision about whether the new field should be captured in the canonical fixture set.

Recommended post-migration check:

1. Ask whether the migration changed fixture data, seeded defaults, or any rows/fields tests rely on.
2. If yes, run:
   - `go run ./cmd/testseed dump --data-dir ./test_pb_data --out ./test_seed_data`
   - `go run ./cmd/testseed verify --data-dir ./test_pb_data --seed-dir ./test_seed_data`
   - `go test ./...`
3. Commit the migration and the resulting text seed diffs together when they are logically coupled.

---

## 7) Validation Checklist (acceptance criteria)

Phase 2 final-state checklist:

- `app/test_pb_data/data.db` is not tracked in git.
- Running `go test ./...` from `app/` passes without requiring a committed binary DB.
- Seeded DB is created from migrations + datapackage on test startup.
- `import_data` no longer depends on a tracked binary fixture DB or on `--init`.
- Fixture updates produce human-reviewable CSV/JSON diffs.

Phase 1 completed checklist:

- `cmd/testseed dump/load/verify` works against the current known-good fixture DB
- test helpers and direct test call sites can build from text seed data
- `go test ./...` passes
- runtime-only exception tests still pass unchanged in behavior
- `_migrations` is no longer part of the text fixture package
- binary DB removal was deferred from Phase 1 and completed in Phase 2

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

The current implementation uses `\\N` as a NULL sentinel. This is a common bulk-load convention but not a CSV standard, and Go's `encoding/csv` does not treat it as NULL automatically.

Alternatives to consider:

1. Keep `\\N` and implement explicit sentinel decoding in the loader.
2. Use empty fields and rely on schema nullability + explicit column rules.
3. Use JSON/JSONL fixtures where `null` is first-class.

If CSV remains the format, include an explicit parsing contract in code and docs so empty string and NULL remain unambiguous.

### C) One seed group vs two (`test-full` and `import-baseline`)

This is now resolved.

The implementation uses two explicit seed groups:

- `test-full` for the canonical test fixture runtime
- `import-baseline` for app DB bootstrap used by import workflows

The two-profile split removed the earlier "seed full DB, then delete rows"
transition path from `import_data`.

### D) Scope boundary for import bootstrap

`import_data --init` was a real dependency because it originally copied `app/test_pb_data/data.db`. That transitional refactor has now been replaced entirely by app-owned bootstrap through `app/cmd/testseed load --profile import-baseline`, but the rollout sequencing note still explains why it was safer to land after the Phase 1 seed/test work.

Two rollout options:

1. Single-phase rollout: fixtures + test runtime + `--init` refactor together.
2. Two-phase rollout:
   - Phase 1: text seed fixtures + test workflow migration.
   - Phase 2: import bootstrap refactor and related docs.

Use the two-phase rollout if reducing blast radius and simplifying rollback is more important than completing all related changes in one PR.

Recommended answer:

- use the two-phase rollout
- keep import bootstrap changes out of the first implementation PR

### E) Suggested pre-implementation checkpoints

Before coding, record answers to:

1. Which fixture packaging format is selected (Frictionless, custom manifest, or convention-only)?
2. Which NULL strategy is selected (`\\N`, empty-field rules, or JSON fixtures)?
3. Is one seed group enough initially?
4. Is import bootstrap in scope for the first implementation phase?

Document these in the PR description so reviewers understand the chosen tradeoffs.

Recommended initial answers:

1. Packaging format: keep the planned `datapackage.json`, but keep loader logic simple and local-first.
2. NULL strategy: CSV with explicit `\\N` handling in loader code.
3. Seed groups: one group initially, `test-full`.
4. Import bootstrap: not in scope for the first implementation phase.
