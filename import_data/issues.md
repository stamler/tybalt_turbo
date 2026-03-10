# Import Tool Follow-Up Issues

This document records issues discovered during the Phase 2 text-seed migration work that should be addressed in a separate follow-up change.

The goal is to preserve the current working behavior while making the `import_data` bootstrap path cleaner, more standalone, and easier to reason about.

## Context

Phase 2 removed the tracked binary fixture DB from the repo and changed `import_data --init` so it no longer copies `../app/test_pb_data/data.db`.

The current implementation works, but it exposed two practical issues and one architectural smell:

- `./tool --init` is no longer truly self-contained
- the documented default `./tool --export` flow is no longer valid on a fresh clone unless a DB already exists
- `--init` currently builds a fully seeded DB and then deletes a large amount of data to get back to an import-ready baseline

## Issue 1: `./tool --init` is no longer a standalone binary workflow

### What changed

`import_data/tool.go` now implements `--init` by invoking:

- `go run ./cmd/testseed load ...`

inside the sibling `app` module.

That means a built `./tool` binary now depends on:

- a checkout that still has the sibling `../app` directory
- a working Go toolchain at runtime
- the `app/cmd/testseed` command being available from source

### Why this is a problem

The current documentation still presents `./tool` as a standalone command-line program:

- `import_data/README.md`
- `import_data/deployment_phases.md`

But the current `--init` implementation is no longer standalone in the old sense.

That creates an expectation mismatch:

- users think `./tool --init` works as a self-contained binary command
- the code now assumes repo-relative source layout and shells out to Go tooling

### Consequences

- a copied or prebuilt `./tool` binary will fail if it is run outside the repo layout
- deployment or operational scripts that expect a real standalone binary now have an implicit source-code dependency
- the runtime dependency on `go run` is hidden rather than explicit in the tool’s interface

## Issue 2: default `./tool --export` is no longer valid on a fresh clone

### What changed

The default `--db` path now points at:

- `../app/pb_data/data.db`

That is a better default for real import workflows, but it also means the old “just run `./tool --export`” examples are no longer universally valid.

### Why this is a problem

`tool.go` still resolves expenditure kind IDs from the target DB before export/import logic runs.

So on a fresh clone:

- if `../app/pb_data/data.db` does not exist yet
- `./tool --export`
- and `./tool --export && ./tool --import ...`

will fail unless the operator first creates a DB with `--init` or provides an explicit `--db`.

### Consequences

- the documented examples are misleading for fresh-clone usage
- the default-path behavior is now coupled to prior bootstrap state
- the tool’s ergonomics are worse than they appear from the docs

## Architectural Smell: `--init` seeds a full DB and then deletes data

### Current behavior

Today `--init` effectively does this:

1. build a blank DB
2. run migrations
3. load the full text seed set
4. copy the DB into the requested target path
5. run `cleanupFreshDatabase(...)`
6. delete imported/test-specific business rows

### Why this is undesirable

This means the system is doing work just to undo it:

- inserting rows only to delete them again
- relying on cleanup rules to define the import baseline
- coupling import bootstrap semantics to the shape of the test fixture set

That is both wasteful and harder to reason about than necessary.

## Proposed Solution

Create a shared DB-build abstraction whose job is:

- create blank DB
- run migrations
- load a named seed profile

Then use that same abstraction from both:

- `app/cmd/testseed`
- `import_data`

### Recommended seed profiles

Start with explicit named profiles:

- `test-full`
  - the current canonical full test fixture set
- `import-baseline`
  - only the lookup/config/reference rows needed to initialize an import-ready app DB

### Expected behavior after refactor

Tests:

1. create blank DB
2. apply migrations
3. load `test-full`

`import_data --init`:

1. create blank DB
2. apply migrations
3. load `import-baseline`

With this design:

- no cleanup pass is needed just to remove seeded rows
- import bootstrap becomes explicit instead of “test DB minus deletions”
- DB construction has one clear owner

## Recommended Package Boundary

Longer term, the cleanest split is:

- shared DB build package in `app`
  - owns migration application
  - owns seed-package loading
  - owns named seed profiles
- `app/cmd/testseed`
  - maintenance CLI around that package
- `import_data`
  - either calls a stable CLI entrypoint or, after a module-boundary refactor, reuses the shared builder directly

## Suggested Follow-Up Tasks

1. Introduce an explicit `import-baseline` profile in `app/test_seed_data`.
2. Refactor the DB builder so profile selection is a first-class concept.
3. Change `import_data --init` to build `import-baseline` directly.
4. Remove `cleanupFreshDatabase` from the init path once the baseline profile fully replaces it.
5. Update `import_data/README.md` and `import_data/deployment_phases.md` so example commands reflect the final bootstrap flow.
6. Optionally move `import_data/tool.go` into `import_data/cmd/importdata/main.go` with a thinner `main`.

## Summary

The current Phase 2 implementation is functional, but it is a transitional shape.

The cleaner end state is:

- one shared DB builder
- multiple explicit seed profiles
- no shelling out to `go run` from `./tool`
- no full-seed-then-delete bootstrap flow
