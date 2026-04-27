# Invoiced Amount Tracking for Job POs, Expenses Feature (Phase 2)

TO ADD LATER FEB 25 2026:

- STORE THE INVOICE NUMBER FROM THE SUBCONTRACTOR AS WELL FOR CROSS REFERENCING
- THESE RECORDS CAN ONLY BE CREATED BY HOLDERS OF THE 'invoicing' CLAIM
- THESE RECORDS CAN BE VIEWABLE BY REPORT CLAIM HOLDERS
- CHECK WITH ANDREW BROWN OR WENDY FOR OTHER FIELDS

Status: Draft (living document)  
Last updated: 2026-02-18

## Context

On job details (`/jobs/{id}/details#pos`), the current `Active POs` tab only shows Active purchase orders. We need a second tab for invoicing workflows that shows all job-referencing POs that can legitimately have client billing recorded against them.

This feature is for tracking what has already been billed to the client for work tied to a PO, including:

- when the client was invoiced
- full invoiced amount (including taxes/fees)
- invoice number
- who recorded the entry
- when the entry was recorded
- optional freeform note

## Scope

## In scope

- Add `Invoicing POs` tab on job details.
- Show eligible POs for the job (rules below).
- Allow creating invoicing records against a specific PO.
- Display invoicing record history per PO.
- Provide per-PO invoiced totals in the tab.

## Out of scope (phase 1)

- Editing/deleting invoicing records (audit-safe append-only first).
- Automatic sync to accounting systems.
- Client-facing invoice document generation.
- Replacing existing client notes.

## Eligibility Rules (Normative)

A PO is eligible for invoicing records only if all are true:

1. `purchase_orders.job` is set to the current job.
2. PO kind is `project` (job-linked POs).
3. PO status is `Active` or `Closed`.
4. PO status is never `Cancelled` or `Unapproved`.

Rationale:

- `Closed` remains valid because expenses may be realized and billed later.
- `Cancelled` means no realized expense and should not be billable.
- `Unapproved` should not be treated as billable.

## Product Decisions

## D1. Use a dedicated collection, not `client_notes`

Recommendation: create `po_invoicing_records` collection.

Why:

- Invoicing needs structured financial fields (`amount`, `invoice_number`, `invoiced_on`) that are not note-native.
- Audit requirements are stronger than generic notes.
- Keeps `client_notes` semantics clean and avoids overloading status-change note behavior.

## D2. Append-only records (phase 1)

- Creation allowed.
- Update/delete disallowed in phase 1.
- Corrections handled by adding compensating entries (if needed later, add explicit adjustment flow).

This preserves clear audit history (`who`, `when`).

## D3. Amount semantics

- `amount` means gross client invoice amount (taxes + fees included).
- Must be positive.
- Stored as currency amount with 2-decimal precision semantics.

## D4. Invoice number

- Dedicated `invoice_number` field on each record.
- Optional in phase 1 (can be made required later if operations demands).

## UX Specification

## Job Details Tabs

Current:

- `Time`
- `Expenses`
- `Active POs`

Proposed:

- `Time`
- `Expenses`
- `Active POs`
- `Invoicing POs`

`Active POs` remains unchanged.

## Invoicing POs list content

Each PO row should show:

- PO number
- PO status (`Active`/`Closed`)
- PO date
- vendor
- division
- PO total
- invoiced total (sum of records)
- invoice record count
- last invoiced date (max `invoiced_on`)
- action: `Add Invoice Record`

Optional row expand/collapsible section:

- list of invoicing records (newest first)

## Add Invoice Record form

Fields:

- `Invoiced On` (required date)
- `Amount` (required currency, full billed amount)
- `Invoice Number` (optional text)
- `Note` (optional freeform text)

System-managed (not user-editable):

- `Recorded By` (current auth user)
- `Recorded At` (server timestamp)

## Data Model (Proposed)

Collection: `po_invoicing_records`

Fields:

- `id` (system)
- `purchase_order` (relation -> `purchase_orders`, required)
- `job` (relation -> `jobs`, required; denormalized for query speed)
- `client` (relation -> `clients`, optional but recommended for future reporting)
- `invoiced_on` (date, required)
- `amount` (number, required; > 0)
- `invoice_number` (text, optional, max ~100)
- `note` (text, optional, max ~1000)
- `uid` (relation -> `_pb_users_auth_`, required; recorder)
- `created` (autodate; recorded timestamp)
- `updated` (autodate)

Recommended indexes:

- `(job, purchase_order, invoiced_on DESC, created DESC)`
- `(purchase_order, created DESC)`
- `(job, created DESC)`

## API / Route Plan (Proposed)

## Job tab endpoints

- `GET /api/jobs/{id}/invoicing_pos/summary`
- `GET /api/jobs/{id}/invoicing_pos/list`

Behavior mirrors existing `.../pos/summary` and `.../pos/list` patterns, but uses invoicing eligibility rules and includes invoicing aggregates.

## PO invoicing record endpoints

- `GET /api/purchase_orders/{id}/invoicing_records`
- `POST /api/purchase_orders/{id}/invoicing_records`

`POST` payload:

- `invoiced_on`
- `amount`
- `invoice_number` (optional)
- `note` (optional)

Server derives:

- `uid` from auth
- `job` and `client` from PO/job linkage

## Validation Rules (Normative)

On create:

1. Auth required.
2. PO must exist.
3. PO must meet eligibility rules (project kind + job-linked + status in Active/Closed).
4. Route PO id must match payload PO id (or omit payload PO id entirely and trust route param).
5. `invoiced_on` required, valid date (and not future-dated in phase 1).
6. `amount` required, `> 0`.
7. `invoice_number` length constrained.
8. `note` length constrained.
9. `uid`/`created` must be system-set only.

## Permissions (Proposed)

Initial recommendation:

- View: any authenticated user who can view the parent job details.
- Create: authenticated users with financial editing authority (same gating style used for PO mutations, e.g. `requireExpensesEditing(..., "purchase_orders")`).
- Update/Delete: disallow in phase 1.

## Implementation Notes (Codebase Fit)

- Existing job details tab framework (`JobDetailTab`) can be reused with new summary/list endpoints.
- Existing active PO SQL (`app/routes/job_pos.sql`, `app/routes/job_po_summary.sql`) can be cloned/adapted for invoicing-eligible PO logic.
- New `PO Invoicing` UI component should mirror `POsTabContent` style and add record list + create action.
- Keep `client_notes` untouched for phase 1.

## Reporting / Derived Metrics

Per PO:

- `invoiced_total = SUM(amount)`
- `invoice_count = COUNT(records)`
- `last_invoiced_on = MAX(invoiced_on)`

Job-level (optional phase 1.5):

- total invoiced across eligible POs in tab summary.

## Test Plan

## Backend

- eligibility enforcement by kind/status/job linkage
- create succeeds for `Active` + `Closed`
- create fails for `Cancelled` + `Unapproved`
- create fails for non-project kind or missing job
- amount/date validation
- recorder identity and recorded timestamp are server-assigned
- list/summary aggregates are correct

## UI

- `Invoicing POs` tab appears and loads
- row action opens/uses create form
- successful create updates row aggregates/history
- validation errors render correctly

## Rollout

1. Add collection + migrations + API handlers + tests.
2. Add UI tab + components + integration wiring.
3. Seed fixture data for happy-path and invalid-state tests.
4. Release behind normal auth/claim controls (no feature flag required unless requested).

## Open Questions

1. Should `invoice_number` be mandatory at creation time?
2. Should future-dated `invoiced_on` be allowed?
3. Do we need explicit reversal entries (`negative` amounts) in phase 1, or keep strictly positive and add a later correction workflow?
4. Should we add export/report endpoints immediately, or defer until users validate workflow?
5. Should this appear in any client-level notes/audit feed, or stay PO-local only?

## Stakeholders

Verify spec with Sudha, Andrew, Scott, Sarah

## Change Log

- 2026-02-18: Initial draft created from product discussion and current codebase behavior.
