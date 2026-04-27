# Work Records

This document turns stakeholder notes and follow-up decisions into a proposed
product and implementation spec for first-class work records in Turbo.

It covers:

- workflow findings from the current paper/Excel process
- resolved product decisions after stakeholder discussion
- schema and backend direction for Phase 2
- UI and workflow expectations
- migration impact on existing `time_entries`
- a revised phased plan

## Executive Summary

Work records are currently a required field-work artifact managed through a
manual workflow centered on Zena, a physical binder, paper forms, Excel, and
PDF handoff to invoicing. Turbo already has a `work_record` string on
`time_entries`, plus a read-only audit surface for searching those strings, but
it does not yet model work records as first-class records.

Safety-related information such as Field Level Hazard Assessments, Pre-Start
Tailgate Meetings, and Incident Reports is already captured and stored in Site
Docs and is not part of the work record scope.

Recent stakeholder discussion changed the most important modeling assumption:
the business does not always operate as "one work record per person per
day per job." Crews may share one work record number for one job/day/work
context while still recording person-specific hours and travel details.

The revised direction is:

- add a first-class `work_records` collection for shared crew-day work
- generate all new work record numbers on the backend with a `W` prefix
- keep `creator` on `work_records`, but remove `uid` from that table
- add `work_records_subjects` as the participant / worker child collection
- move `uid`, `hours_on_site`, and `hours_travel_time` onto
  `work_records_subjects`
- also move other person-specific travel fields onto `work_records_subjects`,
  including `distance_travelled_km`, `vehicle_type`, `is_passenger`, and
  `company_vehicle_unit_number`, because stakeholder discussion strongly
  suggests those are per-person rather than shared crew fields
- keep work record type on the shared `work_records` parent
- use `work_records_types.max_subjects` to limit how many worker rows a work
  record of that type may have
- add optional `work_records.lane_kms` for billing support, with availability
  controlled by the selected work record type
- track consumable usage per work record through normalized supporting
  collections (`work_records_consumables`, `work_records_type_consumables`,
  `work_records_consumable_entries`), with entries referencing
  `work_records_consumables` directly and consumable visibility driven by the
  selected work record type via `work_records_type_consumables`
- replace the planned `time_entries.work_record_id` relation with
  `time_entries.work_record_subject_id`, so each time entry links to one
  participant row rather than ambiguously linking to a shared parent
- enforce that a time entry referencing a work record worker row has
  `hours = hours_on_site + hours_travel_time` for that worker row
- preserve legacy `time_entries.work_record` text for existing historical data
- store notes as append-only records
- allow a scanned paper-copy PDF to be attached to a saved work record for
  reference and billing support
- support related work records through a dedicated `Copy To Tomorrow`
  workflow, with each copied parent record receiving its own independent number
- redesign approval locking so participant rows lock per person while shared
  parent fields lock once any linked participant is approved
- add a print-friendly work-record route similar to purchase orders; the print
  view is the same for every authorized viewer and includes all worker rows
  with their separate hours and vehicle usage in a compact summary

## Findings From Discovery

### Current Business Workflow

- A worker must have a work record in order to do field work.
- Requests for work record numbers arrive through many channels: Teams, email,
  in person, and ad hoc conversation.
- Zena issues a blank work record and a work record number before field work
  when possible, but in practice some records are created after the fact.
- Number assignment is manual and comes from a shared binder.
- The binder is both the number source and the operational tracking mechanism.
- When Zena is absent, someone else uses the same binder and process.
- Workers often complete records on paper.
- Workers sometimes submit a single work record covering multiple days, even
  though the business prefers daily separation.
- Workers also often submit one shared work record covering multiple people
  doing the same work on the same day.
- Zena chases incomplete or missing work records after the work returns.
- Once complete, Zena saves the record as PDF and places it where Sudha can use
  it during invoice-summary preparation.
- The work record number is also recorded on the timesheet today.

### Core Domain Rules

- The accounting unit remains the time entry.
- A work record may represent one crew doing the same work for one job on one
  day.
- If two people on the same job/day are doing materially different work, they
  should use different work record numbers.
- The same worker should not appear on more than one work record for the same
  job/date.
- The same worker may appear on work records for different jobs on the same
  date.
- Daily separation still matters because some charges, such as meals and
  vehicles, are billable per day.
- Shared across a crew work record are things like job, date, location, work
  description, most job-context metadata, and usually consumables.
- Some work record types use lane kilometres to inform invoicing; this is a
  shared work-record value rather than a per-person value.
- Per-person within a crew work record are things like employee identity, hours
  on site, travel time, and vehicle/travel details.
- Work records track time plus consumables.
- Stakeholder discussion cast doubt on the earlier "only four types"
  assumption; the type model should stay flexible until stakeholders resolve
  the current list.
- Notes may contain client-facing context, training details, or explanations of
  what happened on site.
- In rare cases, clients ask to receive work records directly, but this is not
  part of the current implementation scope.

### Operational Constraints

- Zena spends roughly half of her working time on work records.
- She also handles field logistics and legal survey field books, so reducing
  manual work here likely has high value.
- Existing safety-related content appears to have moved out of work records and
  into site documents.
- The business does not want to force paperless field work; structured digital
  records should coexist with paper and scanned PDF packets.

### Findings From The Current Codebase

- Turbo already stores a `work_record` text field on `time_entries`.
- Time entry validation currently expects a string matching
  `^[FKQ][0-9]{2}-[0-9]{3,4}(-[0-9]+)?$` when a job is present.
- Timesheet validation currently prevents the same work record string from
  appearing on more than one time entry in the same timesheet.
- Turbo already has a read-only `/time/work-records` audit surface that groups
  and searches work record strings pulled from `time_entries`.
- The nav already contains a `Work Records` item, but today it is an audit
  view, not a create/edit workflow.

## Resolved Product Decisions

### Numbering

- All new first-class work records will use backend-generated numbers beginning
  with `W`.
- Number generation should follow the same general pattern as purchase order
  numbering, but with a work-record prefix.
- All work records use the same number format: `WYYMM-NNNN`.
- There is no dash-suffix scheme; every work record, whether created fresh or
  through `Copy To Tomorrow`, receives its own independent sequential number.
- The `parent_work_record` relation tracks lineage between related shared
  parent records, but numbering does not encode that relationship.
- Users will not manually enter or reserve work record numbers in the new flow.

### Work Record Identity On Time Entries

- Existing `time_entries.work_record` text values stay in place for historical
  records.
- After rollout, new functionality should stop relying on
  `time_entries.work_record` as the primary link.
- The clean relational link is no longer from `time_entries` to the shared
  `work_records` parent.
- Instead, a new `time_entries.work_record_subject_id` relation should
  reference `work_records_subjects.id`.
- This relation is the first-class link between time entries and work records,
  because the time entry still belongs to one person even when the work record
  parent is shared by a crew.
- New validation and workflow logic should use `work_record_subject_id`.
- Legacy reports and any remaining historical search surfaces outside the main
  work-record list may continue to read `time_entries.work_record` until they
  are migrated.

### Rollout Modes For Time Entries

The bullets above describe the intended end state once participant-linked work
records are fully rolled out. The implementation may pass through an explicit
hybrid phase first, and that phase should be documented separately from both
the current `main` branch and the final linked-only state.

- Current `main` branch behavior:
  - time entries rely on the legacy `time_entries.work_record` text field
  - the existing `/time/work-records` surface is read-only audit/search based
    on legacy text values pulled from time entries
  - there is no requirement yet to upgrade legacy time entries to a
    first-class worker relation before editing
- Hybrid rollout mode:
  - first-class `work_records`, `work_records_subjects`, and
    `time_entries.work_record_subject_id` exist and new work-record-aware
    flows should prefer them
  - legacy `time_entries.work_record` text remains editable and savable for
    pre-existing historical rows
  - legacy text-only work records may still be created or edited through the
    existing time-entry flow while linked-only mode is disabled
  - this mode is intended to let the first-class model ship without forcing an
    all-at-once migration of every historical editing path on day one
  - in the current implementation proposal, hybrid mode is the default and
    linked-only mode is enabled later through `app_config.key =
    "linked_work_records"` with `value.enabled = true`
- Linked-only rollout mode:
  - new time-entry workflows must use `work_record_subject_id` as the primary
    link
  - creating new text-only work-record values through time entries is disabled
  - editing an existing time entry that still relies only on legacy
    `work_record` text requires upgrading it to a real
    `work_record_subject_id` before save

When this document says "after rollout" or "once rollout is live," it refers
to the linked-only mode above, not the hybrid transition phase.

### Type Ownership

- Work record type lives on the shared `work_records` parent.
- Notes do not carry type.
- Type should remain data-driven rather than hard-coded around the earlier
  four-type assumption.

### Granularity

- The business distinction is no longer "one work record per person per day per
  job."
- The shared parent record is one work record per crew-day per job when the
  crew is doing the same work.
- Each shared parent work record contains one or more worker rows in
  `work_records_subjects`.
- Each worker row represents one participant employee on that shared work
  record.
- Multi-day records are still represented as a group of related shared parent
  work records linked through `parent_work_record`.
- Related work records are created only through `Copy To Tomorrow`.
- Each shared parent record in the group receives its own independent
  `WYYMM-NNNN` number.

### Export Scope

- Uploading and viewing a scanned paper-copy PDF attachment is in scope for the
  current implementation.
- Generating a print-friendly work record from Turbo is in scope for the
  current implementation through a dedicated browser print route.
- The print-friendly route should use the same work-record content for every
  authorized viewer, include every worker row, and show each worker's hours
  and vehicle usage separately in a compact, simple layout.
- Direct printer integration and server-generated PDFs remain out of scope;
  users print to paper or save to PDF from the browser print dialog.

## Recommended Product Model

### Main Entity Rules

- A work record is a first-class shared parent record, not just a string on a
  time entry.
- A work record belongs to one job and one calendar day.
- A work record has one or more worker rows in `work_records_subjects`.
- Each worker row belongs to exactly one employee (`uid`) and exactly one
  parent work record.
- A work record may optionally reference a parent work record (via
  `parent_work_record`) when created from `Copy To Tomorrow`, for lineage
  tracking only; numbering is independent.
- A parent work record may be referenced indirectly by zero or more time
  entries through its worker rows.
- A worker row may be referenced by zero or one time entry through
  `time_entries.work_record_subject_id`.
- A work record can exist before, during, or after field work.

### Approval Model

- The old single boolean `work_records.approved` does not fit the shared-parent
  model by itself.
- Approval should become person-specific at the worker level:
  `work_records_subjects.approved` is a system-managed boolean.
- A worker row becomes approved only when the manager approves the
  `time_sheets` record that contains the `time_entries` row referencing that
  worker.
- When a worker row is approved, that worker row becomes locked for normal
  editing.
- Shared parent fields and shared consumables should lock as soon as any linked
  worker row is approved, because those shared values are now part of an
  approved accounting artifact.
- The worker roster is part of the shared parent state for locking purposes:
  after any worker row is approved, worker rows cannot be added, removed, or
  reassigned.
- Worker rows that are not yet individually approved may still have their
  per-person values edited by their own worker user.
- If a related timesheet becomes unapproved, recalled, or rejected, the
  corresponding worker row's `approved` field should be set back to `false`.
- Shared parent fields should unlock again only when no linked worker rows are
  still approved.
- Append-only notes remain allowed regardless of approval state.
- The UI may show a derived parent status such as `Draft`, `Partially
  Approved`, or `Fully Approved`, but those can be computed rather than stored
  in v1.

### Delete Rules

- Deletion of work records is out of scope for the UI in this phase.
- A parent work record may only be deleted if no worker row is referenced by a
  time entry.
- A worker row may only be deleted if no time entry references it via
  `work_record_subject_id`.
- Deletion is restricted to superusers in the PocketBase backend.

### Permissions

- A user must hold the `time` claim to create a work record for a crew that
  includes themselves. In the simple self-create case, `creator` is the current
  user and the first worker row `uid` is also the current user.
- Holders of the `work_record` claim can create work records on behalf of other
  users who hold the `time` claim.
- Users without the `work_record` claim cannot create work records solely for
  other users.
- During creation, the UI may ask whether the record is for "you / your crew"
  or "someone else's crew", but the latter path must only be available to
  holders of the `work_record` claim.
- Any authenticated user may read the lookup collections needed to populate the
  work record UI: `work_records_types`, `work_records_consumables`, and
  `work_records_type_consumables`.
- Read access to first-class work record data should be limited to the relevant
  participants plus reporting access:
  - `time` holders may read work records where they are a worker-row `uid`
  - work record `creator` visibility is independent of participation;
    creators may read work records they created
  - holders of the `report` claim may read all work records
  - `work_record` claim alone does not grant broad organization-wide read
    access beyond records the user created
- `creator` is immutable after parent creation.
- Worker-row `uid` should be immutable after creation; replace the row instead
  of reassigning it.
- Updates to shared parent fields are allowed to the `creator` and any worker
  row `uid` while no linked worker rows are approved.
- Worker rows are created by the parent creator.
- Updates to an unapproved worker row are restricted to that worker-row
  `uid`.

## Phase 2 Data Model

Phase 2 should introduce first-class collections plus the new time-entry
relation.

### `work_records`

Purpose:

- the shared parent record for one crew/day/job/work context

Fields:

- `id`: PocketBase id
- `number`: text, unique, backend generated; schema-optional on create because
  the client does not send it; immutable once created
- `parent_work_record`: optional self-relation pointing at the root parent
  record, used for lineage tracking only; does not affect numbering
- `date`: text, `YYYY-MM-DD`, required
- `creator`: relation to users, required
- `job`: relation to active jobs, required
- `type`: relation to `work_records_types`, required
- `location`: freeform text, required
- `work_description`: freeform text, required
- `report_to`: freeform text, optional
- `field_book_number`: text, optional, regex is `YY-NNN` where `NNN` is a
  zero-padded integer from `001-999`
- `field_book_page_number`: positive integer, optional
- `lane_kms`: number, optional, used to inform invoicing for supported work
  record types
- `sub_contractor`: freeform text, optional
- `equipment`: freeform text, optional
- `supplies`: freeform text, optional
- `attachment`: optional single-file PDF upload storing a scanned paper copy of
  the work record
- `attachment_hash`: optional derived SHA256 hash of the uploaded attachment,
  used for duplicate detection
- `created`: system timestamp
- `updated`: system timestamp

Rules:

- `number` is assigned on first save by the backend.
- Once assigned, `number` is immutable and cannot be changed.
- The normal collection API create path rejects client-supplied `number`
  values; backend/internal create paths currently preserve a prefilled number
  when one is already present so controlled import/backfill workflows remain
  possible.
- Future consideration: if imports/backfills need to be distinguished from
  ordinary backend creates, gate prefilled-number preservation on an explicit
  import/backfill signal rather than any non-empty `number`.
- `creator` is immutable after creation.
- Update rules should also prevent direct client changes to `number` and
  `parent_work_record`, because those fields are system-managed.
- `parent_work_record` linkage is system-managed by the `Copy To Tomorrow`
  workflow.
- If `Copy To Tomorrow` is run from a record that already has a
  `parent_work_record`, the new record must point at the same root parent
  rather than pointing at the intermediate record.
- `lane_kms` may be set only when the selected work record type has
  `allow_lane_kms = true`.
- `lane_kms` must be empty when the selected work record type has
  `allow_lane_kms = false`.
- If changing `type` would make an existing `lane_kms` value invalid, the
  update must fail with a validation error directing the user to clear
  `lane_kms` first.
- `attachment` is optional and may only store a single PDF file.
- Attachment uploads are allowed only after the work record exists, so the
  create flow saves the parent first and attachment upload happens as a
  follow-up edit action.
- When a new attachment is uploaded, the backend derives and stores
  `attachment_hash`.
- Duplicate attachments should be rejected by comparing `attachment_hash`
  against other work records.
- Removing the attachment must also clear `attachment_hash`.
- Shared-parent edits are blocked when any linked worker row is approved.
- Adding, removing, or reassigning worker rows is blocked when any linked
  worker row is approved.
- Linked `work_records_consumable_entries` cannot be added, changed, or deleted
  when any linked worker row is approved.
- Append-only notes remain allowed when some or all linked worker rows are
  approved.

Indexes and constraints:

- unique index on `number`
- index on `(job, date)`
- partial unique index on `attachment_hash` where the hash is non-empty

### `work_records_subjects`

Purpose:

- stores the per-person participant rows under a shared work record parent

Fields:

- `id`: PocketBase id
- `work_record`: relation to `work_records`, required
- `uid`: relation to users, required
- `hours_on_site`: number, required
- `hours_travel_time`: number, required
- `distance_travelled_km`: number, optional for now but expected to live here
- `vehicle_type`: text enum, optional for now but expected to live here,
  `company | personal`
- `is_passenger`: boolean, optional
- `company_vehicle_unit_number`: text, optional, regex enforces a positive
  integer greater than `0` and less than `1000`, possibly zero padded
- `approved`: boolean, system managed
- `created`: system timestamp
- `updated`: system timestamp

Rules:

- `uid`, `hours_on_site`, and `hours_travel_time` definitely live here.
- `distance_travelled_km`, `vehicle_type`, `is_passenger`, and
  `company_vehicle_unit_number` should also live here unless follow-up
  stakeholder review identifies a truly shared crew-level exception.
- The number of worker rows for a parent must be greater than or equal to `1`
  and less than or equal to the parent type's `max_subjects` value.
- `uid` is immutable after creation.
- The collection should enforce at most one row per `(work_record, uid)` so the
  same employee is not duplicated within the same shared parent.
- New worker rows may be created only by the parent work record `creator` and
  only while no worker row under the same parent is approved.
- Worker rows may be deleted or replaced only by the parent work record
  `creator` and only while no worker row under the same parent is approved.
- A worker row may be referenced by zero or one `time_entries` row through
  `time_entries.work_record_subject_id`.
- If a time entry references the worker row and the user edits that time
  entry's hours so they no longer equal
  `hours_on_site + hours_travel_time`, a validation error must tell the user to
  either edit the work record worker row to match or create a separate time
  entry.
- `company_vehicle_unit_number` is required when `vehicle_type = company`.
- `company_vehicle_unit_number` must be empty when `vehicle_type = personal`.
- Normal edits are blocked when `approved = true`.
- `approved` is synchronized from the approval state of the related timesheet,
  not directly edited on the worker row.

Implementation note:

- `company_vehicle_unit_number` is stored as text to leave room for future
  non-numeric unit schemes, but the current regex restricts values to positive
  integers in the allowed range.

Indexes and constraints:

- unique index on `(work_record, uid)`
- index on `(uid, created)`

### `work_records_notes`

Purpose:

- append-only note trail attached to a work record

Fields:

- `id`: PocketBase id
- `work_record`: relation to `work_records`, required
- `uid`: relation to users, required
- `note`: freeform text, required
- `created`: system timestamp

Rules:

- `uid` is the note author / creator.
- Notes are append-only.
- Notes cannot be edited or deleted.
- Notes remain creatable after approval.
- Notes are readable by the note author, all worker-row `uid` users on the
  parent work record, the parent work record `creator`, and holders of the
  `report` claim.
- UI should remind users that notes may be visible to clients or used in client
  discussions later.
- The notes UI should expose guidance behind a disclosure icon, such as an `i`
  in a circle.
- The guidance should tell users:
  - be accurate
  - do not swear
  - do not write anything you would not want a client to see

Indexes:

- index on `(work_record, created)`

### `work_records_types`

Purpose:

- master list of work record types

Fields:

- `id`: PocketBase id
- `name`: text, unique, required
- `max_subjects`: number, required, minimum `1`
- `allow_lane_kms`: boolean, required
- `active`: boolean
- `sort_order`: number, optional

Rules:

- work record type is selected on the shared parent work record itself
- `max_subjects` limits how many `work_records_subjects` rows may exist for a
  work record of this type
- `allow_lane_kms` controls whether `work_records.lane_kms` may be set for
  work records of this type
- `max_subjects` applies to new work records and future roster edits;
  decreasing it does not invalidate historical work records that already have
  more worker rows, but those records cannot add workers beyond the current
  limit
- `allow_lane_kms` applies to new work records and future lane-kilometre edits;
  disabling it does not clear or invalidate historical `lane_kms` values, but
  future saves may not add or change `lane_kms` while the selected type
  disallows it
- the implementation should not assume the earlier four-type list is complete
- seeded / admin-managed records should set `active` explicitly

### `work_records_consumables`

Purpose:

- master list of consumable definitions

Fields:

- `id`: PocketBase id
- `name`: text, unique, required
- `input_kind`: text enum, required
  - `number`
  - `boolean`
- `active`: boolean
- `unit_label`: text, optional

Notes:

- most consumables will use `input_kind = number`
- some consumables represent a single yes/no selection and use
  `input_kind = boolean`
- unit labels such as `each`, `km`, `m`, `g`, `roll`, and `bags` should not be
  stored as the quantity value itself
- unit labels belong on the canonical consumable definition so reporting and
  work-record rendering can resolve them directly from
  `work_records_consumables`
- seeded / admin-managed records should set `active` explicitly

### `work_records_type_consumables`

Purpose:

- defines which consumables are allowed for each work record type
- provides the UI metadata and validation allowlist needed to render and
  validate those consumables

Fields:

- `id`: PocketBase id
- `type`: relation to `work_records_types`, required
- `consumable`: relation to `work_records_consumables`, required
- `sort_order`: number, optional
- `active`: boolean

Rules:

- unique index on `(type, consumable)`
- this collection is the source of truth for which consumables appear for a
  given type
- this collection is also the allowlist used by hooks to validate whether a
  `work_records_consumable_entries` row is permitted for a work record's
  current `type`
- if a type has no rows here, the UI shows no consumables for that type
- seeded / admin-managed records should set `active` explicitly

### `work_records_consumable_entries`

Purpose:

- stores actual consumable usage for a specific parent work record
- denormalizes per-record consumable data into its own collection rather than
  forcing all possible consumables into the `work_records` table

Fields:

- `id`: PocketBase id
- `work_record`: relation to `work_records`, required
- `consumable`: relation to `work_records_consumables`, required
- `quantity_number`: number, optional
- `created`: system timestamp
- `updated`: system timestamp

Rules:

- unique index on `(work_record, consumable)`
- the only user-editable stored value on this collection is `quantity_number`
  for numeric consumables; boolean consumables are represented by row
  existence rather than by a separate stored `selected` field
- each row must contain exactly one meaningful stored value, determined by the
  referenced consumable's `input_kind`
- for `input_kind = number`, `quantity_number` is required and must be greater
  than `0`
- for `input_kind = boolean`, `quantity_number` must be empty, and the
  existence of the row itself represents inclusion / checked / yes
- a row must never store a quantity for a boolean consumable
- rows should be omitted entirely when the user leaves a numeric consumable at
  zero or empty, or a boolean consumable unchecked / false / empty
- if a previously stored numeric quantity is cleared to zero / empty, or a
  boolean entry is unchecked, the row should be deleted rather than updated to
  an empty state
- the backend must validate that the referenced `consumable` is allowed for the
  referenced `work_record.type` by checking for a matching
  `work_records_type_consumables` row
- create, update, and delete operations on this collection must fail when any
  linked worker row on the parent work record is approved
- direct list / view access should follow the same visibility rules as the
  parent `work_records` record; normal mutations may be mediated through a
  backend endpoint rather than through unrestricted raw collection updates
- UI rendering should come from `work_records_type_consumables`, while storage
  lives here and references the canonical consumable directly
- if a consumable is not listed for the selected work record type, the worker
  should record it in the work record notes instead
- `input_kind` validation should come from the referenced
  `work_records_consumables` row rather than from
  `work_records_type_consumables`
- this "exactly one meaningful value" rule should be enforced in a backend hook
  so invalid combinations cannot be persisted
- reads and reports that need the consumable identity should use the direct
  `consumable` relation; joins to `work_records_type_consumables` are only
  needed when allowlist or ordering metadata such as `sort_order` is required

Implementation note:

- this collection is the denormalized per-record storage mentioned in
  discussion; it avoids sparse columns, keeps consumables extensible, and keeps
  stored usage facts pointing at the canonical consumable definition rather
  than at a type-specific config row

### Related `time_entries` Change

Phase 2 also requires a schema change on `time_entries`.

New field:

- `work_record_subject_id`: optional relation to `work_records_subjects`

Behavior:

- historical `work_record` text remains in place
- new work-record-aware flows should write and read
  `work_record_subject_id`
- `time_entries.work_record_subject_id` should have a unique index so one
  worker row cannot be linked to more than one time entry
- validation should move from regex-only string validation toward relational
  validation against `work_records_subjects` plus its parent `work_records`
- when linked, `time_entries.uid` must equal the worker row `uid`
- the legacy `work_record` text field can remain searchable for historical
  audit/report compatibility during migration

## Workflow

### Create

- User opens `Work Records` and creates a new shared parent record.
- Default type comes from a user preference that each worker can view and
  update in their own profile settings. If no preference is set, no default is
  applied and the user must select a type manually.
- The create flow also creates at least one worker row.
- The create flow must prevent the worker count from exceeding the selected
  type's `max_subjects`.
- Save requests treat the submitted `subjects` array as the desired final
  worker roster. When the creator removes workers and changes to a type with a
  lower `max_subjects` in the same save, omitted worker rows are pruned inside
  the transaction before the submitted roster is validated. If any later
  validation fails, the parent changes and pruned worker rows roll back together.
- On first successful save, the backend assigns the next `W` number.
- The user will first see the generated number after save succeeds.
- After creation, the number is read-only and can never be edited.

### Copy To Tomorrow

- `Copy To Tomorrow` uses an existing work record as an editable template for
  the next related record.
- It may be run from any work record in a lineage group.
- If saved without edits, the copied record gets the same job, type, location,
  description, report-to, field book data, subcontractor/equipment/supplies,
  consumables, worker roster, and worker hours/travel/vehicle values as the
  source record.
- The copied record always gets:
  - a new independent backend-generated `WYYMM-NNNN` number
  - tomorrow's date by default, but the date is editable before save
  - `parent_work_record` set to the root parent record for the group
- The source job is locked for the copy. The lineage represents a related work
  sequence for the same job, not an immutable clone of the crew or work details.
- The copied worker roster and per-worker values may be edited before save,
  subject to normal permissions, uniqueness, and `max_subjects` validation.
- `lane_kms` is copied when the selected or inherited type allows lane
  kilometres. If the selected or inherited type does not allow lane kilometres,
  copied records must leave `lane_kms` empty.
- The paper-copy PDF attachment is not copied.
- If the source record already has a `parent_work_record`, the new record
  points at the same root parent, not at the source record.
- This is the only UI path that creates a linked work record lineage group.

### Complete

- The creator and any worker can fill shared fields, consumables, and notes
  until the first worker row is approved.
- The creator creates the worker rows and may add or remove workers only
  before the work record is approved.
- The creator may edit all fields for all worker rows until those rows are
  locked by approval.
- Each worker can edit only their own worker row, and only until that row is
  approved.
- The detail page is identical for every authorized viewer and shows the full
  worker roster, including each worker's hours and vehicle usage.
- The edit page also shows the full worker roster. Non-creator worker users
  can update only their own worker fields; other worker rows remain visible
  but read-only.
- After the first worker row is approved, the shared parent fields,
  consumables, and worker roster are locked.
- After the first successful save, the user may upload or replace a scanned
  paper-copy PDF attachment on the work record.
- The detail and list views may expose the stored attachment as a direct file
  link.
- A Turbo-generated print action is required in this phase. It opens the shared
  print-friendly view for the parent work record and includes all worker rows.

### Time Entry Integration

- The expected workflow is to complete the work record first, then create time
  entries from it.
- A work record can exist independently before any time entries are created,
  and there is no dependency that requires a time entry to exist first.
- The work record details page should expose `Create Time Entry` or
  `Create Missing Time Entries` per worker when:
  - the current user is the worker-row `uid`
  - no time entry exists yet with `work_record_subject_id` equal to that
    worker row id
- The main work-record list should expose a create/open time-entry shortcut
  only for the caller's own worker row. Creators who are not also a worker on
  the record should use the record detail page and edit flow to manage the
  shared work record and full worker roster, but cannot create time entries for
  other workers.
- Clicking the action for one worker creates one time entry using:
  - the worker-row `uid`
  - the parent `job`
  - the parent `date`
  - `hours = hours_on_site + hours_travel_time` from the worker row
  - `work_record_subject_id = work_records_subjects.id`
  - `description` derived from parent work record content
- The legacy `time_entries.work_record` text should not be the primary link for
  newly created records.

Hours validation:

- if a time entry references a work record worker row via
  `work_record_subject_id`, its hours must equal that worker row's
  `hours_on_site + hours_travel_time`
- if the user edits the time entry so the hours no longer match, a validation
  error must tell the user to either edit the work record worker row to correct
  the hours or create a separate time entry that is not linked to the work
  record

Recommendation:

- do not make LLM description generation a hard dependency for v1
- use deterministic text composition first, with optional AI assist later

### Approval And Reopen

- When a time entry referencing a worker row becomes part of an approved
  timesheet, mark that worker row approved and prevent normal edits to that
  worker row.
- While any worker row under a parent is approved, the system must also block
  create, update, and delete operations on the shared parent fields, worker
  roster, and linked `work_records_consumable_entries`.
- If a timesheet is unapproved, recalled, or rejected, clear approval on the
  linked worker row.
- Shared fields and the worker roster unlock only when no worker rows under
  that parent remain approved.
- Notes remain append-only regardless of approval state.

## UI

Recommended navigation under `Time Management`:

- `Work Records`

Recommended screens:

- unified work-record list view, filtered to the caller's visible records by
  default
- create/edit form
- detail page

Detail page actions:

- `Edit` when editable
- `Print`
- `Copy To Tomorrow`
- `Create Missing Time Entries`
- `Add Note`
- open the paper-copy PDF attachment when present

Editor requirements:

- shared parent section for date/job/type/location/work description and other
  crew-level fields
- worker rows section that supports one or more participants and is visible to
  every authorized editor
- non-creator worker users can edit only their own worker-row fields while
  other worker rows remain visible and read-only
- the creator can edit all worker rows and can add or remove workers before
  approval
- worker row count limited by the selected work record type's `max_subjects`
- lane kilometres input shown only when the selected work record type has
  `allow_lane_kms = true`
- worker-row validation errors should be surfaced beside the specific worker
  input whenever possible; custom route errors for `work_records_subjects`
  should use indexed keys such as `subjects_0_hours_on_site` so multi-worker
  forms do not hide per-worker failures
- consumables section driven by `work_records_type_consumables`
- numeric inputs for `input_kind = number`
- checkbox inputs for `input_kind = boolean`
- saving consumable entries should write direct `consumable` relations to
  `work_records_consumable_entries`
- no consumables section content when the chosen type has no allowed
  consumables
- the consumables section should include guidance that if a needed consumable
  is not listed, it should be recorded in the notes
- if the user changes the work record type after adding consumables, the UI
  should either clear incompatible entries or block save with a clear message
- if the user changes the work record type after entering `lane_kms` and the
  new type does not allow lane kilometres, the UI should either clear
  `lane_kms` or block save with a clear message
- the notes UI should include a disclosure icon, such as an `i` in a circle,
  that shows note-writing guidance
- once the work record has been created, the editor should expose a paper-copy
  PDF section that can:
  - show the current attachment when present
  - upload or replace a single PDF attachment
  - remove the current attachment
  - surface duplicate-file and validation errors clearly
  - respect normal work-record edit permissions and approval locking

Print route:

- add a dedicated work-record print route and `Print` action on the detail page
- model this flow on the existing purchase-order print experience: open a
  print-friendly view in a new tab/window and trigger the browser print dialog
- the print-friendly work-record layout should closely match the existing paper
  work record format rather than inventing a new visual structure
- the print view must be the same for every authorized viewer
- the print view must include all workers on the work record and display each
  worker's hours and vehicle usage separately in a compact, simple section
- printing to paper and saving to PDF should both use that print-friendly route

Job details page:

- surface all work records for that job with at least one worker row linked to
  an approved timesheet
- paginate the approved work-record list when the job has many work records
- move this job-scoped approved-record surface onto a first-class job tab
  beside `Active POs`

Main list behavior:

- `/time/work-records` is the single navigation entry point for work records
- by default, it shows records where the caller is either a worker-row `uid`
  or the `creator`
- each visible work record appears once, even when the caller is both creator
  and a worker row participant
- list rows should show the full worker roster in compact form, including each
  worker's open/linked/approved status and a clear marker for the caller's own
  worker row
- list rows should show a `Creator` badge when the caller created the work
  record, because creator visibility is distinct from worker-row participation
- list-level create/open time-entry actions should be scoped only to the
  caller's own worker row; record-level management happens by opening the
  detail page and then editing the work record
- holders of the `report` claim get a top-right scope toggle:
  `Mine/Created` vs `All`
- the default scope for report holders is `Mine/Created`
- the main page supports lightweight list browsing and search rather than
  full reporting filters
- advanced filtering and report-style queries live on the separate
  `/reports/work-records` page in v1
- consolidating advanced filtering back onto the main page remains a valid
  later enhancement, but is not required for this phase

## Reporting

Reporting should cover operations and billing readiness, but not PDF/export in
this phase.

Minimum useful reports:

- work records with no worker time entries
- work record workers with no referencing time entry
- work record workers not referenced by an approved timesheet
- record-shaped incomplete reports must include partial multi-worker records:
  the unlinked report includes a parent when any worker row has no referencing
  time entry, and the pending approval report includes a parent when any worker
  row is not approved, even if other worker rows on the same parent are already
  linked or approved
- work records by job, employee, date range, and type
- consumable usage by consumable across work records, without requiring
  reporting queries to join through the type-allowlist table just to resolve
  the consumable identity

Future reporting work:

- lane kilometre usage by job, date range, and type for billing support; this
  should apply only to work record types where `allow_lane_kms = true`

## Migration And Compatibility

### Existing Behavior To Preserve Initially

- existing `time_entries.work_record` values must remain valid for historical
  data
- once linked-only rollout is live, editing an existing time entry that still
  relies on legacy `time_entries.work_record` text requires linking it to a
  real `work_record_subject_id` before the save can succeed
- the old read-only `/time/work-records` audit/search surface is replaced at
  rollout by the first-class work records list, filtered according to the
  caller's visibility
- legacy prefixes such as `F`, `K`, and `Q` should remain searchable for
  historical records in reporting and other historical views that still read
  legacy `time_entries.work_record` text

### Uniqueness Constraint Change

- The existing system enforces work record string uniqueness per timesheet
  only; the same string cannot appear on two time entries in the same
  timesheet.
- The revised model does not enforce one time entry per shared work record.
- Instead, it enforces one time entry per worker row: each
  `work_records_subjects` record can be referenced by at most one time entry
  via `time_entries.work_record_subject_id`.
- This is a better match for the real crew workflow and avoids over-tightening
  the data model around a false one-to-one assumption.

### Approval Hook Integration

- The existing timesheet approval/unapproval hooks must be extended to fan out
  to `work_records_subjects`.
- When a timesheet is approved, all worker rows referenced by that timesheet's
  time entries (via `work_record_subject_id`) must have their `approved` field
  set to `true`.
- When a timesheet is unapproved, recalled, or rejected, the corresponding
  worker rows must have `approved` set back to `false`.
- Shared parent edit locking should be derived from whether any child worker
  row under that parent remains approved.
- Worker roster locking should use the same derived parent lock state, so
  worker rows cannot be added, removed, or reassigned while any child worker
  row remains approved.

### Recommended Transition Strategy

Rollout should avoid breaking historical reporting:

- add first-class work record collections
- add first-class `work_records_subjects`
- add `time_entries.work_record_subject_id`
- keep legacy `time_entries.work_record` text in place for historical data
- require legacy time entries to link to a real `work_record_subject_id`
  before edited rows can be saved after rollout
- replace the old work-record audit/search page with the first-class work
  records list, filtered by the caller's visibility
- migrate new work-record-aware UI and validation to
  `work_record_subject_id`
- update reports incrementally rather than requiring a big-bang rewrite

This allows historical data to remain readable while new first-class records
become the required editing and operational workflow.

## Remaining Open Questions

- Should `sub_contractor` stay freeform in v1, or should it reference a vendor
  list?
- Should users be allowed to create records after the work date without special
  permissions or just with audit logging?
- Which current type list is authoritative for launch: four, nine, or fully
  admin-configurable?
- Do any travel/vehicle fields need both a shared crew-level field and a
  per-person field, or should they live only on `work_records_subjects`?
- Do we want an explicit stored parent status field, or is a derived
  `Draft/Partially Approved/Fully Approved` UI status enough for v1?
- Should initial `max_subjects` values mirror the current paper form variants,
  or should all types launch with the same conservative default until usage is
  observed?

## Revised Phased Plan

### Phase 1: Completed Decisions

- new work records use backend-generated `W` numbers
- work records are shared parent records with child `work_records_subjects`
- `creator` stays on `work_records`; `uid` moves to worker rows
- `uid`, `hours_on_site`, and `hours_travel_time` move to
  `work_records_subjects`
- per-person travel / vehicle fields should also move to
  `work_records_subjects`
- `time_entries.work_record_subject_id` becomes the new relational link
- `time_entries.work_record` stays for historical compatibility
- work record type lives on `work_records`
- `work_records_types.max_subjects` limits the number of worker rows per work
  record and must be at least `1`
- `work_records.lane_kms` supports invoicing for selected work record types,
  controlled by `work_records_types.allow_lane_kms`
- `work_records_consumable_entries` reference
  `work_records_consumables` directly, while `work_records_type_consumables`
  remain the type-specific allowlist and ordering source
- related work records come only from `Copy To Tomorrow`, each with its own
  independent number
- worker approval is system-managed and driven by timesheet approval state
- scanned paper-copy PDF attachment support is in scope
- Turbo-generated browser printing is in scope through a print-friendly route
  that shows the complete worker roster

### Phase 2: Data Model And Backend

- add `work_records`
- add `work_records_subjects`
- add `work_records_notes`
- add `work_records_types`
- add `work_records_consumables`
- add `work_records_type_consumables`
- add `work_records_consumable_entries`
- add `time_entries.work_record_subject_id`
- add backend number generation for `W` work records
- add CRUD endpoints and permission rules
- add append-only notes support
- add consumable validation based on direct `consumable` references, type
  allowlists, and input kind
- add approval-aware hooks so `work_records_consumable_entries` cannot be
  created, updated, or deleted while any child worker row is approved
- add validation enforcing `work_records_types.max_subjects` when creating or
  updating worker rows and when changing a work record's type, without making
  historical over-limit rosters block type configuration changes
- add validation enforcing `work_records_types.allow_lane_kms` when setting
  `work_records.lane_kms` and when changing a work record's type, without
  making historical lane-kilometre values block type configuration changes
- add validation on `work_records.type` updates so incompatible existing
  consumable entries, excessive worker counts, or invalid `lane_kms` values
  block save until resolved
- add approval/unapproval synchronization with timesheet workflow by extending
  existing timesheet approval hooks to fan out to referenced worker rows
- add unique index on `time_entries.work_record_subject_id` to enforce the
  one-time-entry-per-worker relationship, with a hook to surface constraint
  violations to the user
- update time-entry validation to support participant-linked work records
- add optional paper-copy PDF attachment support on `work_records`, including
  duplicate detection by derived hash and dedicated upload/remove endpoints

### Phase 3: UI

- replace the current read-only audit/search work-record surface with the
  first-class work records list, with contents filtered according to the
  caller's visibility, plus real work record create/edit/detail flows
- add list, create, edit, detail, and copy-to-tomorrow flows
- add multi-worker editor UI
- add `max_subjects` handling to the type-driven worker row editor
- add type-driven `lane_kms` input visibility and validation
- add consumables UI driven by selected type
- add per-worker create-time-entry action
- add note guidance disclosure UI
- add paper-copy PDF attachment UI on the edit/detail/list surfaces
- add a print-friendly work-record route and detail-page `Print` action with
  all worker rows included
- move job-level approved work records into a first-class job tab

### Phase 4: Reporting

- add incomplete/unlinked work-record reports
- add worker-level "missing time entry" and "not yet approved" reports
- add job-level visibility
- keep historical reporting surfaces working with legacy
  `time_entries.work_record` text until each report is migrated to the new
  first-class model
- defer lane-kilometre-specific billing reports to a later reporting pass; the
  first-class model and validation still store `work_records.lane_kms` only for
  work record types where `allow_lane_kms = true`

### Phase 5: Print-Friendly Output

- add a dedicated `/time/work-records/{id}/print` route
- add a `Print` action from the work-record detail page
- follow the existing purchase-order print pattern: open a print-friendly page
  in a new tab/window and trigger the browser print dialog automatically
- use that route for both printing to a physical printer and saving to PDF
- design the printable layout to closely match the existing paper work record
  format used in the field and office workflow

### Phase 6: Migration Cleanup

- review whether legacy `time_entries.work_record` can eventually become
  historical-only
- decide how much historical backfill is worth doing between old text work
  record references and new first-class records
- simplify old regex-only work-record assumptions once the participant-linked
  relational flow is fully adopted

## Recommended Next Step

The next concrete step is to turn the revised Phase 2 into an implementation
checklist: PocketBase collections, migration order, validation changes on
`time_entries`, parent-vs-worker approval hooks, and `Copy To Tomorrow`
behavior for worker-row cloning.
