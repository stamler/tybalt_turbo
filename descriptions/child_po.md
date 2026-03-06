# Child Purchase Order System

## Overview

When a Cumulative PO's total is exceeded by an expense, rather than simply
returning an error, a provision exists to create a child PO that inherits
properties from the parent PO. This streamlines the process of handling overflow
expenses while maintaining proper tracking and relationships.

## Implementation Status

Core child PO functionality is implemented. Items marked **[TODO]** below are planned but not yet built.

### Database Schema

- `parent_po` (relation) field exists on `purchase_orders` collection.
- Child POs must be of type `One-Time` (enforced in hooks).
- Child POs cannot be created from an existing child PO (child POs cannot be parents).

### Backend: Implemented

#### Purchase Order Hook Validation (`app/hooks/purchase_orders.go`)

When `parent_po` is set, the following are enforced:

- Parent PO must exist
- Parent PO must not itself be a child (`parent_po` must be empty)
- Parent PO must be `Active`
- Parent PO must be `Cumulative`
- No other non-terminal (`Closed`/`Cancelled`) child POs may exist for the same parent
- Child PO must match parent on: `job`, `payment_type`, `category`, `description`, `vendor`, `kind`
- Child PO must be of type `One-Time`

#### Expense Validation (`app/hooks/validate_expenses.go`)

Cumulative PO overflow returns error code `cumulative_po_overflow` with structured payload:
- `purchase_order`, `po_number`, `po_total`, `overflow_amount`

This error is caught by the expenses UI to trigger the child PO creation flow.

#### PO Number Generation (`app/routes/purchase_orders.go`)

- Parent POs: `YYMM-NNNN` format (existing behavior)
- Child POs: `YYMM-NNNN-XX` format where `XX` is zero-padded sequential (01-99)
- Maximum 99 children per parent

### Backend: TODO

- **[TODO]** `existing_child_po` pre-check in expense validation — currently the overflow error is returned without first checking whether the parent already has an active child PO. That check only happens when the user attempts to create the child PO.
- **[TODO]** Parent PO cancellation protection — a parent PO with children (but no direct expenses) can currently be cancelled. Should block cancel if non-terminal children exist.
- **[TODO]** Parent PO approval-status-change protection — no guard prevents changes to a parent PO's approval status when it has children.

### Frontend: Implemented

- `CumulativePOOverflowModal.svelte` — modal triggered on overflow error, displays overflow details and navigates to child PO creation
- `/pos/[poid]/add-child/+page.svelte` — route that pre-populates a child PO from parent values (division, description, payment_type, vendor, job, category, kind)
- `PurchaseOrdersEditor.svelte` — displays "Child PO of [parent number]" label when `parent_po` is set
- Details page shows clickable parent PO link

### Testing

`TestGeneratePONumber` in `app/purchase_orders_test.go` has 13 test cases covering parent/child generation, suffix sequencing, the 99-child limit, and the 5000+ reserved range.

- **[TODO]** Integration tests for complete workflow (expense submission through child PO creation)
- **[TODO]** Integration tests for concurrent child PO creation attempts
- **[TODO]** Integration tests for parent PO protection rules
