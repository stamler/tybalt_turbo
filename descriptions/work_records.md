# Work Records

This document turns stakeholder notes and follow-up decisions into a proposed
product and implementation spec for first-class work records in Turbo.

It covers:

- workflow findings from the current paper/Excel process
- resolved product decisions
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

The agreed direction is:

- add a first-class `work_records` collection
- generate all new work record numbers on the backend with a `W` prefix
- keep one work record per person per day per job
- attach work record type to the work record, not the note
- track consumable usage per work record through normalized supporting
  collections (`work_records_consumables`, `work_records_type_consumables`,
  `work_records_consumable_entries`), with consumable visibility driven by the
  selected work record type
- add `time_entries.work_record_id` as the new relation to `work_records`
- enforce that a time entry referencing a work record has hours matching the
  work record's `hours_on_site + hours_travel_time`; mismatches produce a
  validation error directing the user to edit the work record or create a
  separate time entry
- preserve legacy `time_entries.work_record` text for existing historical data
- store notes as append-only records
- support child work records only through a dedicated `Copy To Tomorrow`
  workflow
- lock work records when the referencing timesheet is approved; unlock them
  when it is unapproved, recalled, or rejected
- defer PDF and printing/export work for now

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
  though the business prefers one record per day.
- Zena chases incomplete or missing work records after the work returns.
- Once complete, Zena saves the record as PDF and places it where Sudha can use
  it during invoice-summary preparation.
- The work record number is also recorded on the timesheet.

### Core Domain Rules

- A work record belongs to exactly one worker.
- If three people work together, there should be three work records.
- If one worker is on the same job for multiple days, each day should have its
  own work record.
- Daily separation matters because some charges, such as meals and vehicles,
  are billable per day.
- Work records track time plus consumables.
- The four known types are `Survey`, `General`, `Drilling`, and `Field Test`.
- Type mostly affects which consumables are shown; the rest of the workflow is
  broadly the same.
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

### Findings From The Current Codebase

- Turbo already stores a `work_record` text field on `time_entries`.
- Time entry validation currently expects a string matching
  `^[FKQ][0-9]{2}-[0-9]{3,4}(-[0-9]+)?$` when a job is present.
- Timesheet validation currently prevents the same work record string from
  appearing on more than one time entry in the same timesheet.
- Turbo already has a read-only `/time/work-records` audit surface that groups
  and searches work record strings pulled from `time_entries`.
- The nav already contains a `Work Records` item, but today it is an audit view,
  not a create/edit workflow.

## Resolved Product Decisions

### Numbering

- All new first-class work records will use backend-generated numbers beginning
  with `W`.
- Number generation should follow the same general pattern as purchase order
  numbering, but with a work-record prefix.
- The draft format remains `W<YYMM-XXXX>` for parent records.
- Child records created through `Copy To Tomorrow` will keep the same root
  parent number with a dash suffix, for example `W2604-0123-1`.
- The number regex is `WYYMM-NNNN` for parent records and `WYYMM-NNNN-N` for
  child records.
- `Copy To Tomorrow` may be run from either a parent record or a child record.
- When copying from a child record, the new record must use the same root parent
  number and the next sequential child suffix rather than appending a second
  suffix.
- Users will not manually enter or reserve work record numbers in the new flow.

### Work Record Identity On Time Entries

- Existing `time_entries.work_record` text values stay in place for historical
  records.
- After rollout, new functionality should stop relying on
  `time_entries.work_record` as the primary link.
- A new `time_entries.work_record_id` relation will reference
  `work_records.id`.
- This relation is the only first-class link between time entries and work
  records.
- New validation and workflow logic should use `work_record_id`.
- Legacy reports and search surfaces may continue to read
  `time_entries.work_record` until they are migrated.

### Type Ownership

- Work record type lives on the `work_records` record.
- Notes do not carry type.

### Granularity

- The business distinction for work records is one work record per person per
  day per job.
- Multi-day records are represented as a parent plus child day records.
- Child work records are created only through `Copy To Tomorrow`.
- `Copy To Tomorrow` may start from any record in the group, but child numbering
  always continues from the root parent sequence.
- The UI should not allow arbitrary manual creation of child numbers.

### Export Scope

- PDF generation, printing, and client-facing export are out of scope for the
  current implementation.

## Recommended Product Model

### Main Entity Rules

- A work record is a first-class record, not just a string on a time entry.
- A work record belongs to one employee (`uid`) and one job.
- A work record is for one calendar day.
- A work record may optionally reference a parent work record when created from
  `Copy To Tomorrow`.
- A work record may be referenced by zero or one time entry through
  `time_entries.work_record_id`. This means work_record_id can have unique index. But the error must be surfaced by a hook to catch and report to the user since otherwise it would be buried
- A work record can exist before, during, or after field work.

### Approval Model

- `approved` is a boolean on `work_records`
- it is never manually set by users
- it becomes `true` only when the manager approves the `time_sheets` record
  that contains the `time_entries` record referencing that work record
- when `approved = true`, the work record becomes locked for normal editing
- if the related timesheet becomes unapproved, recalled, or rejected, the
  corresponding work record's `approved` field should be set back to `false`
- approval and unapproval of `work_records.approved` is triggered by the
  timesheet approval/unapproval hooks, which must fan out to work records
  referenced by the timesheet's time entries
- append-only notes remain allowed regardless of approval state

### Delete Rules

- Deletion of work records is out of scope for the UI in this phase.
- A work record may only be deleted if no time entry references it via
  `work_record_id`.
- Deletion is restricted to superusers in the PocketBase backend.

### Permissions

- Any authenticated user can create a work record for themselves. In that case,
  `creator` and `uid` are both set to the current user.
- Holders of the `work_record` claim can create work records on behalf of other
  users.
- Users without the `work_record` claim cannot create work records for other
  users.
- During creation, the UI may ask whether the record is for "you" or "someone
  else", but the "someone else" path must only be available to holders of the
  `work_record` claim.
- `uid` and `creator` are immutable after creation and cannot be changed.
- Updates are restricted to the `creator` or the `uid` of the work record.

## Phase 2 Data Model

Phase 2 should introduce first-class collections plus the new time-entry
relation.

### `work_records`

Purpose:

- the main record for one worker, one day, one job

Fields:

- `id`: PocketBase id
- `number`: text, unique, backend generated, required
- `parent_work_record`: optional self-relation used only for child records and
  always points at the root parent record
- `date`: text, `YYYY-MM-DD`, required
- `uid`: relation to users, required
- `creator`: relation to users, required
- `job`: relation to active jobs, required
- `type`: relation to `work_records_types`, required
- `location`: freeform text, required
- `work_description`: freeform text, required
- `report_to`: freeform text, optional
- `hours_on_site`: number, required
- `hours_travel_time`: number, required
- `distance_travelled_km`: number, required
- `field_book_number`: text, optional, regex is YY-NNN where NNN is zero padded integer from 001-999 and YY is the last two digits of the year, so zero padded integer
- `field_book_page_number`: positive integer, optional
- `sub_contractor`: freeform text, optional
- `equipment`: freeform text, optional
- `supplies`: freeform text, optional
- `vehicle_type`: text enum, required, `company | personal`
- `is_passenger`: boolean, optional
- `company_vehicle_unit_number`:  text, optional, regex enforces a positive integer
  greater than 0 and less than 1000, possibly zero padded
- `approved`: boolean, system managed
- `created`: system timestamp
- `updated`: system timestamp

Rules:

- `number` is assigned on first save by the backend
- once assigned, `number` is immutable and cannot be changed
- `uid` and `creator` are immutable after creation
- parent/child linkage is system-managed by the `Copy To Tomorrow` workflow
- a child record must inherit the same `job`, `uid`, and root parent number
  base
- if `Copy To Tomorrow` is run from a child record, the backend must resolve the
  root parent and assign the next sequential child suffix under that root
- if a time entry references the work record and the user edits that time
  entry's hours so they no longer equal `hours_on_site + hours_travel_time`,
  a validation error must tell the user to either edit the work record to
  match or create a separate time entry
- `company_vehicle_unit_number` is required when `vehicle_type = company`
- `company_vehicle_unit_number` must be empty when `vehicle_type = personal`
- normal edits are blocked when `approved = true`
- append-only notes remain allowed when `approved = true`
- `approved` is synchronized from the approval state of the related timesheet,
  not directly edited on the work record

Implementation note:

- `company_vehicle_unit_number` is stored as text to leave room for future
  non-numeric unit schemes, but the current regex restricts values to positive
  integers in the allowed range

Indexes and constraints:

- unique index on `number`
- index on `(uid, date)`
- index on `(job, date)`
- unique business-rule check should prevent more than one work record for
  the same `(uid, job, date)` combination

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

- notes are append-only
- notes cannot be edited or deleted
- notes remain creatable after approval
- UI should remind users that notes may be visible to clients or used in client
  discussions later
- the notes UI should expose guidance behind a disclosure icon, such as an `i`
  in a circle
- the guidance should tell users:
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
- `active`: boolean, default true
- `sort_order`: number, optional

Rules:

- work record type is selected on the work record itself
- `General` has no consumables in seeded data, though that is a data rule, not
  a schema rule

### `work_records_consumables`

Purpose:

- master list of consumable definitions

Fields:

- `id`: PocketBase id
- `name`: text, unique, required
- `input_kind`: text enum, required
  - `number`
  - `boolean`
- `active`: boolean, default true

Notes:

- most consumables will use `input_kind = number`
- some consumables represent a single yes/no selection and use
  `input_kind = boolean`
- unit labels such as `each`, `km`, `m`, `g`, `roll`, and `bags` should not be
  stored as the quantity value itself
- unit labels belong on the type-specific allowlist record, because the same
  consumable concept could theoretically be shown differently in different
  contexts

### `work_records_type_consumables`

Purpose:

- defines which consumables are allowed for each work record type
- provides the UI metadata needed to render those consumables

Fields:

- `id`: PocketBase id
- `type`: relation to `work_records_types`, required
- `consumable`: relation to `work_records_consumables`, required
- `unit_label`: text, optional
- `sort_order`: number, optional
- `active`: boolean, default true

Rules:

- unique index on `(type, consumable)`
- this collection is the source of truth for which consumables appear for a
  given type
- if a type has no rows here, the UI shows no consumables for that type
- `General` is expected to have no rows here in initial seed data

### `work_records_consumable_entries`

Purpose:

- stores actual consumable usage for a specific work record
- denormalizes per-record consumable data into its own collection rather than
  forcing all possible consumables into the `work_records` table

Fields:

- `id`: PocketBase id
- `work_record`: relation to `work_records`, required
- `type_consumable`: relation to `work_records_type_consumables`, required
- `quantity_number`: number, optional
- `selected`: boolean, optional
- `created`: system timestamp
- `updated`: system timestamp

Rules:

- unique index on `(work_record, type_consumable)`
- for `input_kind = number`, `quantity_number` is required and must be positive
- for `input_kind = boolean`, `selected = true` represents inclusion and
  `quantity_number` must be empty
- rows should be omitted entirely when the user leaves a numeric consumable at
  zero or a boolean consumable unchecked
- UI rendering should come from `work_records_type_consumables`, while storage
  lives here

Implementation note:

- this collection is the denormalized per-record storage mentioned in
  discussion; it avoids sparse columns and keeps consumables extensible

### Related `time_entries` Change

Phase 2 also requires a schema change on `time_entries`.

New field:

- `work_record_id`: optional relation to `work_records`

Behavior:

- historical `work_record` text remains in place
- new work-record-aware flows should write and read `work_record_id`
- validation should move from regex-only string validation toward relational
  validation against `work_records`
- the legacy `work_record` text field can remain searchable for historical
  audit/report compatibility during migration

## Workflow

### Create

- User opens `Work Records` and creates a new record.
- Default type may come from a user preference if configured.
- On first successful save, the backend assigns the next `W` number.
- The user will first see the generated number after save succeeds.
- After creation, the number is read-only and can never be edited.

### Copy To Tomorrow

- `Copy To Tomorrow` pre-populates the editor with an existing work record.
- It may be run from either a parent work record or a child work record.
- The copied record gets:
  - a new backend-generated number
  - tomorrow's date by default, but the date is editable before save
  - `parent_work_record` set to the root parent record for the group
- If the source record is already a child, the backend resolves the root parent
  and assigns the next sequential child suffix under that root, rather than
  appending a second suffix to the source number.
- The unique `(uid, job, date)` constraint still applies, so the user must
  choose a date that does not conflict with an existing record.
- This is the only UI path that can create a child work record number.

### Complete

- The assignee can fill time, travel, location, description, consumables, and
  notes.
- No PDF/export action is required in this phase.

### Time Entry Integration

- The work record details page should expose `Create Time Entry` when:
  - the current user is the record owner
  - no time entry exists yet with `work_record_id` equal to the current work
    record id
- Clicking the button creates one time entry using:
  - `uid`
  - `job`
  - `date`
  - `hours = hours_on_site + hours_travel_time`
  - `work_record_id = work_records.id`
  - `description` derived from work record content
- The legacy `time_entries.work_record` text should not be the primary link for
  newly created records.

Hours validation:

- if a time entry references a work record via `work_record_id`, its hours
  must equal the work record's `hours_on_site + hours_travel_time`
- if the user edits the time entry so the hours no longer match, a validation
  error must tell the user to either edit the work record to correct the
  hours or create a separate time entry that is not linked to the work record

Recommendation:

- do not make LLM description generation a hard dependency for v1
- use deterministic text composition first, with optional AI assist later

### Approval And Reopen

- When a time entry referencing the work record becomes part of an approved
  timesheet, mark the work record approved and prevent normal edits.
- If the timesheet is unapproved, recalled, or rejected, clear approval and
  allow edits again.
- Notes remain append-only regardless of approval state.

## UI

Recommended navigation under `Time Management`:

- `Work Records`
- `My Work Records`
- `Search`

Recommended screens:

- list view for the current user
- organization search/reporting view for admins
- create/edit form
- detail page

Detail page actions:

- `Edit` when editable
- `Copy To Tomorrow`
- `Create Time Entry` when eligible
- `Add Note`

Editor requirements:

- type selector on the work record
- consumables section driven by `work_records_type_consumables`
- numeric inputs for `input_kind = number`
- checkbox inputs for `input_kind = boolean`
- no consumables section content when the chosen type has no allowed
  consumables
- the notes UI should include a disclosure icon, such as an `i` in a circle,
  that shows note-writing guidance

Job details page:

- surface all work records for that job that belong to approved timesheets

## Reporting

Reporting should cover operations and billing readiness, but not PDF/export in
this phase.

Minimum useful reports:

- work records with no referencing time entry
- work records not referenced by an approved timesheet
- work records by job, employee, date range, and type

## Migration And Compatibility

### Existing Behavior To Preserve Initially

- existing `time_entries.work_record` values must remain valid for historical
  data
- existing read-only audit screens should keep working during rollout
- legacy prefixes such as `F`, `K`, and `Q` should remain searchable for
  historical records

### Uniqueness Constraint Change

- The existing system enforces work record string uniqueness per timesheet
  only; the same string cannot appear on two time entries in the same
  timesheet.
- The new model enforces a global one-to-one constraint: each work record can
  be referenced by at most one time entry via `work_record_id` (unique index).
- This is a tightening of the constraint. Historical data should be reviewed to
  confirm no existing work record string appears on multiple time entries
  across different timesheets. But historical data uses a different column and a different work record number scheme (only in time_entries) so this shouldn't be an issue.

### Approval Hook Integration

- The existing timesheet approval/unapproval hooks must be extended to fan out
  to `work_records`. When a timesheet is approved, all work records referenced
  by that timesheet's time entries (via `work_record_id`) must have their
  `approved` field set to `true`. When a timesheet is unapproved, recalled, or
  rejected, the corresponding `approved` fields must be set back to `false`.

### Recommended Transition Strategy

Rollout should avoid breaking historical reporting:

- add first-class work record collections
- add `time_entries.work_record_id`
- keep legacy `time_entries.work_record` text in place for historical data
- migrate new work-record-aware UI and validation to `work_record_id`
- update reports incrementally rather than requiring a big-bang rewrite

This allows historical data and new first-class records to coexist while the
rest of the system catches up.

## Remaining Open Questions

- Should `sub_contractor` stay freeform in v1, or should it reference a vendor
  list?
- Should users be allowed to create records after the work date without special
  permissions or just with audit logging?

## Revised Phased Plan

### Phase 1: Completed Decisions

- new work records use backend-generated `W` numbers
- `time_entries.work_record_id` becomes the new relational link
- `time_entries.work_record` stays for historical compatibility
- work record type lives on `work_records`
- work record granularity is one person per day per job
- child work records come only from `Copy To Tomorrow`
- `work_records.approved` is a system-managed boolean driven by timesheet
  approval state
- PDF/export is out of scope for now

### Phase 2: Data Model And Backend

- add `work_records`
- add `work_records_notes`
- add `work_records_types`
- add `work_records_consumables`
- add `work_records_type_consumables`
- add `work_records_consumable_entries`
- add `time_entries.work_record_id`
- add backend number generation for `W` work records
- add CRUD endpoints and permission rules
- add append-only notes support
- add consumable validation based on type allowlists and input kind
- add approval/unapproval synchronization with timesheet workflow by extending
  existing timesheet approval hooks to fan out to referenced work records
- add unique index on `time_entries.work_record_id` to enforce one-to-one
  relationship, with a hook to surface constraint violations to the user
- update time-entry validation to support relational work records

### Phase 3: UI

- replace the current read-only audit-only work-record surface with real work
  record create/edit/detail flows
- add list, create, edit, detail, and copy-to-tomorrow flows
- add consumables UI driven by selected type
- add create-time-entry action
- add note guidance disclosure UI

### Phase 4: Reporting

- add incomplete/unlinked work-record reports
- add job-level visibility
- migrate or expand the existing work-record audit/search screens to work with
  first-class records and historical text records together

### Phase 5: Migration Cleanup

- review whether legacy `time_entries.work_record` can eventually become
  historical-only
- decide how much historical backfill is worth doing between old text work
  record references and new first-class records
- simplify old regex-only work-record assumptions once the relational flow is
  fully adopted

## Recommended Next Step

The next concrete step is to turn Phase 2 into an implementation checklist:
PocketBase collections, migration order, validation changes on `time_entries`,
and the `Copy To Tomorrow` root-parent/next-suffix number-generation rules.
