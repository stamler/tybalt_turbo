# The expenses system

## Summary of the Expenses System

The expenses system depends on the purchase_orders system and the job categories system. So these must be implemented first.

The user can create, edit, delete, and submit expenses with the expenses system. Additionally, the user can view a list of all their expenses. Each expense has an approver which is set by a hook when the expense is created or updated and is based on the manager of the user who created the expense. This information comes from the user's profile. The approver can approve or reject expenses submitted by their direct reports. Users with the `commit` claim can also reject expenses. When an expense is approved, the approved property is set to the current timestamp, otherwise the expense record's approved value is an empty string. When an expense is rejected, the rejected property is set to the current timestamp, the rejector property is set to the id of the user who rejected the expense, and a rejection reason is provided. Expenses must be committed by a user with the `commit` claim prior to being paid out.

There are two paths to creating expenses, the first is directly through the new expense page, the second is via submitting an expense against an existing purchase order. There is no facility to specify a PO number when creating an expense from scratch. If a PO number is required, the user must submit an expense against an existing purchase order.

### Creating an expense from scratch

Remember, a purchase order cannot be specified when creating an expense from scratch.
The following types of expense can only be created from scratch:

- Allowance
  - Lodging
  - Meals (Breakfast, Lunch, Dinner)
- FuelCard
- Mileage
- PersonalReimbursement
- CorporateCreditCard under the no-PO limit (default $100, configurable via `app_config`)
- Expense under the no-PO limit (default $100, configurable via `app_config`)
- OnAccount under the no-PO limit, or any amount with `payables_admin` claim

### Creating an expense via a purchase order

The following types of expenses can only be created if a purchase order exists:

- CorporateCreditCard at or above the no-PO limit (default $100)
- Expense at or above the no-PO limit (default $100)
- OnAccount at or above the no-PO limit (without `payables_admin` claim)
- Any expense with a `job` (except `Mileage`, `FuelCard`, `PersonalReimbursement`, `Allowance`)

The no-PO limit is configurable via `app_config` under `"expenses"` domain (`no_po_expense_limit`). Default is `$100.00`. See `purchase_orders.md` for the full PO-requirement rules.

## Pocketbase Collection Schema (expenses)

Note: "required" below means enforced in hooks/validation, not necessarily at the PocketBase schema level.

- date (string, required, YYYY-MM-DD)
- pay_period_ending (string, required, YYYY-MM-DD, saturday, derived from date)
- uid (references users collection, required, the user who created the expense)
- payment_type (enum, required) [OnAccount, Expense, CorporateCreditCard, Allowance, FuelCard, Mileage, PersonalReimbursement]
- kind (relation -> expenditure_kinds, inherited from PO when present, otherwise defaulted by job presence)
- division (references divisions collection, required)
- job (references jobs collection)
- category (references the categories collection, category's job must match the expense's job)
- branch (relation -> branches, derived from job or user default)
- approver (references users collection, set by hook from user's manager)
- approved (datetime)
- allowance_types (Multiple Enum OR JSON, required if payment_type is Allowance, else empty) [Lodging, Lunch, Dinner, Breakfast]
- total (number, required for non-Allowance expenses)
- submitted (bool)
- committer (references users collection)
- committed (datetime)
- committed_week_ending (string, YYYY-MM-DD, saturday, derived from committed by commit handler)
- description (string, minimum 5 characters for non-Allowance expenses, enforced in hooks)
- rejected (datetime)
- rejector (references users collection)
- rejection_reason (string, minimum 5 characters)
- distance (number)
- vendor (relation -> vendors collection)
- attachment (file)
- cc_last_4_digits (string)
- purchase_order (references purchase_orders collection)

## The expense entry/edit page

## The expense list page

Primary list UI is implemented in `ui/src/lib/components/ExpensesList.svelte` and used by routes such as `/expenses/list` and `/expenses/pending`.

Current action placement:

- Owner list actions:
  - `Edit` and `Submit` for draft (unsubmitted) expenses
  - `Recall` for submitted-not-approved or rejected expenses (when not committed)
  - `Delete` for unsubmitted (draft) expenses only
- Owner exception for approval on list:
  - owner may see `Approve` on list only when they are also the assigned approver and have `tapr` claim
- Review actions are details-first for non-owners:
  - `Reject`/`Commit` controls are intentionally hidden in list view to encourage opening the details page first

## Schema Comparison: purchase_orders vs expenses

| Property                 | Purchase Orders | Expenses |
| ------------------------ | --------------- | -------- |
| id                       | \*              | \*       |
| po_number                | \*              |          |
| status                   | \*              |          |
| type                     | \*              |          |
| end_date                 | \*              |          |
| frequency                | \*              |          |
| approval_total           | \*              |          |
| second_approver          | \*              |          |
| second_approval          | \*              |          |
| priority_second_approver | \*              |          |
| canceller                | \*              |          |
| cancelled                | \*              |          |
| closer                   | \*              |          |
| closed                   | \*              |          |
| closed_by_system         | \*              |          |
| parent_po                | \*              |          |
| uid                      | \*              | \*       |
| date                     | \*              | \*       |
| division                 | \*              | \*       |
| description              | \*              | \*       |
| total                    | \*              | \*       |
| payment_type             | \*              | \*       |
| kind                     | \*              | \*       |
| vendor                   | \*              | \*       |
| branch                   | \*              | \*       |
| attachment               | \*              | \*       |
| rejector                 | \*              | \*       |
| rejected                 | \*              | \*       |
| rejection_reason         | \*              | \*       |
| approver                 | \*              | \*       |
| approved                 | \*              | \*       |
| job                      | \*              | \*       |
| category                 | \*              | \*       |
| pay_period_ending        |                 | \*       |
| allowance_types          |                 | \*       |
| submitted                |                 | \*       |
| committer                |                 | \*       |
| committed                |                 | \*       |
| committed_week_ending    |                 | \*       |
| distance                 |                 | \*       |
| cc_last_4_digits         |                 | \*       |
| purchase_order (ref)     |                 | \*       |

## The expense details page

Primary details UI is `/expenses/{id}/details` (`ui/src/routes/expenses/[eid]/details/+page.svelte`).

Current action placement:

- Owner actions:
  - `Edit`, `Submit`, `Recall`, `Delete` (subject to submitted/approved/rejected/committed state)
- Approver actions:
  - `Approve` (assigned approver, while submitted and not yet approved/rejected/committed)
  - `Reject` (assigned approver or users with `commit` claim, while submitted and not yet rejected/committed)
- Committer actions:
  - `Commit` requires `commit` claim (or `showAllUi` override)
  - `Reject` is also available to committer-role users while the record is submitted and not yet rejected/committed
