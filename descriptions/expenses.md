# The expenses system

## Summary of the Expenses System

The expenses system depends on the purchase_orders system and the job categories system. So these must be implemented first.

The user can create, edit, delete, and submit expenses with the expenses system. Additionally, the user can view a list of all their expenses. Each expense has an approver which is set by a hook when the expense is created or updated and is based on the manager of the user who created the expense. This information comes from the user's profile. The approver can approve or reject expenses submitted by their direct reports. When an expense is approved, the approved property is set to the current timestamp, otherwise the expense record's approved value is an empty string. When an expense is rejected, the rejected property is set to the current timestamp, the rejector property is set to the id of the user who rejected the expense, and a rejection reason is provided. Expenses must be committed by a user with the commit_expense claim prior to being paid out.

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
- CorporateCreditCard under $500
- Expense under $500

### Creating an expense via a purchase order

The following types of expenses can only be created if a purchase order exists:

- Corporate Credit Card $500 or greater
- Expense $500 or greater

## Pocketbase Collection Schema (expenses)

- date (string, required, YYYY-MM-DD)
- pay_period_ending (string, required, YYYY-MM-DD, saturday)
- uid (references users collection, required, the user who created the expense)
- payment_type (enum, required) [Allowance, Expense, FuelCard, Mileage, CorporateCreditCard, PersonalReimbursement] **DEPRECATE FuelOnAccount (do not replace), Meals (replace with Allowance)**
- division (references divisions collection, required)
- job (references jobs collection)
- category (references the categories collection, category's job must match the expense's job)
- approver (references users collection, required)
- approved (datetime)
- allowance_types (Multiple Enum OR JSON, required if payment_type is Allowance, else empty) [Lodging, Lunch, Dinner, Breakfast]
- total (number, required)
- submitted (datetime)
- committer (references users collection)
- committed (datetime)
- committed_week_ending (string, required, YYYY-MM-DD, saturday)
- description (string, minimum 5 characters)
- rejected (boolean)
- rejector (references users collection)
- rejection_reason (string, minimum 5 characters)
- distance (number)
- vendor_name (string)
- attachment (file)
- cc_last_4_digits (string)
- purchase_order (references purchase_orders collection)
- category (references the categories set of the corresponding job)

## The expense entry/edit page

## The expense list page
