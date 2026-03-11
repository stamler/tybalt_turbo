# Legacy PO Create/Edit Flow

## Purpose

This feature provides a short-lived hidden workflow for manually entering and correcting purchase orders that originated in the legacy system during the phase 2 transition.

It does not relax the normal Turbo purchase order workflow. Legacy PO create/edit uses dedicated backend endpoints and a hidden UI.

## Feature Gates

The flow is enabled only when both of these are true:

- the authenticated user has the `legacy_po_create_update` claim
- `app_config.key = "purchase_orders"` has `value.enable_legacy_po_create_update = true`

The backend enforces both checks.

## Hidden Routes

The legacy editor is intentionally not linked from the main nav.

Hidden UI routes:

- `/pos/legacy/add`
- `/pos/legacy/[poid]/edit`

Hidden backend endpoints:

- `GET /api/purchase_orders/legacy/{id}/edit`
- `POST /api/purchase_orders/legacy`
- `PATCH /api/purchase_orders/legacy/{id}`

## Record Marker

Legacy/manual purchase orders are identified by:

- `purchase_orders.legacy_manual_entry = true`

That field is added by migration and is set server-side by the legacy endpoints.

## Normal Purchase Order API Behavior

The standard `purchase_orders` collection API remains standard-only:

- collection create cannot create legacy rows
- collection update cannot modify legacy rows
- delete rule remains the normal owner-plus-unapproved rule

Legacy rows still cannot be deleted in practice because this feature always creates them as `Active`.

## Normal Viewing Behavior

Legacy POs are still viewed through the normal PO list/details experience.

The visible/list DTOs include `legacy_manual_entry`, and the UI shows a badge:

- list badge: `Manually created`
- details badge: `Manually created`

The normal edit route redirects legacy rows to the hidden legacy editor.

## Hidden Editor Behavior

The hidden UI reuses `PurchaseOrdersEditor.svelte` in `legacyMode`.

In legacy mode:

- `uid` is editable through a profile typeahead
- `approver` is editable through a profile typeahead
- `po_number` is entered manually
- `status` is shown as fixed `Active`
- `type` is limited to `One-Time` and `Cumulative`
- `branch` is visible and required
- `job` may be selected
- `category` is not shown and is always blank
- recurring controls are hidden
- attachment upload is hidden
- standard approver-pool and second-approver workflow UI is hidden

If a job determines the branch, the branch control stays visible but becomes disabled.

## Closed And Cancelled Legacy PO Behavior

Closed and cancelled legacy POs are read-only.

Behavior split:

- `GET /api/purchase_orders/legacy/{id}/edit` allows closed or cancelled legacy rows to load so the hidden editor can render them read-only
- `PATCH /api/purchase_orders/legacy/{id}` rejects closed or cancelled legacy rows

The editor shows a read-only warning and disables save controls for closed and cancelled legacy POs.

## Editability While Active

Legacy POs remain editable while they are still `Active`.

This is true even if expenses already reference the PO. The current edit lock is based on terminal status, not on whether related expenses exist.

Practical result:

- an active legacy PO with one or more expenses can still be edited through the hidden legacy editor
- once the PO becomes `Closed` or `Cancelled`, the hidden editor becomes read-only and `PATCH` is rejected
- `One-Time` POs usually become non-editable after the first committed expense because expense commit auto-closes them
- `Cumulative` POs remain editable while still active, even if they already have expenses against them

## Submitted Fields

The legacy editor submits only these editable fields:

- `uid`
- `approver`
- `po_number`
- `date`
- `division`
- `description`
- `payment_type`
- `total`
- `vendor`
- `type`
- `kind`
- `branch`
- `job`

Required fields:

- `uid`
- `approver`
- `po_number`
- `date`
- `division`
- `description`
- `payment_type`
- `total`
- `vendor`
- `type`
- `kind`
- `branch`

## Manual PO Number Format

Legacy `po_number` is entered manually and must match:

- `YYMM-NNNN`
- `YY` is `25` or `26`
- `MM` is `01` through `12`
- `NNNN` is in the `5XXX` range

Examples:

- `2501-5000`
- `2612-5999`

Uniqueness relies on the existing unique index on `purchase_orders.po_number`.

## Server-Side Normalization

Every legacy create/update is normalized server-side so the client cannot bypass the intended shape.

Forced values:

- `legacy_manual_entry = true`
- `_imported = false`
- `status = "Active"`
- `category = ""`
- `end_date = ""`
- `frequency = ""`
- `parent_po = ""`
- `attachment = ""`
- `attachment_hash = ""`
- `priority_second_approver = ""`
- `second_approver = ""`
- `second_approval = ""`
- `rejector = ""`
- `rejected = ""`
- `rejection_reason = ""`
- `cancelled = ""`
- `canceller = ""`
- `closed = ""`
- `closer = ""`
- `closed_by_system = false`

Additional create-time behavior:

- `approved` is set to the creation timestamp

Update behavior:

- the existing `approved` value is preserved

Allowed types:

- `One-Time`
- `Cumulative`

`Recurring` is not allowed for legacy create/edit.

## Validation Rules

The legacy endpoints validate:

- feature flag enabled
- caller holds `legacy_po_create_update`
- authenticated user exists
- update target is an existing legacy row
- closed rows cannot be patched
- required fields are present
- `branch` is present
- `po_number` matches the legacy format
- `type` is `One-Time` or `Cumulative`
- `total > 0`
- selected `approver` is an active user
- referenced division, vendor, kind, branch, and job data still pass normal validation

The legacy endpoints reject any submitted field outside the approved allowlist.

Notably rejected or ignored workflow fields include:

- `status`
- `_imported`
- `legacy_manual_entry`
- `category`
- `parent_po`
- `attachment`
- approval/rejection/cancellation/closure fields
- second-approver fields
- recurring fields

## What This Flow Does Not Do

The legacy flow does not use the normal PO approval workflow:

- no first-stage approver-pool computation
- no second approver selection
- no approval reset behavior
- no normal PO approval notifications
- no attachment handling
- no category assignment
- no parent/child PO behavior

## Export / Writeback Behavior

Legacy/manual POs are exported through the legacy expenses writeback flow the same way as other unimported POs.

Purchase order export behavior:

- all `purchase_orders` rows where `_imported = false` are emitted
- PO export ignores `updatedAfter`

This ensures manually entered legacy POs are not lost even if they are not yet referenced by committed expenses.

## Lifecycle Notes

The current implemented lifecycle for legacy POs is:

- create/edit requires the `legacy_po_create_update` claim and the feature flag
- legacy create forces the PO directly to `Active`
- legacy POs are not deletable in practice because they are never `Unapproved`
- `One-Time` legacy POs auto-close when an associated expense is committed
- `Cumulative` legacy POs auto-close when committed expenses reach or exceed the PO total
- active legacy POs may be cancelled through the normal cancel route only when they have no associated expenses and the caller has `payables_admin`
- cancelled legacy POs remain visible but are no longer editable through the hidden legacy editor
- active cumulative legacy POs may be manually closed through the normal close route only when they have at least one committed expense and the caller has `payables_admin`

## UI Safety Behavior

The hidden URL is not treated as security.

UI safety behavior is intentionally secondary to backend enforcement:

- if the feature flag is off, the editor shows an error banner and save is blocked
- if the claim is missing, the editor warns and backend save is rejected
- if load fails, the editor renders an error state instead of a fallback create form
- if the PO is closed or cancelled, the editor loads in read-only mode and save stays disabled

## Tested Scenarios

Backend coverage includes:

- feature flag disabled
- missing claim
- legacy create success
- legacy update success
- edit-load success
- closed edit-load success
- non-legacy update denied
- closed update denied
- invalid PO number rejected
- invalid type rejected
- missing branch rejected
- submitted category rejected
- `_imported` forced false
- `status` forced `Active`
- `approved` set on create
- category, recurring, parent, attachment, second-approver, rejection, and cancellation fields cleared on save

Collection regression coverage includes:

- standard collection create cannot create legacy rows
- standard collection update cannot modify legacy rows
- standard collection delete cannot delete active legacy rows
