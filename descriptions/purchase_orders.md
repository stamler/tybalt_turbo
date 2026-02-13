# The Purchase Orders System

## Purchase Order Types and Their Behaviours

| PO Type    | Description                                                                                   | May be Closed Manually                                               | May be Canceled if status is Active                              | Closed automatically                    | Can be converted to different PO type                                         | Approval tier required                                |
| ---------- | --------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- | ---------------------------------------------------------------- | --------------------------------------- | ----------------------------------------------------------------------------- | ----------------------------------------------------- |
| One-Time   | Valid for a single expense                                                                    | No                                                                   | Yes, by payables_admin if no expenses are associated             | When an expense is committed against it | Yes, to Cumulative by payables_admin claim holder                             | Based on approval_total (same as total)               |
| Recurring  | Valid for a fixed number of expenses not exceeding specified value                            | Yes, if status is Active and > 0 expenses associated                 | Yes, by payables_admin if no expenses are associated             | When the final expense is committed     | No                                                                            | Based on approval_total (total × number of periods)   |
| Cumulative | Valid for an unlimited number of expenses where sum of values does not exceed specified value | Yes, if status is Active and > 0 expenses associated                 | Yes, by payables_admin if no expenses are associated             | When the maximum amount is reached      | No                                                                            | Based on approval_total (same as total)               |
|            |                                                                                               | createClosePurchaseOrderHandler (app/routes/close_purchase_order.go) | createCancelPurchaseOrderHandler (app/routes/purchase_orders.go) | n/a (handled in expense processing)     | createConvertToCumulativePurchaseOrderHandler (app/routes/purchase_orders.go) | getSecondApproverClaim (app/hooks/purchase_orders.go) |

## Expenditure Kinds

Every purchase order has a `kind` field that references the `expenditure_kinds` collection. The kind classifies the nature of the expenditure and determines which approval limit column is used for second-approval qualification.

### Available Kinds

| Kind Name          | `po_approver_props` limit column (no job) | `po_approver_props` limit column (with job) | Description                           |
| ------------------ | ----------------------------------------- | ------------------------------------------- | ------------------------------------- |
| `standard`         | `max_amount`                              | `project_max`                               | Default kind for general expenditures |
| `sponsorship`      | `sponsorship_max`                         | `sponsorship_max`                           | Sponsorship-related expenditures      |
| `staff_and_social` | `staff_and_social_max`                    | `staff_and_social_max`                      | Staff and social expenditures         |
| `media_and_event`  | `media_and_event_max`                     | `media_and_event_max`                       | Media and event expenditures          |
| `computer`         | `computer_max`                            | `computer_max`                              | Computer/IT expenditures              |

The `standard` kind is the only kind that differentiates between job-present and no-job scenarios (using `project_max` vs `max_amount` respectively). All other kinds use the same limit column regardless of whether a job is attached.

### How Kind Affects Approval

The kind determines which `po_approver_props` limit column is checked when evaluating second-approval eligibility. The resolution logic is implemented in `ResolvePOApproverLimitColumn` (`app/utilities/expenditure_kinds.go`).

### Kind on Expenses

- Expenses linked to a purchase order **inherit** the PO's kind automatically. The kind is not user-editable on PO-linked expenses.
- Expenses without a purchase order (standalone/no-PO expenses) are always forced to the `standard` kind by the `cleanExpense` hook.

### Kind Validation

- `kind` is required on new purchase orders and new expenses.
- For expenses, clients do not typically need to provide `kind`: the server derives it (`cleanExpense` forces `standard` for no-PO expenses, and `ProcessExpense` overwrites it from the linked PO when a purchase order is present).
- The referenced `expenditure_kinds` record must exist.
- For purchase orders, if `job` is set then the selected kind must have `allow_job = true` in `expenditure_kinds` (currently `standard` and `computer`).
- Legacy records (created before kind existed) with an empty kind are silently upgraded to `standard` when updated.
- Child POs must have the same kind as their parent (with normalization for legacy parents that have an empty kind).

### Kind Configuration Validation

At server startup (`app/routes/routes.go` → `ValidateExpenditureKindsConfig`), the system verifies:

1. All expected kind names exist in the `expenditure_kinds` collection
2. All expected limit columns exist on the `po_approver_props` table

The name↔ID mappings are cached at startup and used throughout the application.

Operational note: these checks protect runtime behavior, but approver correctness in hybrid Turbo/legacy operation depends on synchronized data flow for approver props (`poApproverProps` writeback -> Firestore `TurboPoApproverProps` -> MySQL `TurboPoApproverProps` -> `import_data --users`).

## Approval Total

The `approval_total` field determines which approval tier a purchase order falls into and whether second approval is required. It is calculated automatically when a purchase order is created or updated.

### Calculation by PO Type

| PO Type    | approval_total Calculation      |
| ---------- | ------------------------------- |
| One-Time   | Same as `total`                 |
| Cumulative | Same as `total`                 |
| Recurring  | `total × number_of_occurrences` |

### Recurring PO Calculation Details

For Recurring purchase orders, the number of occurrences is calculated based on the date range and frequency:

```go
occurrences = floor(days_between_start_and_end / frequency_days)
approval_total = total × occurrences
```

Where frequency_days is:

- Weekly: 7 days
- Biweekly: 14 days
- Monthly: 30 days (approximation)

**Constraints:**

- A recurring PO must have at least 2 occurrences
- end_date must be after date (start date)

**Example:** A $500 monthly recurring PO from January 1 to December 31 (365 days):

- Occurrences = floor(365 / 30) = 12
- approval_total = $500 × 12 = $6,000

This ensures the approval tier reflects the total financial commitment over the life of the recurring PO, not just a single payment amount.

### How approval_total is Used

1. **Determining if second approval is required:** If approval_total > lowest threshold in `po_approval_thresholds`, second approval is required
2. **Determining which second approvers are qualified:** Only users whose kind-specific `po_approver_props` limit column value >= approval_total (and <= the tier ceiling) can perform second approval. See [Expenditure Kinds](#expenditure-kinds) for which column is used.
3. **Clearing priority_second_approver:** If approval_total <= lowest threshold, the priority_second_approver field is automatically cleared since second approval is not needed

## Approvals

All purchase orders have a status of `Unapproved` when created.

### Approval Thresholds and Tiers

Approval thresholds are stored in the `po_approval_thresholds` table. These thresholds group purchase orders into tiers:

- **Lower threshold:** The highest threshold that is less than the approval_total
- **Upper threshold (ceiling):** The lowest threshold that is greater than or equal to the approval_total

Purchase orders with an approval_total greater than the lowest threshold require second approval.

### Who Can Approve

**First Approval:** Any user who:

1. Has the `po_approver` claim
2. Has a `po_approver_props` record where divisions is empty (all divisions) OR contains the PO's division

**Second Approval:** Any user who meets the first approval requirements, PLUS:

1. Their kind-specific `po_approver_props` limit column value >= the PO's approval_total
2. Their kind-specific `po_approver_props` limit column value <= the tier ceiling

The limit column is determined by the PO's `kind` and whether the PO has a `job`:

| PO kind            | PO has job? | `po_approver_props` limit column used |
| ------------------ | ----------- | ------------------------------------- |
| `standard`         | No          | `max_amount`                          |
| `standard`         | Yes         | `project_max`                         |
| `sponsorship`      | Either      | `sponsorship_max`                     |
| `staff_and_social` | Either      | `staff_and_social_max`                |
| `media_and_event`  | Either      | `media_and_event_max`                 |
| `computer`         | Either      | `computer_max`                        |

Note: One user can be both the approver and second_approver if they meet the requirements for both.

### Approver Selection at Creation

When creating a PO, the creator must select an approver from a list of qualified approvers for the chosen division. The UI fetches this list via `GET /api/purchase_orders/approvers?division=...&amount=...&kind=...&has_job=...`.

**Special case:** If the creator is themselves a qualified first approver for the division, the approver field is automatically set to the creator (the field is hidden in the UI).

**Important:** The selected approver is stored in the `approver` field, but this acts as a **suggested** approver rather than an exclusive assignment. All qualified first approvers for that division will see the PO in their pending queue, and any of them can approve it. When approval occurs, the `approver` field is **overwritten** with whoever actually performed the approval.

### How Approvers See Pending POs

Approvers access the "Pending My Approval" page (`/pos/pending`) which shows POs they are qualified to approve. The system automatically filters this list based on the approver's permissions — not based on whether they were the originally selected approver.

A PO appears in an approver's pending queue when:

1. **First approval needed:** The PO is unapproved (`approved = ''`) and the approver is qualified for the PO's division
2. **Priority second approval (within 24 hours):** The PO has first approval but needs second approval, and the approver is the designated `priority_second_approver`
3. **General second approval (after 24 hours):** The PO has first approval, needs second approval, the 24-hour priority window has expired, and the approver is qualified for both the division and the approval_total amount (using the kind-specific limit column)

This means:

- All qualified first approvers see unapproved POs for their divisions (not just the one originally selected)
- Any qualified first approver can perform the approval
- The `approver` field reflects who actually approved, not who was originally selected

### Division-Specific Approver Logic

Approvers may have division restrictions in their `po_approver_props`:

1. Approvers with an empty `divisions` array can approve POs for any division
2. Approvers with a non-empty `divisions` array can only approve POs for divisions listed in that array

### priority_second_approver

When a purchase order requires second approval, any qualified user could perform it. This would result in many purchase orders appearing in multiple approval queues, creating noise and confusion over responsibility.

The `priority_second_approver` field addresses this by:

1. Allowing the creator to designate a specific second approver
2. Creating a 24-hour exclusive review window for the designated approver
3. After 24 hours, falling back to making the PO visible to all qualified second approvers

### 24-Hour Window Implementation

When a purchase order requires second approval:

1. For the first 24 hours after first approval, only the `priority_second_approver` sees the PO in their pending queue
2. After 24 hours (based on the `updated` timestamp), the PO becomes visible to all qualified second approvers
3. The `updated` timestamp resets if the PO is modified, restarting the 24-hour window

## Description of the Purchase Orders System

Any user can create a purchase order. The creator's uid is stored in the `uid` column.

A purchase order can be of type `One-Time`, `Cumulative`, or `Recurring`:

- **One-Time:** Valid for one expense, then automatically closed
- **Cumulative:** Valid for multiple expenses until their sum reaches the PO total
- **Recurring:** Valid for a fixed number of expenses at a specified frequency until an end date

A purchase order has a `payment_type` that specifies how expenses are paid, a `kind` that classifies the expenditure (see [Expenditure Kinds](#expenditure-kinds)), and may have an attachment file.

**Approval Flow:**

1. Creator selects a kind, division, amount, and other fields
2. Creator selects an approver from a list of qualified approvers for the division (or auto-set to self if creator is qualified)
3. If second approval will be required, creator may optionally designate a `priority_second_approver`
4. The PO appears in the pending queue of **all** qualified first approvers for that division (not just the selected one)
5. Any qualified first approver can perform first approval — the `approver` field is overwritten with whoever actually approves
6. The endpoint determines whether to set first approval, second approval, or both based on the caller's permissions
7. Upon full approval, a `po_number` is generated in format `YYMM-NNNN` (e.g., `2501-0001`)

**Rejection:** Any qualified approver or second approver can reject an unapproved PO. Rejection records the rejector, rejection reason, and timestamp. The status remains `Unapproved`.

**Child POs:** A purchase order may have a `parent_po` reference. Child POs must match the parent's `job`, `payment_type`, `category`, `description`, `vendor`, and `kind`. Child POs receive a number in format `YYMM-NNNN-XX` (e.g., `2501-0001-01`), supporting up to 99 children per parent.

## Lifecycle of a Purchase Order

1. A purchase order is created by a user who selects a kind and an approver from a list (status: `Unapproved`)
2. The PO appears in the pending queue of **all** qualified first approvers for the division (not just the selected one)
3. The creator may delete the PO as long as it is not `Active`
4. Any qualified first approver can approve or reject the PO (the `approver` field is set to whoever actually approves)
5. If second approval is required (approval_total > lowest threshold):
   - If a `priority_second_approver` was designated, they have a 24-hour exclusive window
   - After 24 hours (or immediately if no priority approver), all qualified second approvers can act
6. Upon full approval, the PO becomes `Active` and receives a po_number
7. Expenses can be committed against the PO (expenses inherit the PO's kind)
8. The PO is closed (manually or automatically) or cancelled

## Cancellation

Only users with the `payables_admin` claim can cancel an Active purchase order, and only if the PO has no associated expenses.

When cancelled:

- The `canceller` property records the user who cancelled
- The `cancelled` timestamp is set
- The status changes to `Cancelled`

## Closure

Purchase orders can be closed manually or automatically:

**Manual Closure:**

- Only `Recurring` and `Cumulative` POs may be manually closed
- The PO must have at least one associated expense
- Requires the user to have appropriate permissions

**Automatic Closure:**

- `One-Time` POs: Closed when their single expense is committed
- `Recurring` POs: Closed when all expected expenses have been committed
- `Cumulative` POs: Closed when expenses total reaches the PO total

When closed:

- The `closer` property records who closed it (or empty if system-closed)
- The `closed` timestamp is set
- The `closed_by_system` flag indicates if closure was automatic
- The status changes to `Closed`

## API Endpoints

Implemented in `app/routes/purchase_orders.go` and registered in `app/routes/routes.go`.

### GET /api/purchase_orders/approvers

Returns a list of users who can first-approve a purchase order with the given parameters.

**Query Parameters:** `division` (required), `amount` (required, number), `kind` (optional, expenditure_kinds ID; defaults to standard), `has_job` (optional, boolean; defaults to false), `type`, `start_date`, `end_date`, `frequency`

**Authorization:** Requires authenticated user.

**Behavior:**

- If the caller is themselves a qualified first approver for the division, returns an empty list (the UI auto-sets the approver to the caller)
- Otherwise, returns a list of qualified first approvers for the division
- For Recurring POs, `type=Recurring` triggers server-side calculation of approval_total from `start_date`, `end_date`, and `frequency`

### GET /api/purchase_orders/second_approvers

Returns second-approval candidates plus explicit second-approval status metadata.

**Query Parameters:** Same as `GET /api/purchase_orders/approvers`

**Authorization:** Requires authenticated user.

**Response:**

- `approvers`: array of qualified second-approver candidates
- `meta.second_approval_required`: whether second approval is required for the evaluated amount
- `meta.requester_qualifies`: whether the caller is qualified to second-approve this PO context
- `meta.reason_code`: machine-readable explanation for current second-approval status
- `meta.reason_message`: user-facing explanation for why candidates are or are not available
- `meta.evaluated_amount`: amount used to evaluate second approval (including recurring recalculation)
- `meta.second_approval_threshold`: lowest threshold that triggers second approval
- `meta.tier_ceiling`: applicable approval ceiling for the evaluated amount
- `meta.limit_column`: `po_approver_props` column used to evaluate second-approver eligibility
- `meta.status`: one of:
  - `not_required`
  - `requester_qualifies`
  - `candidates_available`
  - `required_no_candidates`

**Behavior:**

- Uses the kind-specific `po_approver_props` limit column (determined by `kind` and `has_job`) to filter eligible second approvers
- If amount is below the lowest threshold, returns `meta.status = not_required`
- If caller is qualified second approver, returns `meta.status = requester_qualifies`
- If second approval is required and candidates exist, returns `meta.status = candidates_available` and a non-empty `approvers` list
- If second approval is required and no candidates exist, returns `meta.status = required_no_candidates` with an empty `approvers` list

### Future Option: POS List Diagnostics (recommended approach)

To show second-approval diagnostics in `pos/list` without N+1 client requests, prefer a backend-backed bulk path:

- Add a list-safe diagnostics projection (either new API endpoint or computed fields in `purchase_orders_augmented`) that includes:
  - `second_approval_status`
  - `second_approval_reason_code`
  - `second_approval_reason_message`
- Compute these fields server-side using the same logic as `/api/purchase_orders/second_approvers`.
- Drive list-page red `!` indicators and hover/expand details from that single list payload.

This avoids making one `/second_approvers` request per row in the list UI.

### POST /api/purchase_orders/:id/approve

Approves the purchase order with the given ID.

**Authorization:** Caller must be a qualified first approver OR qualified second approver for the PO's division, kind, and approval_total.

**Behavior:**

- All operations performed in a transaction
- Fails if status is not `Unapproved` or if already rejected
- If second approval is required and first-approval caller is not second-qualified, first approval is blocked when no second-approval owner is assignable (`priority_second_approver` missing and no eligible candidates)
- If caller is qualified as first approver and PO is not yet first-approved, sets `approved` timestamp and **overwrites** `approver` to caller (regardless of who was originally selected)
- If caller is qualified as second approver and PO requires second approval, sets `second_approval` timestamp and `second_approver` to caller
- A single call can perform both approvals if the caller is qualified for both
- Upon full approval, generates `po_number` in format `YYMM-NNNN` and sets status to `Active`

### POST /api/purchase_orders/:id/reject

Rejects the purchase order with the given ID.

**Authorization:** Caller must be a qualified first approver OR qualified second approver.

**Payload:** `{ "rejection_reason": "string" }` (minimum 5 characters)

**Behavior:**

- Fails if status is not `Unapproved` or if already rejected
- Sets `rejection_reason`, `rejector`, and `rejected` timestamp
- Status remains `Unapproved`

### POST /api/purchase_orders/:id/cancel

Cancels the purchase order with the given ID.

**Authorization:** Caller must have the `payables_admin` claim.

**Behavior:**

- Fails if status is not `Active`
- Fails if PO has any associated expenses
- Sets `canceller` and `cancelled` timestamp
- Status changes to `Cancelled`

### POST /api/purchase_orders/:id/close

Manually closes the purchase order with the given ID.

**Authorization:** Requires appropriate permissions.

**Behavior:**

- Only `Recurring` and `Cumulative` POs may be manually closed
- PO must have at least one associated expense
- Sets `closer` and `closed` timestamp
- Status changes to `Closed`

## PO Number Format

**Parent POs:** `YYMM-NNNN`

- YY: 2-digit year
- MM: 2-digit month
- NNNN: Sequential 4-digit number (0001-5999)

**Child POs:** `YYMM-NNNN-XX`

- Same prefix as parent
- XX: Sequential 2-digit suffix (01-99)

**Examples:**

- `2501-0001` (first PO created in January 2025)
- `2501-0001-01` (first child of that PO)

## Pocketbase Collection Schema (purchase_orders)

| Field                    | Type                         | Description                                                                               |
| ------------------------ | ---------------------------- | ----------------------------------------------------------------------------------------- |
| po_number                | string                       | Format `YYMM-NNNN` or `YYMM-NNNN-XX`, unique, required if Active/Cancelled/Closed         |
| status                   | enum                         | `Unapproved`, `Active`, `Cancelled`, `Closed`                                             |
| uid                      | relation → users             | Creator of the PO, required                                                               |
| type                     | enum                         | `One-Time`, `Cumulative`, `Recurring`                                                     |
| kind                     | relation → expenditure_kinds | Expenditure kind, required; determines which approval limit column is used                |
| date                     | string                       | Start date, YYYY-MM-DD format, required                                                   |
| end_date                 | string                       | End date for Recurring POs, YYYY-MM-DD format                                             |
| frequency                | enum                         | `Weekly`, `Biweekly`, `Monthly` (required for Recurring)                                  |
| division                 | relation → divisions         | Required                                                                                  |
| description              | string                       | Minimum 5 characters                                                                      |
| total                    | number                       | Single payment/expense amount, required                                                   |
| approval_total           | number                       | Calculated total for approval tier determination                                          |
| payment_type             | enum                         | `OnAccount`, `Expense`, `CorporateCreditCard`                                             |
| vendor                   | relation → vendors           | Required                                                                                  |
| job                      | relation → jobs              | Optional job reference                                                                    |
| category                 | relation → categories        | Optional category                                                                         |
| branch                   | relation → branches          | Set from job or user default                                                              |
| attachment               | file                         | Optional attachment                                                                       |
| approver                 | relation → users             | Required; selected by creator at creation, but overwritten with whoever actually approves |
| approved                 | datetime                     | First approval timestamp                                                                  |
| second_approver          | relation → users             | Set during second approval                                                                |
| second_approval          | datetime                     | Second approval timestamp                                                                 |
| priority_second_approver | relation → users             | Creator-designated second approver (optional, only when second approval required)         |
| rejector                 | relation → users             | Set if rejected                                                                           |
| rejected                 | datetime                     | Rejection timestamp                                                                       |
| rejection_reason         | string                       | Minimum 5 characters if rejected                                                          |
| canceller                | relation → users             | Set if cancelled                                                                          |
| cancelled                | datetime                     | Cancellation timestamp                                                                    |
| closer                   | relation → users             | Set if manually closed                                                                    |
| closed                   | datetime                     | Closure timestamp                                                                         |
| closed_by_system         | boolean                      | True if automatically closed                                                              |
| parent_po                | relation → purchase_orders   | Parent PO for child POs                                                                   |

## Pocketbase Collection Schema (expenditure_kinds)

| Field       | Type   | Description                                         |
| ----------- | ------ | --------------------------------------------------- |
| id          | string | PocketBase record ID                                |
| name        | string | Semantic key (`standard`, `sponsorship`, etc.)      |
| description | string | Human-readable description of the kind              |
| ui_order    | number | Integer controlling display order in the UI         |
| en_ui_label | string | English display label shown in the UI (min 3 chars) |

## po_approver_props Limit Columns

Each `po_approver_props` record has the following limit columns that control second-approval eligibility per kind:

| Column                 | Used for kind      | When          |
| ---------------------- | ------------------ | ------------- |
| `max_amount`           | `standard`         | PO has no job |
| `project_max`          | `standard`         | PO has a job  |
| `sponsorship_max`      | `sponsorship`      | Always        |
| `staff_and_social_max` | `staff_and_social` | Always        |
| `media_and_event_max`  | `media_and_event`  | Always        |
| `computer_max`         | `computer`         | Always        |
