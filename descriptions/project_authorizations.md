# Project Authorization: Approval And Job Creation

**Status: This work ships in stages. PA1 (client setup and removal of the
legacy workflow) is the first stage; the request and approval workflow follows.
Sections 1 to 6 describe the complete design. Section 7 records what each stage
delivers and where a deployed stage still differs from that design. On September
30, 2026, the allocation permission requirement was clarified: project managers
and alternate managers retain allocation editing without `job`, through the
existing job editor and save route. Production deployment and flag activation
are pending.**

This document records the implementation requirements. Section 1 describes the
system before this change. Section 2 records confirmed requirements. Sections 3
to 5 define the design and acceptance checks. Section 6 gives operating and
release instructions. Section 7 describes the release stages.

## 1. Where We Are Now

Turbo lets holders of the `job` claim create numbered projects and proposals.
Creation validates the job and its division allocations. The job-update API also
permits the manager or alternate manager to edit, subject to field restrictions.
This permission dates from November 11, 2025 (commit `9afe3096`). The new edit
policy below replaces that access for project fields other than division
allocations. Managers and alternate managers retain allocation editing.

`job_owner` is currently an optional relation to `clients`. The project forms
default to `Active`, including subjobs and projects created from proposals.
Users can select `Active`, `Closed`, or `Cancelled`. The editor recommends
`Closed` for parent jobs used as reporting containers.

The code includes a PDF Project Authorization workflow: upload, review, rejection,
revocation, queues, notifications, and document hash checks. Its setting,
`jobs.enforce_project_authorization`, controls PDF approval checks on downstream
work. The last recorded production value is false. Verify it before deployment.

The existing PO authority model uses the `po_approver` claim and a related
`po_approver_props` record. Project authorization will use the same claim and
properties pattern, with its own claim and one value limit. It will not use the
PO division, expenditure-kind, threshold, or approval-stage rules.

Clients have no structured postal address. Contacts have email but no phone
field. There is no reusable client invoicing profile or job-request table.

## 2. Confirmed Requirements

### Approval, Then Creation

An employee prepares an unnumbered job request with structured project data and
one PDF. Turbo shows missing or invalid information. The workflow has two steps:

1. **Approval:** A qualified holder of `project_authorization` reviews the request
   and approves its PDF and project value. This step does not create a job.
2. **Creation:** A holder of `job` checks the approved request and creates the job
   in a transaction. Turbo assigns the number, transfers the allocations and
   approval records, and deletes the request.

One person can complete both steps if that person has both claims and sufficient
approval authority. Keep the actions, identities, and times separate even when
the same person performs both. A `job` claim alone does not authorize a project.

There are no separate client, contact, or invoicing-profile approvals. A request
cannot receive time, work records, purchase orders, or expenses. Existing jobs
remain usable without a legacy marker or approval backfill.

The new workflow retains PDFs as authorization evidence. It replaces the old
single-PDF workflow and its downstream work restrictions.

### Printable Project Request

Provide a printable document for every Project Request, with signature spaces
for the client and the TBT representative. Sarah requested this on September 24
and confirmed this scope on September 25, 2026. She cited the existing ability
to print purchase orders after showing it to Leanna and Felicia. Use PO printing
as the reference for the print action and user flow.
The current reference is `ui/src/routes/pos/[poid]/print/+page.svelte` and its
`+page.ts` loader; adapt its access and state checks to requests.

Staff must be able to generate and print the document before request approval
so they can obtain the client's signature. Printing must not require an uploaded
authorization PDF or an assigned job number. Use the existing request-view
permissions to control access. Printing does not approve the request or create
a job; the separate PDF submission and approval requirements still apply.

#### First-Version Content And Layout

Use the following defaults for the first implementation. The document is for
the client's agreement to the work and fees. Use existing request and related
record data; do not require staff to enter the same information again.

Title the document **Project Authorization**. Use the TBT Engineering Limited
name and logo from the PO print page. Default to Letter paper in portrait,
black text on white, with clear section headings and generous signature space.
Aim for one main page for a short request. Let long content continue onto more
pages without truncation or reduced text size. Keep each signature block
together. Use this section order:

| Section | Content and defaults |
| --- | --- |
| Document reference | Request ID, request version, and date generated. Label the ID **Request reference**, not project number. Show **Project number: To be assigned**. Repeat the request reference on continuation pages. |
| Client | Client name and available postal address. Show the project owner's name separately when set and different from the client. The client remains the party asked to sign. |
| Project and scope | Full project description under **Project / scope of work**. Use the description as the project name as well; add no separate name field. Show the location, including the stored Plus Code. Preserve paragraph breaks. |
| References | Linked proposal number and description when present; client PO and client reference number when supplied. For a subjob, show the parent project number and description. Omit unused optional references. |
| Project contacts | Client project contact name, email, and available phone. Show the TBT project manager's name, available business email/phone, and branch name. Do not substitute the internal approver as the TBT contact or signatory. |
| Fees | Prominent **Project value (CAD, before tax)** with thousands separators and whole dollars. Show **Fee basis: Fixed fee** when T&M is false, or **Fee basis: Time and materials** when true. State **Applicable taxes are additional.** Identify the selected rate sheet by name, revision, and effective date. |
| Invoicing | **Bill to** client name; billing contact name, email, available phone, and billing fax when supplied. Use the billing contact's address when present, otherwise the client's address. Use one address source; do not combine partial addresses from different records. Show the invoicing instructions verbatim when present. Always show this section, even when the project and billing contacts are the same. |
| Agreement and signatures | The agreement text and two signature blocks defined below. |

For T&M work, append the selected rate sheet's client billing rates, using the
existing rate-sheet print layout: role, regular hourly rate, and overtime hourly
rate, in CAD. Include its name, revision, and effective date on the appendix.
Generate the main document and appendix together so the client can read the
rates before signing. Do not print internal labour costs or profit calculations.
Keep **Project value** as the amount label for T&M; do not relabel it as a fixed
fee, estimate, or spending cap. Do not infer payment deadlines, expense markups,
or other commercial terms from the rate sheet or PO template.

Do not copy the PO's statement that the document itself confirms authorization.
Use this agreement text immediately above the signatures:

> The client authorizes TBT Engineering Limited to perform the work described
> above on the stated fee basis and agrees to the stated project value and
> invoicing details. For time and materials work, the attached rate sheet
> applies. Applicable taxes are additional.

Omit the T&M sentence for fixed-fee work. A proposal reference identifies the
related proposal; the generated form does not add unseen terms or claim that
the proposal text is included. Each signature block has blank lines for printed
name, title, signature, and date. Label them **For the client: {client name}**
and **For TBT Engineering Limited**. Leave signatory names blank because the
project contacts need not be the people authorized to sign. These are spaces
for signatures on the document, not electronic signature controls in Turbo.

#### Print Action, Incomplete Requests, And Signed Copy

Provide **Print Project Authorization** on the request detail page. Follow the
PO print flow: open a dedicated print view, load the saved data, and open the
browser print dialog when rendering is complete. Support printing and Save as
PDF, with a screen-only fallback instruction for the browser print command.
If the form has unsaved edits, require staff to save them or cancel printing. Use a
descriptive document title such as **Project Authorization - {request ID}**.
Unlike the existing PO print route, do not require an approved or Active record.

Every request remains printable. When client name, description, location,
project contact name/email, billing contact name/email, positive project value,
explicit fee basis, or selected rate sheet is missing or invalid, mark the
output **DRAFT - NOT FOR SIGNATURE**, list the missing items, and omit signature
lines. For T&M, unavailable billing rates also make the output a draft. Show
**Not provided** for missing required values, never zero as a substitute for
an unknown value. Omit absent optional addresses, phone numbers, and references;
they do not prevent a document from being ready for signature. A complete
client document can be ready for signature before internal submission checks,
PDF upload, approver selection, or approval are complete.

Exclude internal allocations, approval limits, checklist answers, rejection
reasons, audit history, and workflow status from the client document. A request
version and generation date identify the output, but do not certify approval.

Staff obtain the client and TBT signatures, then upload the signed document as
the request's existing authorization PDF. For T&M, include the rate appendix in
that single PDF. Generating or saving an unsigned printout does not attach it
automatically or mark it signed or approved. Keep support for other sufficient
authorization PDFs; this template is a convenience, not a new mandatory format.
The approver checks that the uploaded evidence covers the current work and
value through the existing PDF-sufficiency check. If the agreed work or fees
change, staff obtain updated evidence before approval. No signature recognition
or new signature-status field is required in this version.

Render each printout from one consistent read of the saved request and its
related records. Later contact or invoicing edits can affect a new printout;
they must not rewrite an uploaded signed PDF. Keep the signed PDF as the
retained evidence under the existing attachment rules. Printing is read-only:
it must not change the request version, last-edited time, or workflow state.

### Authority Matrix And Project Value

Add the `project_authorization` claim. Each user assignment has one associated
`max_value`, expressed in whole CAD dollars. There are no branch, division, or
project-type limits. An approver must be active and meet all of these conditions:

* The user has the `project_authorization` claim.
* The claim assignment has a valid, positive `max_value`.
* `max_value` is greater than or equal to the full proposed project value.

A missing claim, missing properties record, or zero limit grants no authority.
An exact match between the limit and the project value permits approval. A
manager assignment, `job`, or `admin` alone does not grant approval authority.
Use the same server policy for approver selection, field validation, and
approval. Validate authority when the selected approver is written and whenever
a proposed project value changes. Check it again at submission and approval.
A user who loses the claim or a sufficient limit after assignment cannot approve.
Do not rely on an earlier dropdown result. Only the selected user can approve.

Project value is the amount, in CAD and before tax, charged to the customer.
It must be greater than zero, including for Time and Materials (T&M) projects.
Use whole dollars for request values, proposed and approved job values, stored
previous values, authority limits, and recorded approval limits. Keep the
existing whole-dollar job field. Reject fractional amounts on the server;
do not round or truncate them. Forms must use whole-dollar inputs.
T&M does not exempt a project from value-based approval. Project value, the T&M
indicator, and a rate sheet represent the required fee arrangements.

Project value is revenue. The difference between this value and time costs,
expense costs, and amortized corporate overhead is the project's profit margin.
This specification does not add a profit calculation or approval test.

Approval always covers the full new value, not only the change. For example, a
change from $80,000 to $120,000 requires a limit of at least $120,000.

### Permissions

A claim is a permission assigned to a user. Workflow actions require an active
user as well as the applicable claim or ownership rule.

| Action | Permission |
| --- | --- |
| Create and submit a request | Employees with `time`. |
| Use Create Subjob on a parent job to start a request | Anyone authorized to create a job request, when the request workflow is enabled. No `job` claim is required. |
| View requests | Creators can view their own requests. Holders of `job` can view all requests. The `project_authorization` claim grants access only to assigned requests and their PDFs. Admins can view requests in the abandoned-request list for triage. |
| Edit or recall a request before job creation | Its creator. Recall a submitted or approved request before editing it. A rejected request can be edited directly. |
| Select or change a request's approver | Its creator only, under the same edit-state rules. |
| Approve a request | Its selected approver, with `project_authorization` and a limit that covers the full value. |
| Approve one's own request | Allowed when the creator is the selected approver and meets the same authority checks. |
| Reject a request before job creation | Its selected approver only, with a reason. |
| Create a job from an approved request | A holder of `job`. |
| View the Job Create Queue and its badge | Active holders of `job`, when the request workflow is enabled. |
| Create an invoicing profile using an existing client contact | Active holders of `time` or `job`. No separate profile approval is required. |
| Edit existing invoicing profiles, or maintain clients and contacts | Holders of `job`. A profile creator without `job` cannot edit it. |
| Add or remove division allocations, or change allocated hours on an existing job | Holders of `job`, or the user assigned to `jobs.manager` or `jobs.alternate_manager` on that job. Managers and alternate managers do not need `job` for these allocation edits. Existing allocation validation applies. |
| Edit other job fields, including proposing a new project value and uploading its PDF | Holders of `job`, subject to validation and additional field permissions. Client is immutable except through the controlled merge process. |
| Select or change the approver for a pending project-value change | Holders of `job`. |
| Approve or reject a pending project-value change | Its selected approver. Approval requires `project_authorization` and a limit that covers the full proposed value. Rejection requires a reason. |
| Delete the latest approved project-value change or clear a pending approval for admin correction | Holders of `admin`, subject to the deletion rules below. |
| Delete an abandoned request | Holders of `admin`. |
| Create proposals | Holders of `job`, subject to existing proposal rules. |

The approver's claim does not grant general request access or permission to edit
a job. Other claims and request ownership retain the access stated above.
Only the selected approver can approve, even when another user has a sufficient
limit. A person with both `job` and sufficient `project_authorization` authority
can select themselves for a project-value change and approve it.

Keep additional field permissions, such as `rate_sheet_revise`, unless an
explicit decision changes them. A `job` holder cannot bypass value approval by
writing the proposed value directly to `jobs.project_value`.

Preserve the existing manager allocation workflow in the UI and on the server.
Before this feature, the job editor let managers and alternate managers add or
remove divisions and change hours. Its save endpoint authorized those users and
replaced the allocation rows in a transaction. This was supported UI behavior,
not merely access through the generic collection API.

The new restrictions on project fields must not remove that allocation access.
Provide an allocation-only save for a manager or alternate manager without
`job`. Such a save must leave other job fields unchanged and must not move an
allocation to another job. Reject attempts to combine allocation edits with
changes that require `job`. Keep the UI controls and server permissions aligned;
an API permission with no usable UI path does not meet this requirement.
Allocation edits require no new authorization PDF or project-value proposal.
The job details page shows **Edit allocations** for these managers. The existing
editor shows only division allocations and sends `PUT /api/jobs/{id}` with
`allocations` and no `job` object. An empty `job` object remains a full job save
and retains the existing project validation. A save that changes allocations
updates the job's writeback marker and timestamp, but no other job fields.
A save with no allocation changes preserves the existing rows and review state.
Generic allocation create, update, and delete APIs are locked for application
users. The editor keeps generic read access and saves through the job route.

### Request States And Handling

Use the same action names as expenses and timesheets. **Recall** is the creator's
action to take a submitted or approved request back for editing. **Reject** is
the selected approver's action to decline it with a reason. Both actions are
available before job creation.

The normal flow is:

`Draft -> Awaiting approval -> Awaiting creation -> Job created`

`Job created` is the result of creation, not a retained request row. Also support
`Rejected` and `Recalled`. Drafts can be incomplete. Submission requires complete,
valid data and a selected, qualified approver.

Only the creator can recall the request, edit it, or select a different
approver. Only the selected approver can reject it. Recall and rejection apply
in `Awaiting approval` and `Awaiting creation`. They clear any current request
approval and prevent creation until a fresh approval occurs.

A rejected request cannot be submitted unchanged. A successful creator edit
clears `rejected`, `rejector`, and `rejection_reason` on the server and moves the
request back to `Draft`. Clear any corresponding attachment rejection fields
in the same transaction. Failed edits must leave the rejection intact. Merely
pressing Submit, sending an unchanged save, or recalling a rejected request must
not bypass the required edit. Recheck completeness and authority on submission.

Previous approvals on a request do not need to be retained. Clear their actual
approving user, approval time, recorded limit, reviewed version, and checklist
data when recall or rejection invalidates approval. The existing request PDF can
remain attached for correction and fresh approval; do not require a duplicate
upload of that same PDF. Only the final approval used for job creation must
survive request deletion.

The selected approver receives submission, rejection, and approval notices.

### Job Create Queue

Provide a **Job Create Queue** for active holders of `job`. Show all approved
requests in `Awaiting creation`, including subjob requests. Do not restrict the
queue to the holder's own requests or approver assignments. Include a navigation
badge that counts these requests. Use the same server selection rule for the
queue and badge; the count covers all matching requests, not only the current page.

Each entry opens the approved request and its PDF for review and provides a
**Create Job** action. Require the duplicate-work check and all validation and
transaction rules in the Job Creation section. Queue membership does not bypass
these checks. If creation fails, retain the request and show the error.

Successful approval adds the request to the queue. Successful creation, recall,
rejection, or admin deletion removes it. Refresh the queue and badge together
after these actions. Age alone does not remove an awaiting-creation request.
When the request flag is off, hide the queue entry point and badge and prevent
creation through its action, while retaining the requests. Restore the queue
and count from retained request state when the flag is enabled again.

### Abandoned-Request List

A request appears in an admin-only abandoned-request list after at least 60 days
without an edit. Store the threshold in the code constant
`JOB_REQUEST_ABANDONMENT_DAYS = 60`. Measure it from the last successful edit to
request data, its attachment, or its allocations. Use request creation time
until the first edit.

Abandonment is a query result, not a stored status or an automatic transition.
Include any retained request that meets the age rule, including one awaiting
creation. Reading a request or changing workflow timestamps does not count as
an edit. Use the same age rule for the list and duplicate-file messages.

Admins can review the list, follow up with the creator, or delete requests.
Do not delete automatically or block normal workflow actions merely because
a request appears in the list. A successful creator edit resets its age.
Request ownership does not transfer; if another employee must prepare the work,
that employee creates a new request.

### Approval Checks And Duplicate Work

The PDF is the evidence on which the approver authorizes the project and its
value. Show one required checkbox for each statement below. All boxes start
unchecked; the selected approver must check each one for the current review.
These are the six Boolean answers for request approval checklist version 1:

| Answer key | Checkbox statement |
| --- | --- |
| `fn_agreement_reviewed` | I reviewed the First Nations agreement answer. |
| `no_known_conflict` | I am not aware of a company conflict of interest for this project. |
| `no_executive_restriction` | There is no executive restriction or moratorium that prevents this work. |
| `no_duplicate_work` | I checked existing jobs and this request does not duplicate work already represented by a job. |
| `pdf_sufficient` | The attached PDF is sufficient to authorize this project and its stated value. |
| `allocations_complete` | All necessary division allocations are present. |

The server must require version 1 and all six answers to be Boolean `true`.
Missing, false, or invalid answers prevent approval. An unchecked box does not
automatically reject the request; the selected approver can reject it with a
reason. Do not permit approval while a known conflict or an executive
restriction prevents the required confirmation.

The First Nations checkbox confirms review, not a `Yes` answer to the request's
First Nations question. Retain the checklist version and answers with the
approval. Recall or rejection clears them; a fresh approval requires a fresh
checklist. This checklist applies to request approval. Existing-job value
changes retain the separate approval rules below.

#### Submission Identity And Old Review Pages

`job_requests.submission_token` identifies one submission for approval. It is a
server-generated, cryptographically random string of 32 lower-case letters and
digits. It is not sequential, a secret credential, an approval, or a replacement
for permission checks. Callers cannot supply or edit it. A draft has an empty
token. Each successful submission or resubmission gets a new token in the same
transaction as its status and submission date. Recall and rejection clear it
in the same transaction as the approval and checklist. Approval keeps it until
job creation transfers the final attachment and deletes the request.

This field closes a specific stale-review case. An approver can leave a page
open with six checked boxes while the creator recalls and resubmits the same
data and PDF. Clearing saved approvals does not clear the old browser page.
The request's business version and PDF hash can still match. A new
`submission_token` makes the old approval action invalid even in that case.

The server includes `submission_token` in the `review_token` digest returned
with request details. The browser echoes that digest on approval and creation;
it does not generate a submission token. Approval compares the supplied digest
with the current digest, then stores the accepted digest with the attachment's
approver, time, and checklist. Creation compares both the caller's digest and
that stored approved digest with the current request. Approval and creation
reject requests with an empty submission token. An old digest causes a conflict
response; the user must reload and complete the six boxes again.

Reads, printing, and approval do not rotate `submission_token`. Failed workflow
transactions do not persist a new or cleared token. Token changes alone do not
advance the business-data `version` or reset `last_edited`, so resubmission does
not reset the abandoned-request clock. A timestamp cannot replace this field:
timestamps can tie, and the same business data can be submitted more than once.

Refuse a request that duplicates work already represented by a job. Show recent
jobs for the client and highlight jobs linked to the selected proposal. Show number, description,
client, status, manager, and parent, with links to details. Also show possible
matches by description, location, and client PO/reference number. For subjobs,
show the parent and existing subjobs. The approver compares against existing
jobs, not requests submitted to other approvers. Do not show unassigned requests
or require the approver to investigate other people's pending work.

Identify why each result appears. A possible match is not proof of duplication.
Do not block approval merely because a proposal has another linked job.

Before creation, the `job` holder checks existing jobs again and compares the
request with other pending requests. That holder coordinates overlaps with
the creators and does not create a job that duplicates an existing job. A
possible match in another request requires review, not automatic rejection.
This responsibility does not grant request recall or rejection permissions;
those remain with the creator and selected approver respectively.

### PDF Storage And Duplicate Files

Use one `job_attachments` table for request PDFs and job authorization PDFs.
Compute SHA-256 on the server at upload. Reject an upload if its hash already
exists in that table. Apply one global unique index across all request and job
attachments, including requests in the abandoned list and earlier job approvals.

A request has one current PDF. A job can have several approved PDFs, ordered by
approval sequence. Each approved attachment records its value, approver, and
times. Approvals transferred to jobs and later approved job attachments are
immutable except for explicit admin deletion. Request approvals can be cleared
through recall or rejection before creation, as described above.

During creation, transfer the request attachment to the job. Retain its ID,
file, hash, and approval data. Do not upload or duplicate it. Deleting the
request must not delete the transferred attachment.

If an abandoned request owns a duplicate PDF, fail the upload and identify that
request in the error. Tell the user to give the request identity to an admin,
who can resolve the conflict by deleting the abandoned request. Do not bypass
the duplicate check or transfer its document automatically.

Example: "This PDF belongs to abandoned job request {id}. Give this request ID
to an admin to resolve the duplicate."

Different PDF files can describe the same work. A unique hash does not replace
the duplicate-work review.

### Required Project And Billing Data

Before submission, require description, client, project contact, manager,
branch, Plus Code location, award date, active rate sheet, positive project
value, T&M indicator, at least one active division allocation, an explicit
First Nations agreement answer (`Yes` or `No`), one PDF, and a qualified selected
approver. Require the project's invoicing profile and validate its relations.
Keep `job_owner` optional, matching the current schema and validation. If set,
it must reference a valid client. Add no new owner-detail requirements.
Projects created from requests default to `Active`. The creator can select
`Closed` when the project will serve as a parent/reporting job; child jobs need
not exist yet. The request-to-job transaction cannot create a `Cancelled` job.
Once the job exists, it can be set to `Cancelled` under the normal job-edit
permissions and validation. Keep the intended project status separate from the
request's workflow status.

Clients and client contacts both have street, city, province/state, postal code,
and country fields. All address fields are optional. Keep the general client
phone field. Every client contact requires an email address, whether used for
project contact, billing contact, both, or neither. Contact phone is optional.

Use `client_contacts` for both project and billing contacts. Each job has a
project `contact` relation and an `invoicing_information` relation. The latter
points to a reusable `client_invoicing_information` profile with a `client`,
one `contact` relation to `client_contacts`, an optional profile name (up to 120
characters), and optional free-form invoicing notes.
The notes can be empty; the profile and contact relations are still required.
Several profiles can belong to the same client. Support a billing fax number.
Do not track HST exemption. Report-delivery information is outside this work;
do not add report-delivery fields or validation.

The job's project contact, invoicing profile, and profile's billing contact must
all belong to the job's client. Apply the same rules to job requests.
"Same as project contact" selects the same `client_contacts` record for the
profile's contact. It does not copy the contact or its address into the profile.
The billing contact's optional address is available through that relation.

Edits to a contact or profile appear wherever that record is referenced. A
change to the job's contact relation selects another record; it does not itself
change the profile's billing-contact relation. Validate contact email and
same-client relations on shared-record edits as well as request/job saves.
Reject changes that would leave existing references with mismatched clients.

Controlled client merges are an exception to the immutable job-client rule.
Use the existing merge permissions and process to consolidate client records.
Update all affected job, request, client-contact, and invoicing-profile relations
together in one transaction, including owner references. Validate same-client
relationships against the final merged state, rather than an intermediate state.
If any part fails, roll back the entire merge.

The merge process can update affected request relations under this exception;
ordinary request edits remain creator-only. Apply request version and
stale-review checks to changed relations. Keep approver assignments and approved PDFs, values,
and approval identities unchanged. Contact merges must also update project and
billing contact references together. Merge undo must preserve these same rules.

### Shared Client Workspace

Each client has a workspace with three tabs: **Client details**, **Jobs**, and
**Notes**. Each tab has its own route under `/clients/{id}`. Reuse the
application's search, list, tab, form, and action components. Keep the
outstanding balance and its date,
business development lead, and existing note permissions and job associations.

**Client details** contains client information and two compact lists: contacts
and invoicing profiles. Profiles store reusable invoice delivery details; the
label leaves invoicing available for future accounts receivable features. Use
one contact editor for all entry points. Creating a client saves only client
information; users then add contacts and invoicing
profiles from the workspace. Each editor has its own save and cancel actions.
Place a question-mark help button beside **Invoicing profiles**. Explain the
recipient, address source, and invoice instructions, where users select a
profile, and how shared edits affect all jobs and requests that use it. State
that a profile is required when creating or editing projects and subjobs, and
before submitting project requests. Draft requests and proposals are exempt.

Filter each complete list as the user types, before pagination. Contact search
covers name, email, and phone. Invoicing search covers profile name, recipient,
and instructions. Display short instructions once as plain text. For long
instructions, show the first few words followed by a clickable ellipsis.
Expanding replaces the preview with the full text and a collapse icon at the
end. Keep keyboard focus on the control when it changes. Keep filters and pages independent. Default to 20 rows per
page, with 10 and 50 also available. Show result counts and an empty state.
Store the list state in the URL so refresh, Back, and editor return preserve it.
After a save, briefly show “Contact saved.” or “Profile saved.” and highlight the
saved row. Show one **Show contact** or **Show profile** link only when a filter,
recipient selection, or page hides that record. The link reveals the exact saved
record and changes only the filters or page needed to show it. Keep other list
state unchanged. Pause message expiry while the user points to it or uses its
link with the keyboard. Clear the return marker so refresh does not repeat the
confirmation.
Allow a contact to filter the invoicing list by recipient.

**Jobs** retains the existing **Projects**, **Proposals**, and **Jobs as owner**
lists, with 20 jobs per page. **Notes** is an open list with search and alternating
row colours. Search covers note text, author, and linked job details. Keep the
shared note form and job links. Use these routes:

| View or action | URL under `/clients/{id}` |
| --- | --- |
| Client details | `/details` |
| Projects | `/jobs` |
| Proposals | `/jobs?view=proposals` |
| Jobs as owner | `/jobs?view=owner` |
| Notes | `/notes` |
| Edit client information | `/details?edit=client` |
| Add or edit a contact | `/contacts/add` or `/contacts/{contactId}/edit` |
| Add or edit an invoicing profile | `/invoicing/add` or `/invoicing/{profileId}/edit` |

Remove the old `/edit` client editor route. Do not add a `/billing` route, and
do not keep redirects or the old `?tab=` links to job and note views. All editor
return links must stay within the same client's workspace. A project-form
return link may target only a supported local job or request form.

Active holders of `time` or `job` can create an invoicing profile using an
existing contact of the same client. Only holders of `job` can edit an existing
profile, including a profile created by the requester, or create and edit client
and contact records. Other authorized viewers can read and select profiles but
cannot create or edit them. Enforce these rules on the server and in the UI;
`admin` alone does not grant profile creation or shared-record editing. Writes
to clients, contacts, and profiles also require an active account. Enforce
these permissions with collection rules, so PocketBase superusers keep
dashboard access for data correction. A profile created by a superuser has no
creator.

Show an Add invoicing profile action to eligible users. If a client has no
profiles, explain that the user must create one using an existing client contact.
On job and request forms, hide the empty profile selector and management link.
Keep the **Invoicing profile** label and show: “This client has no invoicing
profiles. Create one to continue.” Make **Create one** a direct action to the
profile editor, with the unfinished form preserved. If there are no contacts,
show job maintainers an **Add a contact** action to the contact editor; direct
requesters to a job maintainer. When profiles exist, show the selector and
client-management link, with the link beside the label on the job form. Use the
same pattern in value-change corrections. Distinguish loading and failed loads
from an empty profile list, and retain the retry action.

When creating a profile, show existing profiles for the selected recipient and
offer **Use this profile** without writing or changing that record. Requesters
can instead create a different profile when instructions differ. Do not enforce
uniqueness on the client/contact pair. Keep name, instructions, and fax optional.
Only job maintainers can add or edit contacts within the profile editor.

Record the authenticated creator on the server and retain the creation timestamp.
Do not accept a supplied creator or allow later edits to creator attribution.
Show the creator and date in the workspace. Leave the creator unknown for older
records rather than inventing an identity. A new profile is immediately available
for reuse. There is no separate profile approval status or queue: the selected
invoice details remain subject to the normal project request review. Request
approval and numbered job creation remain separate actions with their existing
permissions.

Show how many jobs and requests use a contact or invoicing profile before saving
shared changes. Preserve contact
merge actions and block deletion while references exist. Counts for contacts
include their use as both project contacts and invoice recipients.

When saving changed fields on an existing profile, refresh its usage counts.
If more than one job uses it, offer **Update shared profile**, **Save as new
profile**, and **Cancel**. Show the job count and any request count. Place an
explanation near the buttons that changes with the hovered or focused action.
Updating applies the changes wherever the profile is used. Copying creates a
new profile from the edited fields and leaves the original unchanged. Cancel
returns to editing without losing any changes. Keep the form and choices
available after a failed save; block duplicate submissions while saving.
A failed usage check must prevent a shared update. New profiles, unchanged
fields, and profiles used by zero or one job do not need the shared-update
confirmation. Saving unchanged fields must send no update. If the return page
fails after a successful save, retry only the return without another write. Also provide **Save as new profile** during any existing-profile
edit, regardless of its usage count.

After creating a profile, **Save as new profile**, or **Use this profile**,
return to the originating job or request form and select that profile in the
unfinished form. The user must still save
or submit the form separately. Match the draft's user, client, and return
route before changing its selection. From the client workspace, add the new
profile without moving any jobs. If the originating draft is unavailable,
return to the workspace with the profile save confirmation described above.

An invoicing profile selects an existing client contact. Adding or editing that
contact from the profile uses the same contact editor and preserves the profile's
unfinished fields. Show the invoice recipient and effective address, including
whether the address comes from the contact or the client. A profile name helps
users distinguish profiles that share a recipient; it is optional and does not
change document or approval rules.

Opening client maintenance from a job, request, or value-change form preserves
that form's unsaved values in the current browser tab. Returning refreshes the
available contacts and profiles without changing either selection, except
when creating, copying, or explicitly reusing a profile selects it in the draft.
Preserve the original request and attachment versions. Refresh a value-change
review token after shared client edits only if the job and its saved division
allocations still match the original context. Otherwise require the user to
reopen the value form and review the current job. A selected PDF stays available during
normal navigation; after a browser reload, require the user to select it again.

### Project-Value Changes And Continued Work

Only a change to project value currently requires a new PDF and approval.
This includes a decrease. Other valid job edits do not require a new PDF or
another approval. Holders of `job` can edit the job or upload a new authorization
attachment. The assigned manager and alternate manager also retain the right
to add or remove division allocations and change allocated hours without `job`.
That exception does not permit changes to other project fields or authorization
evidence.

To change project value, a `job` holder uploads a PDF, enters the proposed full
value, and selects the approver. Populate the list with the same authority
policy used for requests. Validate the selected approver when the field is
written and again when approval occurs.

Store one pending approval and proposed value in `job_attachments`, linked
directly to the existing job. Do not create a job request or a separate revision
entity. Keep `jobs.project_value` at its current effective value and show both
the effective and proposed values. Permit only one unapproved value change per
job, including a rejected change awaiting correction or admin deletion.

Only the selected approver can approve the change. Check current authority
against the full proposed value. On approval, record the approval and replace
`jobs.project_value` in one transaction. Approval itself does not require
`job`; entering or editing the proposed change does. No creation step occurs.

The selected approver can reject the pending change with a reason. Rejection
leaves the effective job value intact. Corrections require `job`, clear the
pending record's rejection fields, and require a fresh approval. A rejected
change cannot be approved unchanged. An edit to pending data invalidates any
earlier review; approved job attachments cannot be edited this way.

Show pending value approvals in a queue. The assigned approver must be able to
view the job and PDF evidence needed for that approval. Pending or rejected
changes do not block time, work records, expenses, billing, or purchase orders
against the job. Those actions retain their other validation and permissions.

Apply the same value-change process to existing jobs with no previous
attachment. Keep their current value until approval, without approval backfill.
An older T&M job can have a zero effective value. Validate a pending change
against the proposed positive value, while retaining the historical effective
value until approval. Other job fields must pass their normal validation.
Ordinary job edits must pass the new validation, including a status-only edit
that sets a project to `Active`. An edit that only sets a project to `Closed` or
`Cancelled` keeps the existing status rules and does not require the new
validation, so staff can retire older projects without completing their data.
An `Active` project can receive new time, expenses, and purchase orders, so it
must have complete billing data. The stale and unused job lists show any error
beside the job it affects. The client relation is immutable except through the controlled merge process.

### Admin Deletion And Retained History

An admin can exceptionally delete the latest approved project-value change.
First delete any pending new approval for that job, including an unresolved
rejected change. Then delete the latest approved attachment row, PDF, approval
data, value, and hash. Restore the previous effective job value in the same
database transaction. Do not delete an earlier approval while a later one
remains.

The initial approval used to create a job cannot be deleted because it was
required for that job to exist. For an older job that first acquires an approved
attachment later, an admin can delete that approval and restore the value from
before it. The job then has no approved attachment. Store that previous value;
do not invent an approval or attachment for it.

Refuse deletion of a job while any `job_attachments` row references it, including
an initial approval, a later approval, or a pending or rejected change. Job
deletion must not cascade-delete attachments or clear their job relation.
Admins must use the permitted attachment-deletion process first; the protected
initial approval therefore prevents deletion of a job created from a request.
If no attachments remain, the existing job-deletion permissions and checks
still apply.

Admin restoration can restore the exact stored historical value even when it
is zero or otherwise fails the new positive-value rule. This narrow exception
does not permit an arbitrary replacement value and requires no new PDF or
project-authorization approval. Other validation and admin-deletion guards still
apply. New requests and proposed value changes must still have positive values.

This is deliberate deletion. The admin is responsible for the exceptional
change to history. A successful deletion releases the hash so the same PDF can
be uploaded elsewhere. Do not retain a hash reservation that prevents reuse.
Admin deletion of a pending change also releases its hash and leaves the
effective job value unchanged.

Deleting an abandoned request deletes its attachment and draft allocations
and releases its document hash. Only an admin can perform this cleanup.

Otherwise, retain approvals transferred to jobs and later approved value changes
indefinitely. Preserve requester identity, approval identity and time, creation
identity and time, and checklist version/answers in the creation transaction.
Earlier approvals on an uncreated request do not require history. A full
snapshot of client, contact, invoicing, and other editable fields is not required.

Remove the old PDF references and approval data from the database. Do not archive
or transfer old PDFs or approvals to `job_attachments`. The old S3 objects can
remain orphaned for separate manual cleanup. Existing jobs need no approval
backfill. This removal does not apply to new workflow attachments or approvals.

### Feature Flag

Add `jobs.require_job_request_approval`, with a default of false.

| Flag | Required behavior |
| --- | --- |
| Off | Requests cannot be created or progressed. Holders of `job` create projects directly through the current process. Existing requests remain stored with their current state and age. |
| On | Every new project and subjob must come from an approved request through the creation step. No creation path is exempt, including imports and administrative routes. |

The flag controls project creation. It does not disable approval for value
changes on existing jobs or restrict downstream work. Jobs created while it is
off remain usable when it is enabled. Turning the flag off does not mark
requests abandoned; the admin list continues to use time since the last edit.
Retained requests can progress under their normal rules when the flag is on
again. Do not reuse the old PDF flag.

### Proposals And Subjobs

Proposals remain outside this workflow. Keep their current validation; do not
require project billing fields or project-value approval on a proposal. A
project request can omit a proposal. One proposal can produce several projects;
creating a project does not replace the proposal.

A draft request can select a proposal with status `In Progress`, `Submitted`,
or `Awarded`. Other proposal statuses cannot be selected. A selected proposal
must be `Awarded` at submission, approval, and creation, including resubmission.
If it is no longer awarded, fail the action without creating a job or deleting
the request. Notify the user that the proposal must first be set to Awarded.

With the flag on, **Create Project from Proposal** becomes **Create Project Request
from Proposal**. Copy available values and the proposal relation. The creator
supplies the remaining information.

With the flag on, show a **Create Subjob** button on the parent job's detail
page to anyone authorized to create a job request. Use the same parent-job
eligibility as the current **Create Sub-Job** entry point. A closed parent
remains eligible; proposals cannot be parents. The button opens a new subjob
request form instead of the direct job-creation form, including for users who
also hold `job`.

Follow the current subjob-entry flow: set the parent relation from the selected
job, prefill its client, and prevent selection of a different client. Default
the intended subjob status to `Active`, even when the parent is `Closed`.
The requester completes the remaining data, attaches the PDF, selects a
qualified approver, and submits through the normal request workflow. Opening
the form or saving the request does not allocate a job number or create a job.
Only the separate creation step creates the numbered subjob after approval.

Apply request-creation permissions to the button, form route, and server
actions. An active employee with `time` must be able to use this flow without
`job`; hiding the button is not an access check. With the flag off, retain the
current direct subjob-creation flow and its `job` permission requirement.

A subjob request retains its parent relation. Prefill the parent's proposal
when present, but let the creator keep, replace, or clear it. A selected proposal
must have the same client as the parent. Apply the same proposal-status rules.
Creation must use the selected relations; it must not restore a cleared proposal.

## 3. Implementation Rules

### Request Approval

In a database transaction, check the flag, actor, approval authority, request
state, selected approver identity, and the version reviewed. Reload and validate
the complete request, related records, PDF/hash, and allocations. Check the
proposal status, cleared rejection fields, and required checklist answers. Fail
if reviewed data changed. Validate the exact version 1 checklist defined above;
UI checkbox validation alone is not sufficient.

Record approval against the exact attachment and project value. Retain the
approver's limit at approval and the checklist version/answers. Move the request
to `Awaiting creation`. Do not allocate a job number or delete the request.

### Recall, Rejection, And Correction

Enforce creator-only recall and selected-approver-only rejection on requests,
including after approval and before creation. Clear invalidated approval fields
and change request state atomically. A successful creator edit of a rejected
request clears all rejection fields; submission cannot clear them itself.
Synchronize request and attachment state so neither can authorize creation
after recall, rejection, or an edit. Failed edits preserve existing state.

### Job Creation

The `job` holder completes the duplicate-work check before creation, including
existing jobs created since the request was approved and other pending requests.
An overlap between requests is for the `job` holder to coordinate; the financial
approver is not responsible for reviewing unassigned requests.

Creation must complete in one database transaction:

1. Check the flag, the actor's `job` claim, and `Awaiting creation` state.
2. Reload the request and verify that its version, PDF, and value match the
   approval. Validate current related records and allocations, including an
   `Awarded` linked proposal. Fail if approval is stale or data is invalid.
3. Create the job and allocations through the existing validation and
   number-generation logic. Map `fn_agreement_answer` to `jobs.fn_agreement`.
   Never treat an unanswered draft value as No. Map the approved request's
   `project_status` to `jobs.status`; only `Active` or `Closed` is valid.
4. Transfer the approved attachment to the job and preserve its approval data.
   Copy request origin metadata and record who performed creation and when.
   Preserve only the final valid request approval used for this creation.
5. Delete the request and its draft allocations.

A failure must leave no new job and preserve the request, attachment, and
allocations. Concurrent or repeated creation must not create two jobs. The
creation actor cannot substitute an unapproved PDF or value.

### Project-Value Approval And Admin Deletion

When a proposed value and attachment are saved, require `job`, a selected
approver qualified for the full proposed value, a unique PDF, and valid job
data. Evaluate the positive-value rule against the proposed value, not the
job's unchanged historical value. Validate the other job fields normally.
Store the change against the existing job. Do not write its proposed value to
the job or create a request.

At approval, check that the actor is the selected approver and still has
sufficient authority. Recheck the pending record, reviewed data, rejection
fields, and job's current approval sequence/value in a transaction. Store the
job's previous effective value on the attachment, record approval, and update
`jobs.project_value` atomically. A stale, failed, or repeated approval must not
replace a newer approved value.

For admin deletion, recheck `admin`, the latest-approval rule, and initial
approval protection in a transaction. Refuse deletion while a pending or
unresolved rejected change remains; the admin must delete it first. Restore
the approved row's stored previous value and delete that row atomically.
This also supports an older job returning to a value with no attachment.
Exempt only this exact historical restoration from the positive-value rule
and new-approval requirement. Do not apply pending-change approver checks to
the admin restoration or allow the caller to supply a different restored value.
Concurrent approval or deletion must not cause a wrong value to be restored.

Database transactions do not make file storage transactional. Use PocketBase
file lifecycle handling so rollback does not remove a retained PDF, failed
creation does not orphan its request PDF, and successful authorized deletion
removes the file. Enforce hash uniqueness in the database as well as reporting
duplicates at upload. A database lookup failure must not permit an upload.

Compute SHA-256 from the uploaded bytes. Later workflow actions compare the
recorded hash and review version and check that the stored object exists; they
do not download or rehash the PDF. Request-to-job creation keeps the same
attachment record and storage path. If an editable request or pending value
change receives an upload with the same hash, restore the object only if it is
missing. This recovery leaves workflow state, rejection details, versions, and
approval evidence unchanged. Approved job attachments remain locked.

### Common Guards

The server owns workflow state, actor identities, hashes, approval sequence,
approval metadata, and effective values. Protect them in collection rules and
save hooks as well as custom routes. Block generic writes that bypass approval,
creation, project-value approval, or admin deletion.

Enforce the creation flag on every project write path. Apply the new edit rules
to custom APIs, generic collection APIs, and relevant model saves. A status-only
shortcut may bypass the new validation only when it sets a project to `Closed`
or `Cancelled`. Removing a claim or changing its limit
must not cascade-delete historical approvals.

Apply the allocation exception consistently to the UI and server save paths.
An assigned manager or alternate manager without `job` must be able to save
valid allocation changes on that job. Check the current assignment on the
server. Allocation-only saves must not pass through a blanket `job`-claim gate
or require a project-value approval. Preserve the existing allocation checks,
including active divisions, no duplicate divisions, nonnegative whole hours, and at
least one allocation. Retain stale-review checks for pending approvals when
allocations change.

Enforce the job-deletion attachment guard in collection rules and model-delete
hooks as well as custom routes. Check for attachments in the deletion
transaction. Concurrent attachment creation or approval must not allow a job
to be deleted with its evidence lost or detached.

Validate `job_requests.project_status` at submission, approval, and creation.
The request-to-job transaction must accept only `Active` or `Closed`, including
for subjobs and requests based on proposals. Reject `Cancelled` on the server;
hiding the option is not enough. This restriction does not apply to updates of
existing jobs or change direct-creation behavior outside the request workflow.
Keep `Cancelled` as a valid status in the `jobs` collection.

Keep `authorizing_document = "PA"` for projects and blank for proposals. This
legacy field does not prove approval. Keep `client_po` and
`client_reference_number` as reference fields.

## 4. SQLite Schema Changes

These changes are implemented by two sequential migrations:

1. `1790969510_client_setup.go` adds client/contact fields, invoicing profiles,
   and the optional job-profile relation. It removes legacy authorization fields,
   guards, configuration, and rejection notifications, and skips settings rows
   that do not exist. Profile writes require an active `job` holder, and client
   and contact writes now also require an active account. Contact email and
   existing job/allocation write permissions stay as on main, apart from the
   removed authorization guards. This schema step does not depend on request
   collections.
2. `1790969526_project_request_workflow.go` adds the request, approval, attachment,
   and allocation workflow. It adds request references to deletion rules, makes
   contact email required, and broadens profile creation to active `time` or
   `job` users. Existing profiles and their job links remain intact.

The first migration ships alone in the PA1 release, with the client workspace,
invoicing profiles, and removal of the legacy workflow. The PA1 branch is the
source of truth for that migration; rebase this branch onto main after PA1 merges
and keep PA1's version. The second migration ships with the request workflow.

A local database that already ran `1790780817_project_request_workflow.go` must
be rebuilt from the canonical fixtures or restored from a compatible pre-workflow
database before running these renamed migrations. Do not delete its migration
history row or run the split over the already-installed workflow. No existing
local or production database is changed as part of this source split.

PocketBase owns the SQLite schema. Create fields, tables, indexes, and access rules through PocketBase
collection migrations so `_collections` remains correct. Do not use raw SQL DDL.
SQL is permitted for data changes. Use `date +%s` for each migration prefix;
never edit a migration already on `origin/main`.

### Storage Conventions

* New tables have PocketBase `id` (`TEXT` primary key), `created`, and `updated`
  (`TEXT` timestamps).
* Single relations store IDs as `TEXT`. Set their target collection in
  PocketBase. Server validation checks ownership across related records.
* Text, select, relation, and date fields use `TEXT NOT NULL DEFAULT ''`.
  Numbers use `NUMERIC NOT NULL DEFAULT 0`; Booleans use
  `BOOLEAN NOT NULL DEFAULT FALSE`. PocketBase validation supplies required
  fields, allowed values, and numeric limits.
* Draft business fields can be empty. Apply completeness rules at submission,
  approval, and creation. A default value is not an explicit answer.
* All project-value and authority-limit number fields use PocketBase
  `onlyInt: true`. This includes stored previous values and approval limits.
  Keep their positive-value rules and the stated historical-zero exception.

### New Claim And Table: `project_authorization_props`

Add `project_authorization` to `claims`. Associate properties with the user's
assignment in `user_claims`, as the PO authority model does.

| Column | SQLite / PocketBase definition |
| --- | --- |
| `user_claim` | `TEXT`; required relation to a `user_claims` assignment for `project_authorization`. Unique. |
| `max_value` | `NUMERIC` integer; required positive limit in whole CAD dollars. No other authority dimensions. |

Use existing admin claim-management access for claim and limit maintenance.
Validate that properties reference the correct claim. A claim granted without
valid properties cannot approve anything. Removing a claim must remove its
active authority without deleting approval history or its recorded limit.

### New Table: `job_requests`

| Columns | SQLite / PocketBase definition |
| --- | --- |
| `creator` | `TEXT`; required relation to `users`, set by the server. |
| `status` | `TEXT`; select: `Draft`, `Awaiting approval`, `Awaiting creation`, `Rejected`, `Recalled`. Server-owned. No abandoned state. |
| `project_status` | `TEXT`; select `Active` or `Closed`, default `Active`. The creator selects the intended job status under the request edit rules. Copy to `jobs.status` at creation. |
| `version` | `NUMERIC`; server-owned integer used to detect stale reviews. Changes to request data or allocations must invalidate the reviewed version. |
| `submission_token` | `TEXT`; server-owned random identity for one submission, as defined above. Empty in drafts and after recall/rejection; new on every submit/resubmit. Included in `review_token` so unchanged business data cannot revive an earlier review page. |
| `approver` | `TEXT`; relation to `users`; only the creator can assign it. Validate authority on assignment, value changes, submission, and approval. |
| `submitted`, `rejected` | `TEXT`; server-owned dates. `submitted` records the latest submission. |
| `last_edited` | `TEXT`; server-owned date, initialized at creation and changed on successful business-data, attachment, or allocation edits. Workflow actions and reads do not change it. |
| `rejector`, `rejection_reason` | `TEXT`; relation to `users` and text reason. Required for a rejection. |
| `client`, `job_owner` | `TEXT`; relations to `clients`. `client` is required at submission; `job_owner` remains optional. |
| `contact` | `TEXT`; relation to `client_contacts`. |
| `manager`, `alternate_manager` | `TEXT`; relations to `users`. |
| `branch`, `rate_sheet` | `TEXT`; relations to `branches` and `rate_sheets`. |
| `parent`, `proposal` | `TEXT`; optional relations to `jobs`, with project/proposal type checks. |
| `invoicing_information` | `TEXT`; relation to `client_invoicing_information`. |
| `description`, `location`, `client_po`, `client_reference_number` | `TEXT`; use existing job field definitions. |
| `project_award_date` | `TEXT`; existing date format, `YYYY-MM-DD`. |
| `project_value`, `time_and_materials` | `NUMERIC` integer and `BOOLEAN`; positive value in whole CAD dollars required even for T&M at submission. |
| `fn_agreement_answer` | `TEXT`; select `Yes` or `No`. Empty is allowed only before submission. |

Add indexes on `(creator, status)`, `(approver, status)`, `(status, submitted)`,
`proposal`, `parent`, `client`, and `last_edited`. Query the admin list with the
60-day constant and `last_edited`; do not persist an abandoned flag. There is no
job number or downstream work relation. The PDF and approval evidence are held
in `job_attachments`.

### New Table: `job_attachments`

Use this table as the common document and approval store. A request has one
current attachment. A pending project-value approval belongs directly to its
job. No separate revision or value-change request table is needed.

| Columns | SQLite / PocketBase definition |
| --- | --- |
| `request`, `job` | `TEXT`; relations to `job_requests` and `jobs`. Exactly one must be set. Creation transfers ownership to `job` before deleting the request. |
| `document` | PocketBase file field; one required PDF. |
| `document_hash` | `TEXT`; required server-computed SHA-256 with a global unique index. |
| `uploader`, `uploaded` | `TEXT`; user relation and upload date, set by the server. |
| `project_value` | `NUMERIC` integer; full proposed or approved value in whole CAD dollars before tax, greater than zero. Must match the request value at approval and creation. |
| `status` | `TEXT`; states `Pending`, `Approved`, `Rejected`. Recall/rejection can clear a request approval; job approvals are protected. |
| `assigned_approver` | `TEXT`; selected user for a pending job-value approval. Set by a `job` holder and validated against the proposed full value. Request attachments use `job_requests.approver`. |
| `approver`, `approved` | `TEXT`; actual approving user and approval date, set by the server. |
| `previous_project_value` | `NUMERIC` integer; effective job value in whole CAD dollars immediately before approval of a value change. Set by the server in that approval transaction. Supports admin restoration, including an older job with no prior attachment. |
| `approval_max_value` | `NUMERIC` integer; copy of the limit at approval, in whole CAD dollars. Historical evidence, not a live authority source. |
| `request_version` | `NUMERIC`; approved request version, retained after request deletion. |
| `approval_sequence` | `NUMERIC`; server-owned order within a job. Set at creation or value-change approval. Do not order only by timestamps. |
| `initial_approval` | `BOOLEAN`; true only for the approval used to create the job. Protected from admin deletion. |
| `checklist_version`, `checklist_answers` | `NUMERIC` integer and PocketBase JSON field; request approval stores version 1 and its six named Boolean answers, all true. Value-change approval records the new PDF, full value, and authority. |
| `rejector`, `rejected`, `rejection_reason` | `TEXT`; user relation, date, and required rejection reason. |

Enforce one current request attachment and at most one unapproved value-change
attachment per job (`Pending` or `Rejected`). Add ownership, assigned-approver,
and pending-queue indexes, and a unique
approval sequence per job for approved rows. Hashes remain reserved while
their rows exist, including rejected records and requests in the abandoned list.
An authorized deletion releases them. Clearing a request's approval retains
its current PDF/hash; later approval can use that same row.

Do not cascade-delete transferred attachments with requests or job approval
history with jobs, users, or claims. Block job deletion while attachments remain;
do not clear attachment ownership to permit deletion. Reassignment changes the
selected user, not the identity recorded for an approval already transferred
to a job.

### New Table: `job_request_allocations`

This table stores proposed hours by division before a job exists. The existing
`job_time_allocations` table requires a job reference. At creation, copy these
rows to the job and delete the request rows. These are allocated hours, not
recorded employee time.

| Column | SQLite / PocketBase definition |
| --- | --- |
| `request` | `TEXT`; required relation to `job_requests`; delete with its request. |
| `division` | `TEXT`; required relation to `divisions`. |
| `hours` | `NUMERIC`; nonnegative, matching `job_time_allocations.hours`. |

Add a unique index on `(request, division)`. Validate at least one active
division at submission, approval, and creation. Apply request access and
version controls to allocation rows. Successful allocation edits also update
`job_requests.last_edited`.

### Client And Invoicing Data

| Table | Change |
| --- | --- |
| `clients` | Add `address` (street), `city`, `province_state`, `postal_code`, `country`, and `phone` as `TEXT`. All address fields are optional. |
| `client_contacts` | Add the same optional address fields and optional `phone` as `TEXT`; retain `email` and require it for every contact. Both project and billing contacts use this table. |
| New `client_invoicing_information` | Required `client` relation to `clients` and `contact` relation to `client_contacts`; the contact must belong to that client. Add optional `name` (up to 120 characters), `invoicing_instructions` (free-form notes), and `fax` as `TEXT`. Name and notes can be empty. Add optional `creator` relation to `users` for older records. New API records receive the authenticated creator and server creation time; neither can be changed later. |

Do not add `hst_exempt` or separate approval fields to shared records. Add indexes
on invoicing `client` and `contact`. Allow several profiles per client. Prevent
deletion of records still in use, including a client used only as a job owner.
Deleting an unused client still deletes its unused contacts through the existing
cascade. Extend the controlled client/contact merge process and its undo
handling to include requests and invoicing profiles. This process is the
explicit exception to job-client immutability. Update all affected relations
atomically and validate the final same-client relationships.

Use `jobs.invoicing_information` and `job_requests.invoicing_information` to
select the profile. Both the profile's client and its contact's client must
match the request/job client. "Same as project contact" stores that contact ID
in the profile; do not add copied contact/address fields or a fallback flag.

### Additions To `jobs`

| Columns | SQLite / PocketBase definition |
| --- | --- |
| `invoicing_information` | `TEXT`; relation to `client_invoicing_information`. Empty on existing jobs until completed under the edit rules. |
| `request_creator` | `TEXT`; relation to `users`; original request creator. |
| `request_created`, `request_submitted` | `TEXT`; original creation and final submission dates copied from the request. |
| `created_by`, `creation_completed` | `TEXT`; user relation and date for the separate creation action. |

Keep `jobs.project_value` as the effective approved value for the new workflow.
Pending values belong to attachments. Approval identity, time, value, and
checklist evidence belong to the preserved attachment, not the creation actor.
Leave request-origin metadata empty on existing and directly created jobs. Do
not reuse `pa_reviewer` or `pa_reviewed`; they describe the old PDF approval.
Add an index on `jobs.invoicing_information`.

### Configuration

Add `require_job_request_approval: false` to the JSON `value` of the `app_config`
row with `key = 'jobs'`. Preserve other keys. A missing key also means false.

### Remove The Existing Project Authorization Workflow

This checklist was checked against local `main` at `5eb5bb90`. Remove the entire
existing PDF Project Authorization workflow as part of the replacement release:
its UI, server behavior, database fields, PDF references, and approval data.
The small number of existing PDFs does not justify migration or archival.
There is no requirement to retain the old workflow or its data through a
rollback period. Do not add a compatibility mode, legacy approval records,
hash reservations, or a bridge from old approvals to the new authority matrix.

The new request and `job_attachments` workflow replaces the old behavior.
General file storage, hashing, notifications, navigation, and attachment audits
still serve other features; remove their old PA integration, not those shared
services. Keep the new workflow's PDFs, approvals, claims, queues, and guards.

#### Database And Old PDF References

Use new PocketBase migrations to remove all nine old fields from `jobs`,
including every stored value:

* `project_authorization_doc` and `project_authorization_doc_hash`.
* `pa_uploader` and `pa_uploaded`.
* `pa_reviewer` and `pa_reviewed`.
* `pa_rejector`, `pa_rejected`, and `pa_rejection_reason`.

Remove `idx_jobs_project_authorization_doc_hash` and all collection-rule guards
that reference those fields. Remove `enforce_project_authorization` from the
`jobs` configuration object, preserving unrelated keys. Delete the old
`project_authorization_rejected` notification template and clear pending old
PA notifications so they cannot deliver obsolete links or instructions.

Retain `branches.manager` and its stored assignments as branch information.
Remove its old PA upload permissions, queue visibility checks, and badge logic.
Remove the `branch_manager_id` job response field used by the old PA UI; this
does not remove the underlying branch relation. A branch-manager assignment
grants no approval authority under the new matrix. Preserve job managers,
alternate managers, employee managers, and the separate client
business-development lead relation too.

Physical cleanup of the old PDFs in S3 is outside this work. Drop the old file
field and its references; orphaned S3 objects are acceptable and can be cleaned
up manually later. Do not add storage scans, path inventories, file-deletion
jobs, or retry handling for these old PDFs. Their removal is not a release gate.
Do not copy old PDFs or approval data into the new workflow or an archive.
Preserve existing job values, allocations, and downstream work.
Keep `authorizing_document = "PA"` as specified above for the existing legacy
job-writeback contract in `app/routes/jobs_writeback.go`; it is not approval
evidence and does not preserve the old PDF workflow.

Keep historical migration files, including
`1780417194_project_authorizations.go` and
`1780930403_project_authorization_upload_rejection_metadata.go`, unchanged.
Their old fields may exist temporarily during a fresh migration replay; the
new removal migration must leave the final schema free of them.

#### Server And API

* Remove the old implementation in `app/routes/project_authorization.go`,
  `app/routes/project_authorization_doc_hash_repair.go`, and
  `app/hooks/project_authorization.go`, and their route/hook registrations.
  This includes upload certification, old Accounting review permissions,
  rejection, revocation, missing/pending/rejected queues, hash audit/repair,
  and the old mutation-context and save-invariant logic.
* Remove these old routes under `/api/jobs`: GET
  `/project_authorization/missing`, `/project_authorization/pending`, and
  `/project_authorization/rejected`; POST and DELETE
  `/{id}/project_authorization_doc`; POST
  `/{id}/project_authorization_doc_hash/audit` and
  `/{id}/project_authorization_doc_hash/replace`; and POST
  `/{id}/project_authorization/approve`,
  `/{id}/project_authorization/reject`, and
  `/{id}/project_authorization/revoke`. Do not retain aliases to these endpoints.
* Remove PA enforcement from `app/routes/bundle_timesheet.go`,
  `app/hooks/purchase_orders.go`, and `app/hooks/validate_expenses.go`.
  Remove `IsProjectAuthorizationEnforced` from `app/utilities/config.go`, its
  callers, blocking-job lookups, and PA-specific errors. The checked `main`
  has no work-record PA integration; remove one too if present when implemented.
* Remove old PA guards from `app/hooks/jobs.go`, `app/hooks/hooks.go`, and
  `app/routes/job_upsert_api.go`. Apply the new workflow's guards separately.
* Remove PA columns, joins, response fields, and file URLs from
  `app/routes/job_details.sql` and `app/routes/job_details_api.go`. Remove
  `has_project_authorization` from the shared PO visibility query, pending,
  visible, and search PO queries, and their Go/TypeScript response models.
* Remove old PA badge counts from `app/routes/nav_badges.go` and the rejection
  notification producer from `app/notifications/queue_events.go`.
  Remove the `jobs_project_authorization_doc` target from
  `app/routes/attachment_audit.go`. Preserve auditing of expense and PO files.

#### User Interface

* Remove `ui/src/routes/jobs/project_authorization/+page.svelte` and its loader,
  including the missing, pending, and rejected PDF views. Remove the old
  navigation item and its badge/access handling in `ui/src/lib/navConfig.ts`
  and `ui/src/routes/+layout.svelte`.
* Remove `ui/src/lib/components/jobs/ProjectAuthorizationDetails.svelte` and
  its old upload, certification, review, rejection, revocation, and hash-repair
  controls from job details. Remove the old PA fields from the detail loader
  and `JobsEditor.svelte` field handling.
* Remove old PA status displays from PO details and the `has_project_authorization`
  property in `ui/src/lib/poVisibility.ts`.
* Remove `ui/src/lib/projectAuthorization.ts` and its PA-blocking messages,
  manager guidance, and links from expense and PO editors and the time-entry
  list. Keep the unrelated validation in those screens.

Build the request approval UI, **Job Create Queue**, pending-value approval UI,
and new job history views defined in this document. Do not leave the old PA
screens accessible beside them.

#### Generated Files, Fixtures, Tests, And Completion Checks

Regenerate types and update API models, `_collections.csv`, `jobs.csv`,
`app_config.csv`, `notification_templates.csv`, and
`datapackage.json` for the removed fields and records. Remove the old PA PDF
fixtures under `app/test_seed_data/files/jobs/`. Update fixture rows surgically;
do not regenerate the test seed or remove records used by unrelated tests.
Preserve the branch manager field in the schema and generated types, and keep
its assignments in `branches.csv`.

Remove tests that require the old behavior in the dedicated PA hook, route,
and downstream-route test files. Remove or replace PA-specific cases in
timesheet bundle, job, PO visibility, navigation badge, and attachment-audit
tests. Add the new workflow and removal tests described below. Update help and
other documentation that still instruct users to use the retired workflow.

Verify that the removed fields, old endpoints, old error codes, and old audit
target have no remaining runtime consumers. Historical migrations and tests
that prove removal may name them. The new `project_authorization` claim and
related names are expected; a search for that phrase alone is not a cleanup test.

## 5. Delivery And Acceptance

1. Review the final schema and transitions against the confirmed requirements.
2. Complete the existing-workflow removal checklist above, including old UI,
   server behavior, approval fields, configuration, and old PDF references.
3. Add the claim, properties, attachments, requests, fields, and config. Build
   with the new creation flag off. Existing-job value changes still require
   approval. Do not migrate old PA approvals or PDFs into the new tables.
4. Build request entry and printing, using PO printing as the reference. Build
   approval, the **Job Create Queue** and its badge,
   creation, duplicate comparison, pending project-value approval,
   admin triage/deletion, and job history views. Update
   notifications and badges for the queues. Train users on value changes.
5. Update generated types, API responses, and CSV fixtures. Update
   `datapackage.json` where needed. Add new workflow fixtures and remove obsolete
   PA fields and test fixture files as specified above; do not dump new test data.
6. Run all tests and frontend checks/build. Review the changes and complete
   acceptance cases. Verify both flag states on every app instance.
7. Coordinate the replacement release so old application instances cannot read
   or write fields while they are removed. Verify the final schema and enable
   request approval after acceptance. Old S3 objects do not block release.
   Turning the new flag off retains new requests; it does not restore the old
   workflow. Recovery must use a release compatible with the cleaned schema.

Acceptance tests must cover these cases:

* Every Project Request can produce a printable document. Complete client
  documents have client and TBT representative signature spaces; incomplete
  documents have the draft warning, missing-item list, and no signature lines.
  Test printing before approval, before PDF upload, and without a job number.
  Missing internal approval data does not force an otherwise complete client
  document into draft form. Missing optional contact details and references
  do not block signature readiness.
* Check the print action and user flow against PO printing, including Save as
  PDF, the fallback instruction, and save-or-cancel handling for unsaved edits. Enforce
  request-view permissions, including direct print access. Printing does not
  change request state, version, or last-edited time, grant approval, attach a
  PDF, or create a job.
* Verify client name, full description/location, proposal number, parent project,
  client references, project contacts, billing contacts/address fallback, and
  invoicing instructions against saved data. Test shared and different project
  and billing contacts, missing optional data, and long multi-page content.
  Check that no internal approval, allocation, or cost data appears.
* Verify whole-dollar CAD values before tax, the additional-tax wording, fixed
  fee and T&M labels, and the selected rate-sheet name/revision/effective date.
  T&M output includes the matching billing-rate appendix in the same printout;
  unavailable rates produce a draft. No estimate or spending cap is inferred.
* Check agreement wording, blank signatory name/title/signature/date fields,
  readable Letter-page layout, and intact signature blocks in print and PDF.
  Upload a signed copy through the existing PDF flow, including the T&M appendix.
  Other sufficient authorization PDFs remain accepted. Later shared-record
  edits affect new printouts but leave the uploaded signed evidence unchanged.
* Removal migrations succeed with old PA fields populated and with them empty.
  All nine old job fields, the old hash index, obsolete field guards, config
  key, and notification template are absent afterward. Remaining old S3 objects
  do not prevent completion. No old PDFs, approvals, or hashes are imported into
  the new workflow. Existing job business data and all new workflow evidence
  remain intact. Test fresh migration replay.
* Cleanup preserves `branches.manager` and every existing assignment. A branch
  manager without the required claim and limit cannot approve a request or
  project-value change. The old branch-based PA upload and queue permissions
  are removed.
* Old PA pages, actions, queues, badges, response fields, notification producers,
  hash-repair tools, and downstream blocks are gone. Old API routes cannot
  execute their former actions. Expense and PO attachment audits, general
  notification delivery, and unrelated job/PO/time/expense behavior still work.
* Authority with no claim, missing properties, zero/invalid limit, inactive user,
  insufficient limit, exact limit, and greater limit. Validate assignment and
  value writes, then recheck current authority at approval. Test claim removal
  and limit reduction after assignment. No division/branch filter.
* Selected-approver-only approval and rejection, creator-only assignment and
  recall, qualified creator self-approval, and removal of assignment-based
  access after reassignment. Approvers cannot list or directly read unassigned
  requests through that claim. Preserve independent creator/job/admin access.
* Approvers compare requests only with existing jobs. Show no unassigned
  requests in their duplicate-review results. The `job` holder checks existing
  jobs again before creation and coordinates overlaps between pending requests,
  without gaining recall or rejection rights. A possible match alone does not
  cause automatic rejection.
* Separate approval and creation permissions, including one person with both
  claims. Approval creates no job; `job` alone cannot approve a value.
* The **Job Create Queue** and badge are available only to active `job` holders
  while the request flag is on. Both include all approved requests in
  `Awaiting creation`, including subjobs and requests created by other users.
  Test matching queue/count selection across pagination, addition on approval,
  and removal on creation, recall, rejection, and admin deletion. Failed
  creation retains the request and reports the error. Age alone does not
  remove it. Turning the flag off hides the entry point and badge and blocks
  creation; turning it on restores the queue and count from retained state.
  Direct queue and creation API calls must enforce the same permissions.
* Requests default their intended project status to `Active`. `Closed` can be
  selected for an intended parent/reporting job before child jobs exist.
  Preserve the approved selection through creation. Reject `Cancelled` at
  request submission, approval, and the request-to-job transaction, including
  direct API calls to that workflow. Permit an existing job to become
  `Cancelled` under its normal edit rules. Do not apply this restriction to
  direct creation outside the request workflow. Keep request workflow status
  separate from the resulting job status.
* Recall and rejection before and after request approval, with earlier approval
  fields cleared. Reuse the retained PDF for fresh approval. Only the final
  valid approval survives the creation transaction.
* Rejected requests cannot be submitted unchanged. Successful creator edits
  clear all rejection fields; failed edits, unchanged saves, and submission
  calls do not. Check request/attachment state consistency and complete data
  on resubmission. Test drafts, allocations, and approver changes too.
* Positive values for fixed-fee and T&M projects, CAD before-tax comparison,
  full proposed value rather than its increase, and value decreases.
* Whole-dollar values and limits throughout assignment, request entry,
  approval, creation, value changes, and historical restoration. Reject
  fractional inputs, including direct API writes, without rounding or
  truncation. Retained values and limits must match the approved integers.
* Request checklist version 1 starts unchecked and requires all six named
  Boolean answers to be true. Test each missing or false answer, invalid answer
  types, unsupported versions, and direct approval calls that bypass the UI.
  A reviewed First Nations answer of either `Yes` or `No` is permitted.
  Recall and rejection clear checklist evidence; fresh approval requires all
  confirmations again. Preserve the final checklist through job creation.
* Stale review data, invalid related records, missing checklist answers,
  concurrent actions, and transaction rollback. Failed creation preserves the
  approved request, PDF, and allocations. Repeated creation cannot create two jobs.
* Attachment transfer with unchanged file/hash/approval data, global duplicate
  uploads including simultaneous uploads, and abandoned-request errors that
  identify the request and explain admin resolution.
* Only `job` holders can enter or edit a proposed project value and upload its
  PDF. Only its selected, qualified approver can approve it. A user with both
  claims can perform both actions. No new request is created.
* Pending and rejected values do not replace the job value. Approval updates
  it atomically and records the previous value. Corrections clear rejection
  fields and require a fresh approval. Test the one-pending-change rule.
* Other valid job edits need no new PDF or approval. Direct value writes cannot
  bypass approval. Downstream work continues during value review, including on
  older jobs with no previous attachment or approval backfill.
* A project's manager and alternate manager, each without `job`, can use the UI
  to add and remove division allocations and change hours. Valid saves persist
  through the server in both creation-flag states. An unrelated user is denied.
  Reject invalid allocations, changes to other job fields, and attempts to move
  an allocation to another job. Removing a manager assignment removes this
  exception. Holders of `job` retain their existing allocation access. These
  edits require no new PDF or proposed project value and must not change the
  effective project value or approved evidence.
* An older T&M job with zero effective value can receive a pending positive
  value without changing the stored zero. Approval applies the positive value.
  Admin deletion can restore the exact historical zero with no attachment and
  no new approval. Reject arbitrary restored values and continue to reject
  zero values on new requests and proposed changes.
* Admin deletion of the latest approved value change, refusal to delete an
  earlier or initial creation approval, and restoration of the previous value.
  On an older job, deletion of its first later approval restores its original
  value with no attachment. A pending or unresolved rejected change must be
  deleted first. Test concurrent changes, rollback, file cleanup, and PDF reuse.
* The admin-only abandoned-request list uses the 60-day code constant and time
  since the last edit. Test just below and at the threshold, initial creation,
  attachment/allocation edits, and removal from the list after an edit. Reads
  and workflow actions do not reset the age. List membership changes no status,
  does not block normal actions, and triggers no automatic deletion.
* Admin deletion of abandoned requests releases their PDF hashes. Ordinary
  users cannot perform this cleanup or delete approved job history.
* Job deletion fails while any initial, later, pending, or rejected attachment
  remains, including through generic APIs and model deletion. Preserve the
  job, PDFs, hashes, and approval history on failure. Test concurrent attachment
  creation or approval. After permitted admin cleanup leaves no attachments,
  existing job-deletion permissions and dependent-record checks still apply.
* Both flag states on every creation path, including imports and admin routes.
  Flag changes retain request state and age. Existing-job value changes still
  require approval while the creation flag is off.
* Allowed draft proposal statuses; award checks at submission, resubmission,
  approval, and creation; and the same rules for subjobs. Test multiple projects
  per proposal, requests without proposals, and proposal replacement/removal.
* With the request flag on, an active employee with `time` and without `job`
  can use **Create Subjob** on an eligible parent's detail page, load the form,
  save a draft, and submit a complete request. Users with both claims follow
  the same request flow. Check parent/client setup, the client-selection lock,
  proposal prefill, and an `Active` subjob default under a closed parent.
  Proposals are not eligible parents. Unauthorized users cannot use the form
  or server actions. No job or number is created before approval and the
  separate creation step. With the flag off, direct subjob creation still
  requires `job` and the button cannot create a request.
* Same-client relations, immutable project client, duplicate-work review without
  blocking mere possible matches, allocation transfer, required billing data,
  shared-record edits, and ordinary edit/field permissions.
* Controlled client merges and undo update all affected job, request, contact,
  owner, and invoicing-profile references atomically. Contact merges update
  both project and billing contact references. Failed merges leave all records
  unchanged. Apply stale-review guards to affected requests, preserve approved
  document/value/approver history, and block ordinary job-client reassignment.
* Optional address fields on clients and contacts, required email on every
  contact, and optional contact phone. A job can use the same contact record for
  project and billing contact or select different contacts from the same client.
  Profiles are reusable and hold a contact reference rather than copied data.
  Shared edits remain visible through their references and cannot break client
  ownership rules. Changing a job's contact does not rewrite its profile.
* Empty invoicing notes are allowed, but required profile/contact relations
  remain enforced. `job_owner` can be empty; a supplied owner must reference a
  valid client. No report-delivery fields are required.
* Client workspace navigation, the direct tab routes, all three job lists, note
  associations, removal of the old client edit route, and safe editor return
  links. Test lists with at least 100 entries, filtering beyond the first page,
  independent pagination, and retained state after save, cancel, Back, and
  refresh. Test empty lists and read-only viewers.
* One contact editor across client, job, and invoicing entry points. Cover
  validation failures, referenced-record delete failures, shared updates,
  optional profile names, nested contact editing, and preserved project drafts.
  Cover shared-update counts, all three save choices, hover and focus help,
  cancelled and failed saves, and created, copied, or reused profiles that select
  only the originating draft. Cover time-only creation with existing contacts,
  denial of edits to all existing profiles and contacts, same-client validation,
  creator attribution and spoof prevention, missing contacts, and empty lists.
  Profile creation must not bypass request approval or save the originating job.
* Protected job approval history, preserved identities and times after request
  deletion, and claim changes that do not delete historical evidence. Full
  snapshots of other editable job fields are not required.


## 6. Operating And Release Instructions

### Staff Workflow

1. Open **Project Requests** and select **New Project Request**. A holder of
   `time` can also use **Create Subjob** on an eligible parent project.
2. Save the project details and division hours. Select a billing profile and
   an approver whose limit covers the full project value. Active projects also
   require the existing project completion date, including an estimated date.
3. Use **Print Project Authorization** to obtain signatures. Save edits before
   printing. Incomplete client documents show a draft warning and no signature
   lines. T&M documents include the client rate appendix.
4. Upload one authorization PDF and select **Submit**. The selected approver
   uses **Request Approvals**, reviews possible duplicate jobs, and confirms
   all six checklist statements before approval.
5. A holder of `job` opens **Job Create Queue**, checks the approved request and
   PDF, and checks existing jobs and other requests for duplicate work. Select
   **Create Job** to assign the number and transfer the approval evidence.

The creator can recall a submitted or approved request before creation. The
selected approver can reject it with a reason. Rejected requests require a real
edit before resubmission. Both actions clear the earlier approval.

On an existing job, use **Propose a new project value**. Upload the new PDF and
select an approver. The effective value stays unchanged until approval. Use
**Value Approvals** to review these changes. The creation flag does not disable
this process. Shared-record edits can make an open review stale; reload and
review the saved data again. A job editor can save a pending value change again
to refresh its review, without changing its amount or PDF. The value proposal
form also permits corrections to job details and allocations in the same
transaction. This lets staff complete billing data for an older zero-value job
while keeping its historical effective value unchanged until approval. A failed
save leaves both the job and its approval evidence unchanged.

Holders of `job` maintain client details, contacts, and reusable invoicing profiles
from the **Client details** tab of the client's workspace. Active holders of
`time` can create profiles from existing contacts, but cannot edit shared records.
Select a contact or profile to edit it when authorized. Use a different profile when only one project's invoicing
arrangement should change. Admins maintain approval limits from the claims
page. A branch manager assignment does not grant approval authority.

Admins use **Abandoned Requests** for requests with no edits in at least 60 days.
Deletion removes the request's PDF and releases its hash. On a job, an admin can
delete pending changes, or the latest later approval after pending changes are
removed. Deleting a later approval restores its exact previous value. Initial
creation approvals remain protected.

### Release Checks

The migration defaults `jobs.require_job_request_approval` to false. The old
workflow is removed in either flag state. The new value-change controls apply
in either flag state. No old approval or PDF is copied into the new tables.

Before production deployment:

1. Read the live configuration on each instance. Record the old setting before
   replacing the release. Take a database backup for recovery.
2. Stop old application instances before the removal migration. Do not run an
   old binary against the new schema.
3. Deploy the server and UI together. Verify the final schema and application
   version on every instance. Check direct project creation with the flag off.
4. Assign the new claim and whole-dollar limits to the approved users. Complete
   billing profiles as jobs are edited. Train staff on requests and value changes.
5. Enable `jobs.require_job_request_approval` after acceptance. Check request
   entry, printing, approval, the queue and badge, and job creation on each
   instance. Verify direct project creation is blocked.

An absent creation flag means off. A malformed setting or database read failure
blocks creation until corrected. Turning the flag off retains requests and
approval evidence; it does not restore the old workflow. Recovery requires a
release that supports the cleaned schema, or restoration of the pre-release
backup. Old S3 objects require no automatic cleanup and do not block deployment.

The legacy MySQL import is no longer used and will be removed separately. Until
then, its bulk job import refuses to run while request approval is required, or
while requests, authorization attachments, or billing profiles exist. This
prevents its table-replacement process from removing retained evidence or shared
data. In practice the import stops working once the first billing profile exists
after PA1; that is expected.

Local verification includes both Go modules, frontend checks and build, all
frontend unit and browser tests, migration replay with empty and populated old
evidence, and PDF output checks. Short fixed-fee output fits one Letter page;
T&M adds a rate appendix; long descriptions continue without truncation.
Production deployment, production data checks, and activation are separate
release work and have not been performed in this checkout.

## 7. Release Stages

Each stage branches from main and carries this complete document. Until the last
stage ships, sections 1 to 6 describe the target design, and this section says
what main delivers. Update this section in each stage.

### PA1: Client Setup And Legacy Removal

PA1 ships `1790969510_client_setup.go` and:

* The client workspace: separate Client details, Jobs, and Notes routes, client
  addresses and phone numbers, contacts, and reusable invoicing profiles with the
  shared-update, save-as-new, and reuse choices.
* A required invoicing profile, with a project contact and profile of the same
  client, on project creation and ordinary project edits. This applies through
  the collection API and the custom job API, including parent projects and
  subjobs. Existing projects receive no guessed profile. Setting only the
  status of an older project to `Closed` or `Cancelled` needs no profile;
  setting it to `Active` does. The stale and unused job lists show save errors
  beside the job.
* The return from client maintenance to an unsaved job form, with the new,
  copied, or reused profile selected but not saved.
* Complete removal of the legacy PDF workflow listed in section 4, including its
  nine job fields, setting, notification template, routes, and screens.
* The import guard described in section 6.

Until later stages, PA1 differs from the complete design in these ways:

* There are no job requests, approvals, attachments, value-change approvals,
  Job Create Queue, `project_authorization` claim, or creation flag. Holders of
  `job` create and edit projects directly, including client and project value.
* Only active holders of `job` can create invoicing profiles. Holders of `time`
  alone cannot.
* Contact email remains optional. A supplied email must be valid.
* Managers and alternate managers without `job` keep main's job editor. When
  only division hours change, it saves just the allocations, so a project with
  no profile needs none. Other changes to such a project require selecting an
  existing profile; only holders of `job` can create one.

Before deploying PA1:

1. Back up the production database, which must be on the deployed main schema.
   Confirm that no request workflow migration has run on it.
2. Count jobs whose project contact belongs to a different client. Client merges
   that touch those jobs fail until the contacts are corrected.
3. Stop old application instances, then deploy the server and UI together.

The migration refuses to run down because the removed legacy data cannot be
rebuilt. To roll back, restore the backup with the matching application
version. Old S3 objects from the legacy workflow can be cleaned up separately.

After PA1 merges, rebase the `project_authorization` branch onto main. Keep
PA1's client setup migration and client code, and resolve conflicts in this
document by taking the latest version. Then restore the request workflow on top,
including `1790969526_project_request_workflow.go`.

### Later Stages

The request, approval, job creation, and value-change workflow in sections 2 to
6 ships after PA1, with `1790969526_project_request_workflow.go`. It also makes
contact email required and lets active holders of `time` create invoicing
profiles from existing contacts.
