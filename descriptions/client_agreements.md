# Client Agreements

* STATUS: UNIMPLEMENTED, EARLY DRAFT, NOT A BUILDABLE SPEC *

Turbo will eventually need to store client agreement documents associated with
jobs. These documents should not inherit the broad visibility that ordinary job
details currently have, so the agreement file should live in its own
`client_agreements` collection rather than directly on `jobs`.

This document records the current intended shape and the unresolved policy
questions. It is not yet specific enough to implement safely.

## Goals

* Store client agreement documents independently from the job record.
* Associate each agreement document with exactly one job.
* Record who uploaded the agreement and when they uploaded it.
* Keep agreement visibility narrower than general job visibility.
* Define a future upload path that can allow job managers and other approved
  operational roles without exposing the document to everyone who can view jobs.

## Proposed Data Model

Add a new base collection named `client_agreements`.

| Field              | Type                 | Notes                                                                                                                                                    |
|--------------------|----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------|
| `id`               | PocketBase record id | Standard record identifier.                                                                                                                              |
| `client_agreement` | file                 | The uploaded client agreement document. This should be `maxSelect: 1`. Allowed file types and max size still need to be defined.                         |
| `job`              | relation to `jobs`   | Required. Identifies the job this client agreement belongs to.                                                                                           |
| `uploader`         | relation to `users`  | Required. Stores the uid of the user who uploaded the document. A relation is preferred over free text so user identity stays queryable and enforceable. |
| `uploaded`         | datetime             | Required. Server-owned timestamp for the upload event.                                                                                                   |

PocketBase will also have standard `created` and `updated` timestamps. The
business timestamp for the upload should be `uploaded`, not client-submitted
`created`.

Direct API access to `client_agreements` should be narrow. The generic
collection API should not become a general file cabinet where any authenticated
job viewer can list, view, create, update, or download agreement documents.

## Relationship To Jobs

The current assumption is that a client agreement belongs to one job through
`client_agreements.job`.

It is not yet decided whether one job can have multiple client agreements. That
decision changes the model and UI:

* If a job can have only one client agreement, `client_agreements.job` should be
  unique and the UI can show a single current agreement.
* If a job can have multiple client agreements, the spec must define whether the
  records are versions, separate agreement types, replacement history, or
  independent documents.
* If multiple records are allowed, the spec must define how Turbo identifies the
  active/current agreement, whether older agreements remain visible, and whether
  replacement requires explicit audit history.

Until this is decided, no uniqueness index or job details UI should be treated
as final.

## Upload Permissions

At minimum, upload should be available to:

* the job's `manager`, and
* the job's `alternate_manager`.

Additional likely uploaders include:

* the division manager,
* the branch manager, and
* project admin users.

These are not yet defined well enough for implementation. Before this becomes a
real spec, Turbo needs exact definitions for each role:

* **Job manager**: probably `jobs.manager`.
* **Alternate manager**: probably `jobs.alternate_manager`.
* **Branch manager**: likely the manager of the job's branch, but this depends
  on whether Turbo has or will add a canonical `branches.manager` field.
* **Division manager**: undefined. The spec must say which division determines
  the manager when a job has multiple allocations or no obvious primary
  division.
* **Project admin**: undefined. The spec must say whether this is an existing
  claim, a new claim, a branch/division-scoped role, or a per-job assignment.

Upload permission is separate from visibility. A user may be allowed to upload a
client agreement without necessarily being allowed to view every agreement
document for every job.

## File Visibility

Client agreement documents are not for everyone who can view a job. The final
spec must define the visibility matrix before implementation.

At minimum, the visibility rules must answer:

* Can the job manager view the uploaded agreement?
* Can the alternate manager view it?
* Can branch managers view agreements only for jobs in their branch?
* Can division managers view agreements only for jobs in divisions they manage?
* Can project admins view all agreements, or only agreements for scoped jobs?
* Can Accounting, Payables, Business Development, or Admin users view these
  documents?
* Can the original uploader view the document after upload if they no longer
  satisfy the job-based role?
* Can ordinary authenticated job viewers see that an agreement exists without
  being able to download it?

Because file visibility is narrower than job visibility, agreement downloads
should probably use an authenticated route rather than direct PocketBase file
URLs:

```text
GET /api/jobs/:id/client_agreements/:agreement_id/file
```

That route would load the job and agreement, apply the final visibility rule,
and only then stream or redirect to the stored file.

## Candidate API Shape

The exact endpoints are not final, but implementation should probably avoid
direct generic collection writes from the UI.

Candidate write route:

```text
POST /api/jobs/:id/client_agreements
```

Expected behavior:

1. Require an authenticated user.
2. Load the job.
3. Confirm the caller has upload permission for that job.
4. Validate the uploaded file type, size, and single-file requirement.
5. Create the `client_agreements` record.
6. Set `job` from the URL, not from client-submitted form data.
7. Set `uploader` to the authenticated user's uid.
8. Set `uploaded` to the current server timestamp.

Candidate list route:

```text
GET /api/jobs/:id/client_agreements
```

The list route must apply the same agreement visibility policy. If ordinary job
viewers need an existence indicator, that should be a separate, explicit
decision rather than a side effect of this route.

## Lifecycle Questions

The final spec must decide:

* Can an agreement be replaced?
* Can an agreement be deleted?
* Who can replace or delete an agreement?
* Does replacement create a new row or mutate the existing row?
* If the file is replaced, are `uploader` and `uploaded` overwritten?
* Is there an approval/review step after upload?
* Is there a required agreement before a job can become active, receive time, or
  receive purchase orders?
* Are duplicate agreement files allowed across jobs?
* Should Turbo calculate and store a file hash for duplicate detection,
  auditing, or replacement safety?

## Required Decisions Before Implementation

This draft needs the following decisions before it becomes a real spec:

1. Decide whether a job may have zero, one, or many client agreements.
2. Define the agreement lifecycle: create-only, replaceable, deletable,
   versioned, reviewed, or immutable.
3. Define the upload roles in terms of existing schema fields, existing claims,
   or new schema/claim work.
4. Define the visibility matrix for upload, list, details, and file download.
5. Decide whether there is a visible existence/status indicator for job viewers
   who cannot download the agreement.
6. Define file constraints: allowed MIME types, max size, file count, and whether
   PDF-only is required.
7. Decide whether agreement files need server-owned hashes and duplicate-file
   handling.
8. Define whether client agreements affect downstream job workflows.
9. Define UI placement on job details/edit pages and any dedicated review queue.
10. Define the migration, PocketBase rules, routes, hooks, and tests once the
    policy choices above are settled.

## Testing Requirements For The Real Spec

Once the policy is decided, implementation should include fixture-backed tests
for at least:

* allowed upload by the job manager;
* allowed upload by the alternate manager;
* rejected upload by an authenticated user who can view the job but lacks upload
  permission;
* rejected direct generic collection access when it would bypass route rules;
* allowed and rejected file download according to the final visibility matrix;
* one-agreement or multiple-agreement behavior, depending on the chosen
  cardinality;
* file validation failures for disallowed type, missing file, and oversized file;
* server ownership of `job`, `uploader`, and `uploaded`.

Use CSV fixtures for persistent test data where possible, and update
`datapackage.json` if new fixture files or fields are added.
