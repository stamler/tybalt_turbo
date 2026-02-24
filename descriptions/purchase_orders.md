# Purchase Orders System (Current Model)

This document describes the current PO approval model implemented in February 2026.

For implementation-level details and decision history, see:

- `/Users/dean/code/tybalt_turbo/descriptions/feb26_po_mods.md`

## Core Approval Model

Approval is stage-based and kind-aware.

- Stage 1 is a vetting step.
- Stage 2 is final approval.
- A PO requires Stage 2 only when:
  - `expenditure_kinds.second_approval_threshold > 0`, and
  - `approval_total > second_approval_threshold`.

Global `po_approval_thresholds` tiers are no longer used.

### Threshold = 0 Implication (Explicit)

- If `expenditure_kinds.second_approval_threshold` is `0` (or null-coalesced to `0`), dual approval is never required for that kind.
- In that case, there is no first-stage/second-stage split: first-stage candidates are all eligible approvers with `limit >= approval_total`.
- A high-authority approver appears as "second approver only" only when both conditions are true:
  - `second_approval_threshold > 0`
  - `approval_total > second_approval_threshold`

## Expenditure Kinds and Limit Columns

The PO `kind` determines which `po_approver_props` limit column is used:

- `capital` (no job, `allow_job=false`): `max_amount`
- `project` (requires job, `allow_job=true`): `project_max`
- `sponsorship`: `sponsorship_max`
- `staff_and_social`: `staff_and_social_max`
- `media_and_event`: `media_and_event_max`
- `computer`: `computer_max`

`kind` handling differs by surface:

- Approver lookup endpoints default missing `kind` to `capital` (no job) or `project` (with job).
- PO create/update validation requires a non-empty, valid `kind`; save fails if `kind` is missing or invalid.

## approval_total

`approval_total` is computed in hooks and drives stage requirements:

- `One-Time`: `approval_total = total`
- `Cumulative`: `approval_total = total`
- `Recurring`: `approval_total = total * occurrences`

## Stage Pools

Eligibility always requires:

- `po_approver` claim
- active user
- division authorization (`divisions` empty means all divisions)
- non-null value in the resolved limit column

If dual approval is required:

- First-stage pool: eligible approvers with `limit <= second_approval_threshold`
- Second-stage pool: eligible approvers with `limit > second_approval_threshold` and `limit >= approval_total`

If dual approval is not required:

- First-stage pool includes eligible approvers with `limit >= approval_total` (single-stage approver is a final approver)

## Second Approver Endpoint Contract

`GET /api/purchase_orders/second_approvers` returns candidates who can final-approve the evaluated amount.

Behavior by policy state:

- If second approval is not required: `200` with metadata indicating `not_required`.
- If second approval is required and requester is self-qualified:
  - `200` with metadata indicating `requester_qualifies`
  - `approvers` is intentionally returned as an empty list for UI auto-self handling.
- If second approval is required and the requester is not self-qualified:
  - if at least one final-capable candidate exists: `200` with candidates/meta
  - if no final-capable candidate exists: `400` with `code = second_pool_empty`

## First Approver Endpoint Contract

`GET /api/purchase_orders/approvers` returns the full first-stage pool for the evaluated PO context.

- If requester is in the first-stage pool, UI may auto-assign requester.
- If no first-stage approvers qualify, endpoint returns `200` with an empty list (`[]`).

## Editor UX (Current)

- Second-approver UI is shown only when second approval is required.
- If required and second-stage candidates are available, show the `priority_second_approver` selector.
- If required and no second-stage candidates are available, show an error state with explanation/diagnostics ("Why?").
- If second approval is not required, no second-approver status hint is shown.
- Own-PO bypass UX exception: when PO is dual-required and requester is second-stage-qualified (`requester_qualifies`), hide both approver selectors and persist `approver = requester`, `priority_second_approver = requester`.

## Required Assignment Rules on Save

For dual-required POs (`approval_total > second_approval_threshold`):

- `approver` is required and must be in first-stage pool
- `priority_second_approver` is required and must be in second-stage pool
- first-stage pool must be non-empty
- second-stage pool must be non-empty
- Narrow create/update exception: allow `approver = uid` when all are true:
  - PO is dual-required
  - `uid` is second-stage-qualified for the PO context
  - `priority_second_approver = uid`

For single-stage POs:

- `priority_second_approver` is cleared

## Editability Rules

Direct record updates are allowed only when all of the following are true:

- Caller is the PO creator (`uid = @request.auth.id`)
- `status = Unapproved`
- `second_approval = ""`
- Submitted `status` must remain `Unapproved` (hook-level validation rejects create/update requests that attempt `Active`/`Cancelled`/`Closed`)

Direct record deletion is allowed only when all of the following are true:

- Caller is the PO creator (`uid = @request.auth.id`)
- `status = Unapproved`

Create-path guardrails (collection API rules):

- New POs must be created with `status = Unapproved`.
- Approval/cancellation/closure fields must not be submitted on create.
- UI delete action is shown only when the caller meets the delete rule above (creator + `Unapproved`).

Implications:

- First-approved POs that are still `Unapproved` are editable by the creator.
- Fully approved / final-state records remain non-editable.
- Rule-denied updates can surface as a generic `404 The requested resource wasn't found.` response from PocketBase.
- While a PO is still editable, save-time cleaning clears rejection fields (`rejected`, `rejector`, `rejection_reason`) to support resubmission after changes.

### Editability Matrix

| PO State | Editable by Creator? | Notes |
| --- | --- | --- |
| `Unapproved`, `approved = ""`, `second_approval = ""` | Yes | Draft / pending first approval |
| `Unapproved`, `approved != ""`, `second_approval = ""` | Yes | First-approved, pending second approval |
| `Unapproved`, `second_approval != ""` | No | Final approval complete; update rule blocks |
| `Active` | No | Use approve/cancel/close lifecycle actions |
| `Cancelled` | No | Terminal |
| `Closed` | No | Terminal |

### Meaningful Edit Detection

Approval reset behavior is gated by `utilities.RecordHasMeaningfulChanges(...)` with PO-specific skipped fields:

- `approved`, `second_approval`, `second_approver`
- `rejector`, `rejected`, `rejection_reason`
- `cancelled`, `canceller`
- `closed`, `closer`, `closed_by_system`
- `po_number`, `status`

`created`/`updated` are always skipped by the shared utility.

### Approval Reset on Editable Updates

When a first-approved editable PO is updated and the change is meaningful:

- approval state is reset (`approved`, `second_approval`, `second_approver` cleared)
- first-stage assignment is reset and revalidated (`approver`)
- approver assignment is revalidated against current policy
- a new `po_approval_required` notification is sent to the selected first-stage approver

Because `approver` is not in the skipped-field list, changing `approver` on a first-approved editable PO is always treated as a meaningful change and triggers this reset/re-notification flow.

When the update is a no-op (no meaningful business-field change), approval state is preserved and no new approval notification is sent.

When the PO is still a draft (`approved = ""`), edits do not trigger this reset/re-notification flow. This suppresses repeated notification spam during iterative drafting.

### Save vs Approve

- Creating or saving a PO does **not** approve it.
- This is true even when the creator is also approval-qualified and assigns themself as approver.
- A saved PO remains editable by the creator while it stays `Unapproved` and not second-approved.
- Approval only happens when the explicit approve action is executed (`POST /api/purchase_orders/{id}/approve`).
- Therefore, managers can draft and save their own POs, review/edit them, and only lock them once they intentionally run approval.

## Approve Endpoint Behavior

Route: `POST /api/purchase_orders/{id}/approve`

### Stage 1 (normal path)

When PO is not first-approved yet (`approved` empty):

- Caller must equal assigned `approver`
- Assigned `approver` must still be valid first-stage approver
- Sets:
  - `approved = now`
  - `approver = caller`
- If PO is single-stage, it is activated immediately and receives `po_number`

### Stage 2 (normal path)

When PO is first-approved and still pending second approval:

- Caller must be valid second-stage approver
- Hard safety gate: caller's resolved limit must be `>= approval_total`
- Sets:
  - `second_approval = now`
  - `second_approver = caller`
- Activates PO and assigns `po_number`

### Combined Dual Approval (bypass fast path)

When PO is dual-required, not yet first-approved, and caller already qualifies for second stage:

- API allows one-call full approval
- Sets in one action:
  - `approver = caller`
  - `approved = now`
  - `second_approver = caller`
  - `second_approval = now`
  - `status = Active`
  - `po_number` generated/applied

## Pending Queue Visibility (`GET /api/purchase_orders/pending*`)

Visibility is stage-based and denoising-oriented.

### Stage 1 visibility

Before first approval (`approved` empty):

- Only assigned `approver` sees the PO in pending

### Stage 2 exclusive visibility

After first approval and before second approval:

- `priority_second_approver` gets exclusive visibility for timeout window `T`

### Stage 2 fallback visibility

After `approved + T`:

- Any second-stage eligible approver may see it

`T` comes from `app_config.purchase_orders.second_stage_timeout_hours`.

Fallback behavior:

- missing/invalid/non-positive config value uses `24` hours

## Broad Visibility (`GET /api/purchase_orders/visible*`)

The `visible` endpoints use broader read semantics than `pending`:

- `Active`: visible to any authenticated user
- `Cancelled`/`Closed`: visible to creator, approvers, and `report` claim holders
- `Unapproved`:
  - direct visibility: creator, assigned first approver (pre-first-approval), priority second approver (post-first-approval)
  - plus policy-based second-stage visibility for eligible second-stage approvers on first-approved/not-second-approved records (not timeout-gated)

`GET /api/purchase_orders/visible` supports `scope`:

- `all` (default)
- `mine` (where `uid = caller`)
- `active` (status `Active`)
- `stale` (status `Active` and older than `stale_before`; uses `second_approval` when present, otherwise `approved`)

Note:

- Collection `list/view` rules are direct-only for unapproved records; the broader policy-based second-stage visibility above is implemented by the custom `/visible*` SQL endpoints.

## priority_second_approver

`priority_second_approver` is mandatory for dual-required POs and defines the Stage 2 priority owner for the pending queue during the timeout window.

This is noise control, not an authorization bypass. Approval authorization is still policy-based:

- final approval requires second-stage eligibility and sufficient limit
- approve-route authorization is not restricted to `priority_second_approver` during the timeout window

## Notifications

Daily second-approval reminder recipients are sourced from:

- `pending_items_for_qualified_po_second_approvers`

That view follows the same stage-2 eligibility model:

- first-approved
- not second-approved
- still unapproved
- beyond timeout
- second-stage eligible for kind/division/amount

## Data/View Model Changes

Removed:

- `po_approval_thresholds`
- threshold-derived view fields (`lower_threshold`, `upper_threshold`)
- `user_po_permission_data`

Current canonical views:

- `user_claims_summary`: global claims/navigation context
- `user_po_approver_profile`: PO approver profile context (limits/divisions/claims)

## Lifecycle Summary

1. PO created as `Unapproved`
2. Stage 1 performed by assigned first approver (or bypass fast path)
3. If dual-required, pending queue ownership is priority-owner first, then broader pool after timeout; final approval authorization remains any second-stage-eligible approver
4. PO becomes `Active` only when approval requirements are fully satisfied
5. Active PO is later cancelled/closed per existing status rules

## Purchase Order Types and Behaviors

| PO Type    | Description                                                                  | May be Closed Manually                                              | May be Canceled if status is Active                  | Closed Automatically                     |
| ---------- | ---------------------------------------------------------------------------- | ------------------------------------------------------------------- | ---------------------------------------------------- | ---------------------------------------- |
| One-Time   | Valid for a single expense                                                   | No                                                                  | Yes, by `payables_admin` if no expenses are attached | When an expense is committed             |
| Recurring  | Valid for a fixed number of expenses not exceeding specified value           | Yes, if status is `Active` and has at least 1 **committed** expense | Yes, by `payables_admin` if no expenses are attached | When final expected expense is committed |
| Cumulative | Valid for multiple expenses where sum of values does not exceed the PO total | Yes, if status is `Active` and has at least 1 **committed** expense | Yes, by `payables_admin` if no expenses are attached | When committed total reaches PO total    |

Operational handlers:

- close: `createClosePurchaseOrderHandler` (`/Users/dean/code/tybalt_turbo/app/routes/close_purchase_order.go`)
- cancel: `createCancelPurchaseOrderHandler` (`/Users/dean/code/tybalt_turbo/app/routes/purchase_orders.go`)
- convert one-time to cumulative: `createConvertToCumulativePurchaseOrderHandler` (`/Users/dean/code/tybalt_turbo/app/routes/purchase_orders.go`)

## When a Purchase Order Is Required

Expenses require a linked PO under two independent rules, enforced in `validate_expenses.go`:

### Rule 1: Job-based requirement (line 265)

Any expense with a `job` requires a PO, **except** these payment types which are exempt:

- `Mileage`
- `FuelCard`
- `PersonalReimbursement`
- `Allowance`

### Rule 2: Amount-based requirement (line 247)

When `LIMIT_NON_PO_AMOUNTS` is enabled (currently `true`), any expense at or above the configured limit requires a PO. The limit is exclusive (e.g., $99.99 is allowed without a PO when the limit is $100).

The limit is configurable via `app_config` under the `"expenses"` domain:

```json
{ "no_po_expense_limit": 100.0 }
```

- Positive number: limit enforced at that amount
- Zero: PO required for all non-exempt expenses (strictest policy)
- Absent/null: defaults to `constants.NO_PO_EXPENSE_LIMIT` (`100.0`)
- Negative: ignored, uses default

Read by `utilities.GetNoPOExpenseLimit(app)` in `app/utilities/config.go`.

**Exempt payment types** (no amount cap, PO never required by this rule):

- `Mileage`
- `FuelCard`
- `PersonalReimbursement`
- `Allowance`

**Payables admin bypass**: Users with the `payables_admin` claim can exceed the limit without a PO, but **only** when `payment_type = OnAccount`.

### Combined effect

| Scenario | PO Required? |
| --- | --- |
| Has job, payment type is `OnAccount` / `Expense` / `CorporateCreditCard` | Yes (any amount) |
| Has job, payment type is `Mileage` / `FuelCard` / `PersonalReimbursement` / `Allowance` | No |
| No job, total >= configured limit (default $100), not exempt type | Yes |
| No job, total < configured limit (default $100) | No |
| No job, total >= configured limit (default $100), `OnAccount` + `payables_admin` claim | No |

## Expense Validation Against PO Totals

When saving an expense linked to a PO, validation behavior depends on PO type.

Baseline gate:

- The referenced PO must be `Active` or save fails.

### One-Time and Recurring

For `One-Time` and `Recurring` POs, a small overage is allowed.

- Allowed maximum for a single expense is:
  - `po.total * (1 + MAX_PURCHASE_ORDER_EXCESS_PERCENT)`, capped by
  - `po.total + MAX_PURCHASE_ORDER_EXCESS_VALUE`
- The lower of those two limits is used.
- Current constants are:
  - `MAX_PURCHASE_ORDER_EXCESS_PERCENT = 0.05` (5%)
  - `MAX_PURCHASE_ORDER_EXCESS_VALUE = 100.0` ($100)
- If expense total exceeds that limit, save fails with a `total` validation error:
  - `expense exceeds purchase order total of $... by more than ...`

Recurring-specific date rules still apply (expense date must be within PO date and end date range).

### Cumulative

For `Cumulative` POs, overage is checked against the running sum of associated expenses:

- `new_total = existing_expenses_total + new_expense_total`
- If `new_total > po.total`, save fails immediately with code `cumulative_po_overflow`.
- The error payload includes:
  - `purchase_order`
  - `po_number`
  - `po_total`
  - `overflow_amount`

Notes:

- The running total check currently uses all associated expenses (not just committed ones) during create/update validation.
- This overflow error is used by the expense UI to trigger the child-PO overflow workflow.

### Known Limitation

- `One-Time` is described as "single expense", but current validation does not yet hard-block creating a second expense against the same one-time PO.
- There is an existing TODO in validation code to enforce this.

## Child PO Semantics

- A purchase order may reference `parent_po`.
- Child POs must match parent values for:
  - `job`
  - `payment_type`
  - `category`
  - `description`
  - `vendor`
  - `kind`
- Child POs use number format `YYMM-NNNN-XX` and support up to 99 children per parent.

## Cancellation

Route: `POST /api/purchase_orders/:id/cancel`

- Caller must have `payables_admin` claim.
- PO must be `Active`.
- PO must have no associated expenses.
- On success:
  - `canceller` is set
  - `cancelled` timestamp is set
  - `status` becomes `Cancelled`

## Closure

Route: `POST /api/purchase_orders/:id/close`

Manual closure rules:

- Only `Recurring` and `Cumulative` POs may be manually closed.
- PO must have at least one associated **committed** expense.
- On success:
  - `closer` is set
  - `closed` timestamp is set
  - `status` becomes `Closed`

Automatic closure rules:

- `One-Time`: closes when a committed expense is recorded.
- `Recurring`: closes when all expected expenses are committed.
- `Cumulative`: closes when committed total reaches PO total.

Automatic close sets `closed_by_system = true`.

## API Endpoints (Non-approver-list)

Implemented in `/Users/dean/code/tybalt_turbo/app/routes/purchase_orders.go` and `/Users/dean/code/tybalt_turbo/app/routes/close_purchase_order.go`, and registered in `/Users/dean/code/tybalt_turbo/app/routes/routes.go`.

- `POST /api/purchase_orders/:id/approve`
- `POST /api/purchase_orders/:id/reject`
- `POST /api/purchase_orders/:id/cancel`
- `POST /api/purchase_orders/:id/close`
- `POST /api/purchase_orders/:id/make_cumulative`
- `GET /api/purchase_orders/pending`
- `GET /api/purchase_orders/pending/:id`
- `GET /api/purchase_orders/visible`
- `GET /api/purchase_orders/visible/:id`

## PO Number Format

Parent POs: `YYMM-NNNN`

- `YY`: 2-digit year
- `MM`: 2-digit month
- `NNNN`: sequential 4-digit auto-generated number (`0001`-`4999`)
- `5000+` values are reserved for manually assigned/imported PO numbers

Child POs: `YYMM-NNNN-XX`

- `XX`: sequential 2-digit child suffix (`01`-`99`)

## Pocketbase Collection Schema (`purchase_orders`)

| Field                    | Type                          | Description                                                               |
| ------------------------ | ----------------------------- | ------------------------------------------------------------------------- |
| po_number                | string                        | `YYMM-NNNN` or `YYMM-NNNN-XX`; required for `Active`/`Cancelled`/`Closed` |
| status                   | enum                          | `Unapproved`, `Active`, `Cancelled`, `Closed`                             |
| uid                      | relation -> users             | Creator                                                                   |
| type                     | enum                          | `One-Time`, `Cumulative`, `Recurring`                                     |
| kind                     | relation -> expenditure_kinds | Expenditure kind; drives limit-column resolution                          |
| date                     | string                        | Start date (`YYYY-MM-DD`)                                                 |
| end_date                 | string                        | End date for recurring POs                                                |
| frequency                | enum                          | `Weekly`, `Biweekly`, `Monthly`                                           |
| division                 | relation -> divisions         | Division context                                                          |
| description              | string                        | Minimum 5 chars                                                           |
| total                    | number                        | User-entered PO amount                                                    |
| approval_total           | number                        | Computed amount used for approval policy                                  |
| payment_type             | enum                          | `OnAccount`, `Expense`, `CorporateCreditCard`                             |
| vendor                   | relation -> vendors           | Vendor                                                                    |
| job                      | relation -> jobs              | Optional job                                                              |
| category                 | relation -> categories        | Optional category                                                         |
| branch                   | relation -> branches          | Derived from job or user default                                          |
| attachment               | file                          | Optional file                                                             |
| approver                 | relation -> users             | Assigned first-stage approver                                             |
| approved                 | datetime                      | First-approval timestamp                                                  |
| second_approver          | relation -> users             | Final approver                                                            |
| second_approval          | datetime                      | Final-approval timestamp                                                  |
| priority_second_approver | relation -> users             | Exclusive stage-2 owner during timeout window                             |
| rejector                 | relation -> users             | Rejecting user                                                            |
| rejected                 | datetime                      | Rejection timestamp                                                       |
| rejection_reason         | string                        | Rejection explanation                                                     |
| canceller                | relation -> users             | Cancelling user                                                           |
| cancelled                | datetime                      | Cancellation timestamp                                                    |
| closer                   | relation -> users             | Manual closer                                                             |
| closed                   | datetime                      | Closure timestamp                                                         |
| closed_by_system         | boolean                       | Whether closure was automatic                                             |
| parent_po                | relation -> purchase_orders   | Parent pointer for child POs                                              |

## Pocketbase Collection Schema (`expenditure_kinds`)

| Field                     | Type   | Description                                                   |
| ------------------------- | ------ | ------------------------------------------------------------- |
| id                        | string | PocketBase record ID                                          |
| name                      | string | Semantic key (`capital`, `project`, `sponsorship`, etc.)      |
| description               | string | Human-readable description                                    |
| ui_order                  | number | Display ordering                                              |
| en_ui_label               | string | UI label                                                      |
| allow_job                 | bool   | Whether this kind permits a `job` on PO                       |
| second_approval_threshold | number | Threshold above which dual approval is required for this kind |

## `po_approver_props` Limit Columns

| Column                 | Used for kind      | When          |
| ---------------------- | ------------------ | ------------- |
| `max_amount`           | `capital`          | Always        |
| `project_max`          | `project`          | Always        |
| `sponsorship_max`      | `sponsorship`      | Always        |
| `staff_and_social_max` | `staff_and_social` | Always        |
| `media_and_event_max`  | `media_and_event`  | Always        |
| `computer_max`         | `computer`         | Always        |
