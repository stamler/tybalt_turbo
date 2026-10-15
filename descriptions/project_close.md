# Close Project

## Summary

Any `Active` project can be closed with one action, **Close project**, from the
job page's "⋯" menu or the jobs list. Closing has no side effects on time,
expenses, or purchase orders, so it does not need the project's billing data.

- Closing requires the `job` claim, and jobs editing must be enabled.
- No invoicing profile, project value, or location is needed. An older project
  with `PO` or `Unauthorized` as its authorizing document is stored as `PA`, as
  on any project save. This is the same retirement rule as a status-only close in the job
  editor, so the editor and the action cannot disagree.
- A referenced proposal must be `Awarded` when the transaction ends. An imported
  proposal that is still `In Progress` or `Submitted` is awarded by the close.
- Every close writes a project note. An auto-award writes a proposal note.
- The close, the proposal update, and the notes commit in one transaction.
- The UI asks for confirmation first, and says what will happen to the proposal.

History: until October 2026 this was an imported-only "fast close". Imported
projects bypassed validation, and every other project went through full
validation, which made the action stricter than the editor. The action now
applies one rule to every project.

---

## 1. Endpoint

- Route: `POST /api/jobs/{id}/close`
- Handler: `app/routes/job_close_api.go`
- Registration: `app/routes/routes.go`

### 1.1 Authorization

The authenticated user must have the `job` claim. Managers and alternate
managers without `job` can change allocations only (see
`project_authorizations.md`). `hooks.ProcessJobCore` also requires the user to
be active.

### 1.2 Eligibility

The target job must:

- be a project (its number does not start with `P`), and
- currently be `Active`.

### 1.3 Referenced proposal rules

If the project has a `proposal`:

- `Awarded`: the close proceeds.
- `Not Awarded`, `Cancelled`, or `No Bid`: rejected with
  `proposal_terminal_status_blocks_close`.
- `In Progress` or `Submitted`, and the proposal is imported: auto-awarded in
  the transaction. The proposal's status becomes `Awarded`, its `_imported`
  becomes false, and a proposal note is created.
- Any other case: rejected with `proposal_not_awarded`.

### 1.4 Close and validation

The endpoint sets the project's status to `Closed` and calls
`hooks.ProcessJobCore`. Because only the status changed, the save is a project
retirement (`isProjectRetirement` in `app/hooks/jobs.go`), and `validateJob`
returns before any billing check. `ProcessJobCore` clears `_imported`, because
a field changed, so the close is written back to the legacy system.

### 1.5 Audit notes

Project note, on every successful close:

- `Project closed with the Close project action`

Proposal note, only when an auto-award happens:

- `Proposal auto-awarded when project <project_number> (<project_id>) was closed`

`job_status_changed_to` is left unset on both notes.

### 1.6 Transaction

In one transaction:

1. proposal auto-award (if applicable)
2. project close
3. project note
4. proposal note (if applicable)

Any failure rolls back all of them.

### 1.7 Response

- `id`
- `status`
- `imported` (always false after a close)
- `project_note_created`
- optional `proposal`: `id`, `auto_awarded`, `from_status`, `to_status`,
  `proposal_note_created`

### 1.8 Error codes

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

Hook errors from `ProcessJobCore`, such as jobs editing being disabled, keep
their own status and shape.

---

## 2. The editor's status-only close

A save that changes only a project's status to `Closed` or `Cancelled` is a
retirement. `ProcessJobCore` decides this before `cleanJob` normalizes the
record, and `validateJob` then skips every billing check: no invoicing profile
or project value. A save that changes any other field, or that sets a
project `Active`, gets full validation.

The editor cannot reconcile a referenced proposal. That is the reason to use
**Close project** for a project whose imported proposal is not yet `Awarded`.

---

## 3. UI

### 3.1 Where the action appears

**Close project** is shown for a project (not a proposal) whose status is
`Active`, when jobs editing is enabled and the user has the `job` claim:

- the "⋯" menu in `ui/src/lib/components/jobs/JobHeader.svelte`
- the row actions in `ui/src/routes/jobs/list/+page.svelte`

### 3.2 Confirmation

`ui/src/lib/components/CloseProjectConfirmPopover.svelte`:

- The action opens the dialog. Nothing closes until the user confirms.
- If the project has a proposal, the dialog loads it first. It warns when the
  proposal will be awarded, and it disables **Close project** when the
  proposal blocks the close.

---

## 4. Tests

- `app/job_close_test.go`: closes for imported and incomplete projects, the
  proposal rules, authorization, eligibility, and the editor's status-only close
  following the same rule.
- `app/project_value_test.go`, `app/project_completion_date_test.go`, and
  `app/job_billing_test.go`: closing without a value, completion date, or
  current billing data.
- `ui/tests/closeProject.browser.mjs`: menu visibility, the proposal outcomes
  in the dialog, and the close request.
