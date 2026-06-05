# Project Authorization Documents

Turbo needs a formal project authorization flow for project jobs. The current
`jobs.authorizing_document` field records the legacy business authorization type
(`Unauthorized`, `PO`, or `PA`), but it does not prove that Accounting has
reviewed a scanned signed PA document. This feature makes reviewed PA documents
the authorization source of truth for project jobs and optionally blocks
downstream project spending and time capture until that proof exists.

## Goals

* Store the scanned PA PDF on the project job record.
* Detect duplicate uploaded PA documents across all jobs.
* Give Accounting a queue of uploaded-but-unreviewed PAs.
* Approve a PA only when Accounting approves the same file they reviewed.
* Prevent timesheet bundling, purchase order saves, and expense saves against
  unapproved PA projects when enforcement is enabled.
* Allow admins to revoke PA approval without deleting the uploaded PDF.

## Authorization Type Deprecation

In this document, "project job" means a non-proposal record in the `jobs`
collection, as determined by Turbo's existing job-number/type logic. Proposal
records are not project jobs and should not be blocked by the PA approval gate.

`jobs.authorizing_document` already identifies the kind of project authorization.
Existing code treats `authorizing_document = "PA"` as project-authorized in PO
visibility queries. That is too weak for this new workflow because it only means
the project is marked as PA-backed; it does not mean a scanned PA exists or has
been reviewed.

This feature deprecates `jobs.authorizing_document` as the authorization source
of truth for projects. The field remains in the schema for existing records and
legacy data readability, but new and edited project records must use
`authorizing_document = "PA"`. `Unauthorized` becomes an implied state: a project
is unauthorized for downstream use when it lacks a reviewed PA, not because a
user selected an `Unauthorized` value.

The new approval state is therefore the source of truth:

* `project_authorization_doc` is populated: a scanned signed PA PDF has been
  uploaded.
* `pa_reviewed` and `pa_reviewer` are populated: Accounting has approved the
  currently uploaded PA file.

Any code currently using `authorizing_document = "PA"` as a proxy for project
authorization should be revisited where actual approval matters. Any code
currently treating `authorizing_document = "PO"` and `client_po` as a separate
authorization path should be updated because client PO data is no longer the
authorization state.

`client_po` and `client_reference_number` are optional project reference fields.
They should remain editable independently of `authorizing_document`; neither
field should be required for PA approval, and neither field should be cleared
merely because the project is PA-authorized.

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

Uniqueness must be enforced with a database-level partial unique index for
non-empty hashes, matching the existing attachment-hash pattern used elsewhere in
Turbo. The upload path should still translate duplicate-hash constraint failures
into a validation error on `project_authorization_doc`.

Admin hash repair is an intentional maintenance exception to the ordinary
server-owned hash flow. Holders of the `admin` claim may use the PA document hash
repair function to recalculate and replace `project_authorization_doc_hash` from
the file already stored on the job, including when the PA is already approved,
without revoking the PA or requiring Accounting to reapprove it. This function is
admin-only and is designed to repair stored metadata so it matches the existing
uploaded PDF; it must not replace, remove, or otherwise change the uploaded PA
document or the approval fields.

## Upload Permissions

The meeting notes say that either the project manager or a `job` claim holder can
upload the PA after the job is created. Later feedback also says the BM may be
the last signer and may upload the signed PA. In the current schema, the project
manager is `jobs.manager`; there is no separate `project_manager` field, and the
branch manager should be represented as `branches.manager`.

Because the current `jobs` collection update rule is scoped to `job` claim
holders, upload should use a dedicated route:

```text
POST /api/jobs/:id/project_authorization_doc
```

The route is the user-facing write path for managers, alternate managers, branch
managers, and `job` claim holders. It should enforce upload permission, PDF-only
files, server-owned hash calculation, duplicate-hash handling, and approval reset
behavior.

The `jobs` hook must also enforce the invariant as a backstop for every save
path. Clients must not be able to set or override
`project_authorization_doc_hash`, approval fields must remain server-owned, and
an approved PA document must not be deleted, cleared, replaced, or otherwise
changed until admin revocation clears `pa_reviewed` and `pa_reviewer`.
The generic `jobs` PocketBase update rule should also be updated so normal
client-side job edits cannot directly change `project_authorization_doc`,
`project_authorization_doc_hash`, `pa_reviewer`, or `pa_reviewed`. Document
changes should go through the dedicated upload route, approval fields should go
through the Accounting approval endpoint, and revocation should go through the
admin revocation endpoint. PocketBase rules and hooks should both defend this:
rules keep ordinary client updates narrow, while hooks preserve the invariant for
any save path.

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
* `project_authorization_doc` is populated,
* `project_authorization_doc_hash` is populated,
* `pa_reviewed` is blank, and
* `pa_reviewer` is blank.

The queue should include enough context for review:

* job id, number, description, client, manager, branch, status,
* uploaded file URL or preview action,
* `project_authorization_doc_hash`.

The browser must receive the hash for the file being reviewed. Approval then
submits that exact hash to the server.

## Missing And Incomplete PA Queue

Turbo should also surface active project jobs whose PA document state is missing
or incomplete so the rollout backlog can be worked before enforcement is enabled.
This queue is separate from the Accounting approval queue: it is for operational
follow-up and upload, not Accounting approval.

Add a missing/incomplete queue endpoint:

```text
GET /api/jobs/project_authorization/missing
```

Query parameters:

* `priority`: one of `in_use`, `recent`, `dormant`, or `all`; default `in_use`.
* `page`: positive page number; default `1`.
* `limit`: positive page size; default `50`, maximum `200`.

The queue lists project jobs where:

* `status = "Active"`,
* the job is not a proposal record,
* either `project_authorization_doc` is blank or
  `project_authorization_doc_hash` is blank.

Visibility is scoped by role:

* users with the `accounting` claim can see all missing/incomplete PA jobs,
* users with the `job` claim can see all missing/incomplete PA jobs,
* other users can see only jobs where they are the job manager, alternate
  manager, or related branch manager.

The response should include enough context for prioritization and upload:

* job id, number, description, client, manager, branch, status,
* whether the row is missing the PDF or only missing the stored hash,
* time entry, purchase order, active purchase order, and expense counts,
* latest downstream activity date,
* priority segment,
* whether the caller can upload the PA from the queue,
* for Accounting users, the pending Accounting review count so the UI can badge
  the pending-review tab before loading all pending-review rows.

Priority segments are:

* `in_use`: jobs with at least one time entry, purchase order, or expense,
* `recent`: jobs without downstream activity that were recently created,
  updated, or awarded,
* `dormant`: all other missing/incomplete active project jobs,
* `all`: all visible rows across the three segments.

The default queue segment should be `in_use` because those jobs are most likely
to block real work once enforcement is enabled. Sorting should keep the most
urgent work at the top: `in_use` by latest downstream activity descending,
`recent` by recent award/update date descending, `dormant` by oldest award/update
date ascending, then job number as a stable tie-breaker.

The missing/incomplete queue should support direct upload for callers who have PA
upload permission. Users without upload permission should still be able to open
the job details page for context when the row is visible to them.

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
that make project work usable downstream. The gate does not apply to raw
`time_entries` save.

When `jobs.enforce_project_authorization = false`, none of the gates in this
section should block timesheet bundling, purchase order saves, or expense saves.

### Time

Turbo should allow users to create and edit time entries even when the
referenced project job does not yet have a reviewed PA. This gives the project
team time to upload and review the PA before payroll/accounting workflow reaches
the timesheet bundle step.

Timesheet bundle must fail atomically if any time entry being bundled references
a project job without a reviewed PA. No partial bundle should be created in that
case, and the existing time entries should remain unbundled.

### Purchase Orders

Turbo must reject purchase order save on both create and update when the PO
references a project job without a reviewed PA. This closes the create/update
loophole where a PO could be created jobless or against a different job and then
moved onto an unapproved project.

Implement this in the purchase order hook validation path so all normal PO save
paths receive the same field-level error, not only one HTTP route or UI flow.

### Expenses

Turbo must reject expense save on both create and update when the expense
references a project job without a reviewed PA.

This direct expense gate is required even though many job expenses already
require a PO, because current Turbo expense rules allow some job expenses
without a PO for exempt payment types such as mileage, fuel card, personal
reimbursement, and allowance. Those exempt expense types must also be blocked
until the referenced project is reviewed.

Implement this in the expense hook validation path so all normal expense save
paths receive the same field-level error, not only one HTTP route or UI flow.

### Approval Predicate

A project is PA-approved only when all of the following are true:

* the referenced project job exists,
* `jobs.project_authorization_doc` is populated,
* `jobs.project_authorization_doc_hash` is populated,
* `jobs.pa_reviewed` is populated, and
* `jobs.pa_reviewer` is populated.

This gate applies to projects regardless of the legacy `authorizing_document`
value. Jobs marked `Unauthorized` or `PO` should not be treated as PA-approved
merely because PA enforcement is disabled or because a `client_po` value exists.

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

* non-project records: no PA review controls,
* PA expected, no file: "PA document missing",
* PA uploaded, pending review: "PA pending Accounting approval",
* PA approved: reviewer and reviewed timestamp,
* PA revoked: same as uploaded pending review; prior approval history is not
  retained.

Users with upload permission should see a PDF upload control when
the job is a project. Admins should see a revoke action only when the PA is
currently approved.

The job editor should keep surfacing the legacy `authorizing_document` value for
existing records so old data remains understandable. For new project records,
the UI should hide `authorizing_document`, and the backend clean function should
write `authorizing_document = "PA"`. When editing existing project records, the
UI may submit the existing legacy value, but the backend must fail loudly unless
the submitted value is `PA`. Users must manually select `PA` before saving a
project that still has `authorizing_document = "PO"` or
`authorizing_document = "Unauthorized"`.

The `client_po` and `client_reference_number` fields should always be visible
and editable in the job editor. Job details should show either field when it is
present.

### Accounting Queue

Accounting users should get a pending-review tab filtered to uploaded PA
documents awaiting Accounting approval. The row action should place the "Open
PDF" control next to the approval action because Accounting is expected to read
the uploaded PDF before approving it.

The UI does not need to enforce that the PDF was opened in the current browser
session. Instead, clicking Approve should open a confirmation modal that warns
the Accounting user what approval means: they are confirming they reviewed the
uploaded PA PDF, accept it as the authorization document for the job, and agree
it meets the business criteria required to allow billing and related job activity
against the project.

### Missing And Incomplete Queue

The project authorization page should be a shared PA work hub rather than only an
Accounting queue. It should default to the Missing / Incomplete PDF tab and the
`in_use` priority segment. The missing/incomplete tab should show sub-tabs for
In Use, Recent, Dormant, and All, with counts for each segment and pagination for
large backlogs.

Rows should show the job, client, manager, branch, PA state, downstream usage,
latest activity, and a primary action. Users with PA upload permission should be
able to upload the PDF directly from the queue. Users without upload permission
should be routed to the job details page for context.

The navigation entry should be labelled "Project Authorizations". Accounting
users should always see it, and their badge should combine missing/incomplete
rows with pending Accounting review rows. Non-accounting users should see the
navigation entry only when they have a scoped missing/incomplete count greater
than zero.

### Downstream Surfaces

The PO editor, expense editor, and timesheet bundle flow should surface the
backend error without reducing it to a generic save failure. The message should
name the project manager when the relevant API payload already includes that
information; otherwise use the generic "project's manager" wording.

## Migration And Backfill

Initial migration should add the `branches.manager` field, the four `jobs`
fields, and the hash uniqueness guard. Existing projects should remain
unapproved until a scanned signed PA PDF is uploaded and Accounting approves it.

Do not backfill `pa_reviewed` based only on `authorizing_document = "PA"` or a
populated `client_po`. That would recreate the current weak proxy and defeat the
purpose of the review workflow.

This is intentional. The rollout review period happens while
`jobs.enforce_project_authorization` is `false`. The flag should remain off
until the existing PA backlog has been uploaded and reviewed, because enabling
the flag will block purchase order saves, expense saves, and timesheet bundling
for still unreviewed PA projects.

The previous open question about whether PA approval replaces the PO/PA
distinction is resolved by this plan: PA review is the authorization gate for
projects, and `client_po` is an optional reference field rather than an
alternate approval path.

## Implementation Checklist

1. Add a PocketBase migration for `branches.manager`, the four `jobs` fields,
   and hash uniqueness.
2. Add a dedicated PA document upload route, server-owned PDF upload handling,
   PDF-only PocketBase MIME restriction, and hash calculation.
3. Allow upload for `job` claim holders, the assigned job manager, the alternate
   manager, and the related branch manager.
4. Update the generic `jobs` PocketBase update rule so ordinary client job edits
   cannot directly mutate PA document, hash, reviewer, or reviewed fields.
5. Enforce PA document invariants in the `jobs` hook, including server-owned
   hash/review fields and no approved-document delete, clear, replace, or update
   before admin revocation.
6. Add the Accounting approval queue endpoint/query.
7. Add the missing/incomplete PA queue endpoint with priority segments,
   pagination, scoped visibility, and upload-permission flags.
8. Add the Accounting approval endpoint with hash compare-and-set semantics.
9. Add the admin revocation endpoint.
10. Add `jobs.enforce_project_authorization` config lookup with default `false`.
11. Gate timesheet bundle, purchase order create/update, and expense
   create/update when the flag is enabled.
12. Deprecate project editing of `authorizing_document`: hide it for new project
   records and have the backend write `PA`; for existing project records, surface
   the legacy value and fail loudly unless the user manually selects `PA`.
13. Stop requiring `client_po` when `authorizing_document = "PO"` and stop
   clearing `client_po` when `authorizing_document != "PO"`.
14. Show `client_po` and `client_reference_number` in the job editor at all
   times, and show them on job details when present.
15. Update PO visibility fields that currently expose `has_project_authorization`
   from `authorizing_document = "PA"` if they are meant to mean approved PA.
16. Add UI controls on job details, the Project Authorizations hub, Accounting
    review, and downstream editor error handling.

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
* generic client-side `jobs` updates cannot directly mutate PA document, hash,
  reviewer, or reviewed fields,
* hash is calculated server-side,
* duplicate hashes are rejected,
* changing an unreviewed uploaded PDF keeps `pa_reviewed` and `pa_reviewer`
  blank,
* replacing or removing an approved PA document fails until admin revocation,
* Accounting approval succeeds with the current hash,
* Accounting approval fails when the submitted hash is stale,
* Accounting approval fails when the PA has already been approved,
* non-Accounting users cannot approve,
* missing/incomplete queue defaults to the `in_use` priority,
* missing/incomplete queue segments active project jobs into `in_use`, `recent`,
  and `dormant`,
* missing/incomplete queue excludes closed jobs and proposal records,
* missing/incomplete queue paginates and clamps out-of-range pages,
* missing/incomplete queue visibility is broad for Accounting and `job` claim
  holders but scoped for managers, alternate managers, and branch managers,
* missing/incomplete queue returns the pending Accounting review count for
  Accounting users and zero for non-Accounting users,
* navigation badges include missing/incomplete PA counts and pending Accounting
  review counts according to the caller's visibility,
* admin revocation clears only review fields,
* non-admin users cannot revoke,
* new project records write `authorizing_document = "PA"` from backend cleaning
  even though the UI hides the field,
* editing an existing project fails loudly when `authorizing_document = "PO"` or
  `authorizing_document = "Unauthorized"` is submitted,
* `client_po` remains optional when `authorizing_document = "PO"`,
* `client_po` is preserved when `authorizing_document != "PO"`,
* `client_reference_number` remains optional,
* enforcement disabled allows existing time bundle, PO save, and expense save
  behavior,
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
* legacy PO/client-PO values do not bypass PA approval.

Fixture updates should be append-only where possible, with new job rows for:

* PA expected but no uploaded PDF,
* PA uploaded and pending review,
* PA uploaded and approved,
* PA uploaded, approved, then stale/changed for hash mismatch tests,
* legacy PO/client-PO project to prove legacy values do not bypass PA approval.
