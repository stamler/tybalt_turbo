# Legacy Job Fast Close (As Implemented)

## Executive Summary: Business Implications

The staged implementation delivers a controlled fast-close path for legacy imported projects while preserving strict validation for non-imported projects.

- Imported Active projects can be closed quickly from the UI using a dedicated endpoint.
- Non-imported projects are still subject to strict/full validation in this close flow.
- Proposal consistency is enforced: a referenced proposal must be `Awarded` by transaction end.
- Imported referenced proposals in `In Progress` or `Submitted` can be auto-awarded during close.
- Every successful close writes a project note; auto-award writes a separate proposal note.
- Close operations are transactional across project/proposal updates and note creation.
- UI now requires an explicit confirmation modal before close executes to reduce accidental closes.

---

## 1. Implemented Scope

### Backend

- Added `POST /api/jobs/{id}/close`.
- Added strict hook entry point for close-only strict validation path.
- Added jobs payload fields needed by list/details UI decisions:
  - `status`
  - `imported`

### Frontend

- Added fast-close action in:
  - Jobs List
  - Job Details
- Added confirmation modal with explicit side-effect warnings.
- Added referenced-proposal contextual warnings in that modal.

### Not changed

- Jobs Editor save flow remains unchanged.

---

## 2. Backend Behavior (Actual)

### 2.1 Endpoint

- Route: `POST /api/jobs/{id}/close`
- File: `/Users/dean/code/tybalt_turbo/app/routes/job_close_api.go`
- Route registration: `/Users/dean/code/tybalt_turbo/app/routes/routes.go`

### 2.2 Authorization

Request is allowed only when authenticated and either:

- user has `job` claim, or
- user is the job `manager` or `alternate_manager`.

### 2.3 Eligibility checks

Target job must:

- be a project (number must not start with `P`), and
- currently be `Active`.

### 2.4 Mode selection

- Target `_imported=true` -> `mode="bypass"`.
- Target `_imported=false` -> `mode="validated"`.

### 2.5 Referenced proposal rules

If project has `proposal`:

- Proposal `Awarded` -> proceed.
- Proposal terminal (`Not Awarded`, `Cancelled`, `No Bid`) -> reject with `proposal_terminal_status_blocks_close`.
- Proposal status `In Progress` or `Submitted` and proposal `_imported=true` -> auto-award in transaction:
  - set proposal status `Awarded`
  - set proposal `_imported=false`
  - create proposal auto-award note
- Otherwise reject with `proposal_not_awarded`.

### 2.6 Close and strict validation behavior

- Endpoint sets project status to `Closed`.
- If mode is `validated`:
  - calls `hooks.ProcessJobCoreStrict(...)` before save.
- If mode is `bypass`:
  - sets project `_imported=false` and skips full validation path.

### 2.7 Audit note behavior

Project note (always on successful close):

- note text: `Project closed via imported fast close flow`
- created via `client_notes`

Proposal note (only when auto-award occurs):

- note text pattern:
  - `Proposal auto-awarded during imported fast close of project <project_number> (<project_id>)`

Important implementation detail:

- `job_status_changed_to` is intentionally left unset for both close and auto-award notes in this flow.

### 2.8 Transaction semantics

All happen in one transaction:

1. proposal auto-award update (if applicable)
2. project close update
3. project note creation
4. proposal note creation (if applicable)

Any failure rolls back all of the above.

### 2.9 Response shape

Success response includes:

- `id`
- `status`
- `imported`
- `mode` (`bypass` or `validated`)
- `project_note_created`
- optional `proposal` object with:
  - `id`
  - `auto_awarded`
  - `from_status`
  - `to_status`
  - `proposal_note_created`

---

## 3. Hook Changes (Actual)

File: `/Users/dean/code/tybalt_turbo/app/hooks/jobs.go`

- `ProcessJobCore(...)` now delegates to internal mode-aware function with normal behavior.
- Added `ProcessJobCoreStrict(...)` to force full validation by disabling the status-only shortcut.
- `validateJob(...)` now takes `forceFullValidation bool`.

This preserves existing behavior for normal callers and enables strict validation only where explicitly requested.

---

## 4. UI Behavior (Actual)

### 4.1 Visibility conditions

Fast-close button is shown only for:

- project (not proposal)
- `status === "Active"`
- `imported === true`

Files:

- `/Users/dean/code/tybalt_turbo/ui/src/routes/jobs/list/+page.svelte`
- `/Users/dean/code/tybalt_turbo/ui/src/routes/jobs/[jid]/details/+page.svelte`

### 4.2 Confirmation modal

Shared component:

- `/Users/dean/code/tybalt_turbo/ui/src/lib/components/FastCloseConfirmPopover.svelte`

Behavior:

- clicking fast-close opens modal (does not close immediately)
- modal warns exactly what happens on close
- if referenced proposal exists, modal shows proposal status/imported context and auto-award/blocking outcomes
- close executes only after explicit user confirmation

### 4.3 Context loading

- List flow loads job details, then proposal details (if linked) for modal context.
- Details flow loads proposal details (if linked) for modal context.

---

## 5. API Payload Additions Used by UI

### Jobs list API

Files:

- `/Users/dean/code/tybalt_turbo/app/routes/jobs.sql`
- `/Users/dean/code/tybalt_turbo/app/routes/jobs_api.go`

Added fields:

- `status`
- `_imported AS imported`

### Job details API

Files:

- `/Users/dean/code/tybalt_turbo/app/routes/job_details.sql`
- `/Users/dean/code/tybalt_turbo/app/routes/job_details_api.go`

Added field:

- `_imported AS imported`

### Jobs store

File:

- `/Users/dean/code/tybalt_turbo/ui/src/lib/stores/jobs.ts`

- `JobApiResponse` now includes `status` and `imported`.
- `storeFields` now includes both so action rendering logic receives them in search results.

---

## 6. Tests (As Implemented)

### New endpoint test file

- `/Users/dean/code/tybalt_turbo/app/job_close_test.go`

Covered scenarios include:

- imported bypass close success
- imported close with imported proposal auto-award success
- non-imported strict validation failure
- terminal proposal blocks close
- auth/unauth rejections
- proposal target rejected
- non-active project rejected
- non-imported in-progress proposal cannot auto-award
- response without proposal object when no proposal link
- strict validation error shape

### API field exposure test

- `/Users/dean/code/tybalt_turbo/app/jobs_test.go`
- Added assertion coverage for jobs read endpoints exposing `status` and `imported`.

### Test fixtures

- Test DB includes dedicated fast-close fixtures used by these tests.
- Tests were written to avoid runtime setup mutation of fixture rows.

---

## 7. Error Codes Implemented in Close Endpoint

Implemented codes include:

- `job_not_found`
- `unauthorized`
- `not_a_project`
- `invalid_status_for_close`
- `proposal_terminal_status_blocks_close`
- `proposal_not_awarded`
- `error_creating_project_close_note`
- `error_creating_proposal_auto_award_note`
- `error_updating_job`
- `claim_check_failed`

---

## 8. Current Limitations / Follow-ups

- Jobs Editor still does not route close through this endpoint (intentional for this phase).
- No dedicated injected-failure rollback tests for note-creation failures are staged today.
- No separate telemetry/logging additions are staged specifically for this endpoint.
