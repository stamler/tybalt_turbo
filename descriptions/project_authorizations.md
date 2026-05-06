# Project Authorization Documents

* STATUS: UNIMPLEMENTED, DRAFT, REQUESTING FEEDBACK *

Turbo needs a formal project authorization flow for projects whose authorizing
document is a Project Authorization (`PA`). The current `jobs.authorizing_document`
field records the business authorization type (`Unauthorized`, `PO`, or `PA`),
but it does not prove that Accounting has reviewed a scanned PA document. This
feature adds that proof and optionally blocks downstream project spending and
time capture until the proof exists.

## Goals

* Store the scanned PA PDF on the project job record.
* Detect duplicate uploaded PA documents across all jobs.
* Give Accounting a queue of uploaded-but-unreviewed PAs.
* Approve a PA only when Accounting approves the same file they reviewed.
* Prevent timesheet bundling, purchase order saves, and expense saves against
  unapproved PA projects when enforcement is enabled.
* Allow admins to revoke PA approval without deleting the uploaded PDF.

## Current Model To Preserve

`jobs.authorizing_document` already identifies the kind of project authorization.
Existing code treats `authorizing_document = "PA"` as project-authorized in PO
visibility queries. That is too weak for this new workflow because it only means
the project is marked as PA-backed; it does not mean a scanned PA exists or has
been reviewed.

The new approval state should therefore be separate from `authorizing_document`:

* `authorizing_document = "PA"`: the project is expected to be PA-backed.
* `project_authorization_doc` is populated: a scanned PA PDF has been uploaded.
* `pa_reviewed` and `pa_reviewer` are populated: Accounting has approved the
  currently uploaded PA file.

Any code currently using `authorizing_document = "PA"` as a proxy for project
authorization should be revisited where actual approval matters.

## Fundamental Open Decision

Management and administrators must decide whether the new PA approval gate
applies only to jobs where `authorizing_document = "PA"` or whether it replaces
the current PO/PA distinction and applies to all projects.

This is a fundamental architectural blocker. Implementation should not move
forward until this decision is made because it affects schema meaning,
migration/backfill strategy, PO behavior, UI copy, validation hooks, and how
existing projects are interpreted.

The rest of this document assumes the narrower interpretation: the new
scanned-document approval gate applies to PA-backed projects, not PO-backed
projects. If the intended policy is that every project must have a scanned PA
regardless of `authorizing_document`, then `authorizing_document = "PO"` and the
existing `client_po` path need to be redefined or deprecated as part of this
feature.

## Data Model

Add this field to `branches`:

| Field | Type | Notes |
| --- | --- | --- |
| `manager` | relation to `users` | Branch manager for the branch. Used as one of the operational users allowed to upload a signed PA document. |

Add these fields to `jobs`:

| Field | Type | Notes |
| --- | --- | --- |
| `project_authorization_doc` | file | Required to be a PDF when present. Stores the scanned PA. Configure the PocketBase file field with `mimeTypes = ["application/pdf"]`. |
| `project_authorization_doc_hash` | text | Server-calculated content hash of `project_authorization_doc`. Must be globally unique across `jobs` when non-empty. |
| `pa_reviewer` | relation to `users` | Accounting user who approved the current PA document. Blank until Accounting approves. |
| `pa_reviewed` | datetime | Accounting approval timestamp. Blank until Accounting approves. |

Hashing must be server-owned. Clients may upload files, but they must not submit
or override the hash. A changed unreviewed file must always produce a new hash,
and any change to an unreviewed `project_authorization_doc` must keep
`pa_reviewed` and `pa_reviewer` blank.
Use the existing attachment hash convention: SHA-256 over the raw uploaded file
bytes, stored as a lowercase hex string. Turbo already uses this through
`CalculateFileFieldHash` for expense and purchase order attachment hashes.
Turbo's existing PO and expense attachment file fields already use PocketBase
`mimeTypes` allow lists, so the PA field should use the same schema mechanism
with PDF as the only allowed type.

Uniqueness should be enforced at the database level if PocketBase schema/index
support allows a partial unique index for non-empty hashes. If not, enforce it
inside the same transaction that saves the uploaded document and treat duplicate
hashes as a validation error on `project_authorization_doc`.

## Upload Permissions

The meeting notes say that either the project manager or a `job` claim holder can
upload the PA after the job is created. Later feedback also says the BM may be
the last signer and may upload the signed PA. In the current schema, the project
manager is `jobs.manager`; there is no separate `project_manager` field, and the
branch manager should be represented as `branches.manager`.

Because the current `jobs` collection update rule is scoped to `job` claim
holders, manager, alternate-manager, and branch-manager upload access will need
either:

* a dedicated route such as `POST /api/jobs/:id/project_authorization_doc`, or
* a collection rule change that allows `jobs.manager`, `jobs.alternate_manager`,
  and the related branch's `manager` to update only the PA file fields.

A dedicated route is safer because it can enforce file type, hashing,
duplicate-hash checks, and approval reset behavior in one place without opening
general job editing to managers.

Upload should be allowed only when:

* the caller has the `job` claim, or
* the caller is the job's `manager`, or
* the caller is the job's `alternate_manager`, or
* the caller is the `manager` for the job's related `branch`.

Upload permission is operational access, not approval authority. The uploaded PA
is expected to already include the required paper approval signature before it is
submitted to Turbo. Turbo does not model the Functional Authority Matrix, the
division-manager identity, value thresholds, or executive approval routing; those
requirements belong to the external PA form/signature process. Accounting
approval in Turbo confirms the uploaded PA package is complete and unlocks the
project for downstream use.

Approved PA documents are immutable. Once `pa_reviewed` and `pa_reviewer` are
populated, nobody may replace or remove `project_authorization_doc`, including
admins, managers, alternate managers, branch managers, and `job` claim holders.
An admin must revoke the PA approval first; only then may the unreviewed
document be changed.

## File Visibility

The PA PDF should be viewable by all job viewers. In the current Turbo model,
job viewing is broadly available to authenticated users, so the uploaded PA PDF
is not a confidential Accounting-only attachment. The UI should not hide the
document from ordinary authenticated job viewers, even though upload, approval,
and revocation remain permissioned actions.

## Accounting Approval Queue

Users with the `accounting` claim get a PA approval queue and are the only
non-admin users who can approve PA documents. The existing `payables_admin`
claim is related to payables workflows but is not the same claim and does not
grant PA approval authority.

The required project-side approval happens outside Turbo on the signed PA form.
Accounting approval is the only approval action represented in Turbo. By
approving, Accounting confirms the uploaded PA package includes the required
paper approval and the internal/client information needed for AR processing.

The queue lists project jobs where:

* `status = "Active"`,
* `authorizing_document = "PA"`,
* `project_authorization_doc` is populated,
* `pa_reviewed` is blank, and
* `pa_reviewer` is blank.

The queue should include enough context for review:

* job id, number, description, client, manager, branch, status,
* uploaded file URL or preview action,
* `project_authorization_doc_hash`.

The browser must receive the hash for the file being reviewed. Approval then
submits that exact hash to the server.

## Approval Endpoint

Add an Accounting-only endpoint:

```text
POST /api/jobs/:id/project_authorization/approve
```

Request body:

```json
{
  "project_authorization_doc_hash": "hash-shown-to-reviewer"
}
```

The endpoint must run atomically. It should:

1. Confirm the caller has the `accounting` claim.
2. Load the job inside the transaction.
3. Confirm `project_authorization_doc` is populated.
4. Confirm `project_authorization_doc_hash` equals the submitted hash.
5. Confirm `pa_reviewed` and `pa_reviewer` are blank.
6. Set `pa_reviewed` to the current server timestamp.
7. Set `pa_reviewer` to the caller.

If the hash does not match, return a clear conflict-style error because the file
changed after Accounting opened it:

```json
{
  "code": "project_authorization_doc_changed",
  "message": "The project authorization document changed after you opened it. Please review the current document before approving."
}
```

If the PA is already approved, the endpoint must fail. Approval is not
idempotent: the first successful approval wins until an admin revokes it.

```json
{
  "code": "project_authorization_already_approved",
  "message": "This project authorization document has already been approved."
}
```

## Revocation

Once approved, PA approval is immutable except for admin revocation. Revocation
does not delete the uploaded PDF or hash. It clears only:

* `pa_reviewed`
* `pa_reviewer`

After revocation the PA is no longer approved, and the now-unreviewed PDF may be
replaced or removed by users who have PA upload permission.

Add an admin-only endpoint:

```text
POST /api/jobs/:id/project_authorization/revoke
```

The job details UI should surface this action only to users with the `admin`
claim and should require a modal confirmation. The confirmation copy should make
it clear that revoking approval may block new time, purchase orders, and
expenses when PA enforcement is enabled.

Revocation does not need audit history because it is an admin-only action. This
is an explicit limitation: after revocation, Turbo keeps no first-class history
of who previously approved the PA or when. If the PA is reviewed again after
revocation, the latest review wins because it is the only approval represented
by `pa_reviewed` and `pa_reviewer`.

## Enforcement Flag

Enforcement must be behind an `app_config` flag that defaults to permissive
behavior.

Recommended config:

```json
// key: "jobs"
{
  "enforce_project_authorization": false
}
```

Default: `false`.

When the key or property is missing, invalid, or unreadable, Turbo should not
block writes. This matches the requested rollout behavior: deployment can fall
back to permissive behavior if a problem appears.

This spec interprets "not active until approved" as operational blocking, not as
a new job status lifecycle. A job may still have `status = "Active"` so it can
appear in review queues and normal project views, but when enforcement is
enabled it is not usable for timesheet bundling, purchase order saves, or
expense saves until the signed PA document is uploaded and Accounting approval
exists.

## Enforcement Gates

When `jobs.enforce_project_authorization = true`, Turbo must gate the workflows
that make PA-backed project work usable downstream. The gate does not apply to
raw `time_entries` save.

### Time

Turbo should allow users to create and edit time entries even when the referenced
job requires a reviewed PA but does not yet have one. This gives the project
team time to upload and review the PA before payroll/accounting workflow reaches
the timesheet bundle step.

Timesheet bundle must fail atomically if any time entry being bundled references
a job that requires a reviewed PA and is not approved. No partial bundle should
be created in that case, and the existing time entries should remain unbundled.

### Purchase Orders

Turbo must reject purchase order save on both create and update when the PO
references a job that requires a reviewed PA and does not have one. This closes
the create/update loophole where a PO could be created jobless or against a
different job and then moved onto an unapproved PA project.

Implement this in the purchase order hook validation path so all normal PO save
paths receive the same field-level error, not only one HTTP route or UI flow.

### Expenses

Turbo must reject expense save on both create and update when the expense
references a job that requires a reviewed PA and does not have one.

This direct expense gate is required even though many job expenses already
require a PO, because current Turbo expense rules allow some job expenses
without a PO for exempt payment types such as mileage, fuel card, personal
reimbursement, and allowance. Those exempt expense types must also be blocked
until the referenced PA project is reviewed.

Implement this in the expense hook validation path so all normal expense save
paths receive the same field-level error, not only one HTTP route or UI flow.

### Approval Predicate

A project is PA-approved only when all of the following are true:

* the referenced job exists,
* `jobs.authorizing_document = "PA"`,
* `jobs.project_authorization_doc` is populated,
* `jobs.project_authorization_doc_hash` is populated,
* `jobs.pa_reviewed` is populated, and
* `jobs.pa_reviewer` is populated.

This gate should apply only to projects that require PA approval. Jobs whose
`authorizing_document` is `PO` should continue to use the existing client PO
rules. Jobs marked `Unauthorized` should not be treated as PA-approved merely
because PA enforcement is disabled.

Failure responses should be user-actionable:

```json
{
  "code": "project_authorization_not_approved",
  "message": "This project is not approved for time, purchase orders, or expenses yet. Please speak with the project's manager."
}
```

For purchase orders and expenses, the error should attach to the `job` field
where possible. For timesheet bundle, the response should identify at least one
blocking job and should prefer a structured list of all blocking jobs when
practical.

## UI Surfaces

### Job Details

For projects, show the PA status near the existing authorizing document details:

* no PA expected: no PA review controls,
* PA expected, no file: "PA document missing",
* PA uploaded, pending review: "PA pending Accounting approval",
* PA approved: reviewer and reviewed timestamp,
* PA revoked: same as uploaded pending review; prior approval history is not
  retained.

Users with upload permission should see a PDF upload control when
`authorizing_document = "PA"`. Admins should see a revoke action only when the PA
is currently approved.

### Accounting Queue

Accounting users should get a queue filtered to pending PA review. The approval
button should be disabled until the document has been opened or downloaded in
the current browser session, so the UI encourages actual review before approval.

### Downstream Surfaces

The PO editor, expense editor, and timesheet bundle flow should surface the
backend error without reducing it to a generic save failure. The message should
name the project manager when the relevant API payload already includes that
information; otherwise use the generic "project's manager" wording.

## Migration And Backfill

Initial migration should add the `branches.manager` field, the four `jobs`
fields, and the hash uniqueness guard. Existing projects should remain
unapproved until a scanned signed PA PDF is uploaded and Accounting approves it.

Do not backfill `pa_reviewed` based only on `authorizing_document = "PA"`. That
would recreate the current weak proxy and defeat the purpose of the review
workflow.

This is intentional. The rollout review period happens while
`jobs.enforce_project_authorization` is `false`. The flag should remain off
until the existing PA backlog has been uploaded and reviewed, because enabling
the flag will block purchase order saves, expense saves, and timesheet bundling
for still unreviewed PA projects.

## Open Questions

* Management and administrators must decide whether the PA approval gate applies
  only to jobs where `authorizing_document = "PA"` or whether it replaces the
  current PO/PA distinction and applies to all projects. This is a fundamental
  architectural blocker and must be resolved before implementation starts.

## Implementation Checklist

1. Add a PocketBase migration for `branches.manager`, the four `jobs` fields,
   and hash uniqueness.
2. Add server-owned PDF upload handling, PDF-only PocketBase MIME restriction,
   and hash calculation.
3. Allow upload for `job` claim holders, the assigned job manager, the alternate
   manager, and the related branch manager.
4. Prevent replacing or removing an approved PA document until admin revocation.
5. Add the Accounting approval queue endpoint/query.
6. Add the Accounting approval endpoint with hash compare-and-set semantics.
7. Add the admin revocation endpoint.
8. Add `jobs.enforce_project_authorization` config lookup with default `false`.
9. Gate timesheet bundle, purchase order create/update, and expense
   create/update when the flag is enabled.
10. Update PO visibility fields that currently expose `has_project_authorization`
   from `authorizing_document = "PA"` if they are meant to mean approved PA.
11. Add UI controls on job details, Accounting queue UI, and downstream editor
    error handling.

## Test Coverage

Backend tests should cover:

* upload accepts PDFs and rejects non-PDFs,
* upload is allowed for `job` claim holders,
* upload is allowed for the assigned job manager,
* upload is allowed for the job's alternate manager,
* upload is allowed for the job's branch manager,
* upload is rejected for other authenticated users,
* `project_authorization_doc` is configured with only `application/pdf` in its
  PocketBase `mimeTypes`,
* hash is calculated server-side,
* duplicate hashes are rejected,
* changing an unreviewed uploaded PDF keeps `pa_reviewed` and `pa_reviewer`
  blank,
* replacing or removing an approved PA document fails until admin revocation,
* Accounting approval succeeds with the current hash,
* Accounting approval fails when the submitted hash is stale,
* Accounting approval fails when the PA has already been approved,
* non-Accounting users cannot approve,
* admin revocation clears only review fields,
* non-admin users cannot revoke,
* enforcement disabled allows existing time bundle and PO save behavior,
* enforcement enabled allows time entry create/update against unapproved PA
  projects,
* enforcement enabled blocks timesheet bundle when bundled entries reference
  unapproved PA projects,
* enforcement enabled blocks purchase order create/update for unapproved PA
  projects through the purchase order hook validation path,
* enforcement enabled blocks expense create/update for unapproved PA projects
  through the expense hook validation path,
* enforcement enabled allows timesheet bundle, PO saves, and expense saves for
  approved PA projects,
* PO/client-PO projects are not incorrectly handled as PA projects.

Fixture updates should be append-only where possible, with new job rows for:

* PA expected but no uploaded PDF,
* PA uploaded and pending review,
* PA uploaded and approved,
* PA uploaded, approved, then stale/changed for hash mismatch tests,
* PO-authorized project to prove non-PA behavior remains unchanged.
